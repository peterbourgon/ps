package pshttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/peterbourgon/eventsource"
	"github.com/peterbourgon/ps"
)

type Client[T any] struct {
	client *http.Client
	uri    string
	encode EncodeFunc[T]
	decode DecodeFunc[T]
}

func NewDefaultClient[T any](uri string) (*Client[T], error) {
	return NewClient(http.DefaultClient, uri, Encode[T], Decode[T])
}

func NewClient[T any](client *http.Client, uri string, encode EncodeFunc[T], decode DecodeFunc[T]) (*Client[T], error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid URI: %w", err)
	}

	uri = u.String()

	return &Client[T]{
		client: client,
		uri:    uri,
		encode: encode,
		decode: decode,
	}, nil
}

func (c *Client[T]) Publish(ctx context.Context, v T) (ps.Stats, error) {
	var buf bytes.Buffer
	if err := c.encode(v, &buf); err != nil {
		return ps.Stats{}, fmt.Errorf("encode value: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.uri, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return ps.Stats{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return ps.Stats{}, fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return ps.Stats{}, fmt.Errorf("invalid response (%s)", resp.Status)
	}

	var stats ps.Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return ps.Stats{}, fmt.Errorf("decode response stats: %w", err)
	}

	return stats, nil
}

func (c *Client[T]) Subscribe(ctx context.Context, ch chan<- T, retry time.Duration) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.uri, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	es := eventsource.New(req, retry)
	defer es.Close()

	for {
		ev, err := es.Read()
		if err != nil {
			return fmt.Errorf("read event: %w", err)
		}
		if ev.Type != EventTypeData {
			continue // TODO
		}

		var v T
		if err := c.decode(bytes.NewReader(ev.Data), &v); err != nil {
			return fmt.Errorf("decode event: %w", err)
		}

		select {
		case ch <- v:
			// good
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
