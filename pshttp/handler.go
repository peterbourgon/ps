package pshttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/peterbourgon/eventsource"
	"github.com/peterbourgon/ps"
)

type handler[T any] struct {
	http.Handler
	broker *ps.Broker[T]
	encode EncodeFunc[T]
	decode DecodeFunc[T]
	logger *log.Logger
}

// NewDefaultHandler calls NewHandler with the default [EncodeJSON] and [DecodeJSON]
// functions, and logging to [io.Discard].
func NewDefaultHandler[T any](broker *ps.Broker[T]) http.Handler {
	return NewHandler(broker, EncodeJSON[T], DecodeJSON[T], io.Discard)
}

// NewHandler constructs a new [http.Handler] wrapping the provided [ps.Broker].
func NewHandler[T any](broker *ps.Broker[T], encode EncodeFunc[T], decode DecodeFunc[T], logs io.Writer) http.Handler {
	h := &handler[T]{
		broker: broker,
		encode: encode,
		decode: decode,
		logger: log.New(logs, "pshttp.Handler: ", log.Lmsgprefix),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", h.handlePublish)
	mux.HandleFunc("GET /", h.handleSubscribe)
	h.Handler = mux

	return h
}

func (h *handler[T]) handlePublish(w http.ResponseWriter, r *http.Request) {
	var v T
	if err := h.decode(r.Body, &v); err != nil {
		respondJSON(w, http.StatusBadRequest, err)
		return
	}
	stats := h.broker.Publish(v)
	respondJSON(w, http.StatusOK, stats)
}

func (h *handler[T]) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if !requestExplicitlyAccepts(r, "text/event-stream") {
		respondJSON(w, http.StatusBadRequest, fmt.Errorf("request must accept: text/event-stream"))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondJSON(w, http.StatusInternalServerError, fmt.Errorf("response writer must support flushing"))
		return
	}

	var (
		ctx       = r.Context()
		logger    = log.New(h.logger.Writer(), h.logger.Prefix()+fmt.Sprintf("%s: ", r.RemoteAddr), h.logger.Flags())
		buffer    = parseDefault(r.URL.Query().Get("buffer"), strconv.Atoi, 100)
		heartbeat = parseDefault(r.URL.Query().Get("heartbeat"), parseDurationMinMax(1*time.Second, 60*time.Second), 3*time.Second)
		c         = make(chan T, buffer)
	)

	if err := h.broker.SubscribeAll(c); err != nil {
		respondJSON(w, http.StatusInternalServerError, fmt.Errorf("subscribe: %w", err))
		return
	}
	defer func() {
		stats, err := h.broker.Unsubscribe(c)
		logger.Printf("unsubscribe: %v (err: %v)", stats, err)
	}()

	logger.Printf("subscribe: buffer=%d heartbeat=%v", buffer, heartbeat)

	heartbeats := time.NewTicker(heartbeat)
	defer heartbeats.Stop()

	eventsource.Handler(func(_ string, enc *eventsource.Encoder, stop <-chan bool) {
		err := func() error {
			var buf bytes.Buffer
			for {
				select {
				case v := <-c:
					buf.Reset()
					if err := h.encode(v, &buf); err != nil {
						return fmt.Errorf("encode value: %w", err)
					}
					if err := enc.Encode(eventsource.Event{
						Type: EventTypeData,
						Data: buf.Bytes(),
					}); err != nil {
						return fmt.Errorf("encode data event: %w", err)
					}
					flusher.Flush()

				case ts := <-heartbeats.C:
					ev := HeartbeatEvent{
						Timestamp: ts,
					}
					if stats, err := h.broker.Stats(c); err == nil {
						ev.Stats = stats
					} else {
						ev.Error = err.Error()
					}
					data, err := json.Marshal(ev)
					if err != nil {
						return fmt.Errorf("marshal heartbeat event: %w", err)
					}
					if err := enc.Encode(eventsource.Event{
						Type: EventTypeHeartbeat,
						Data: data,
					}); err != nil {
						return fmt.Errorf("encode heartbeat event: %w", err)
					}
					flusher.Flush()

				case <-stop:
					return fmt.Errorf("stop signaled")

				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}()
		logger.Printf("handler exiting (err: %v)", err)
	}).ServeHTTP(w, r)
}
