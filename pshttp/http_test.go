package pshttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bernerdschaefer/eventsource"
	"github.com/peterbourgon/ps"
	"github.com/peterbourgon/ps/pshttp"
)

func TestHTTP(t *testing.T) {
	t.Parallel()

	t.Run("manual", func(t *testing.T) {
		broker := ps.NewBroker[int]()

		handler := &pshttp.Handler[int]{
			Broker: broker,
			Encode: func(v int) ([]byte, error) { return []byte(strconv.Itoa(v)), nil },
			Decode: func(d []byte) (int, error) { return strconv.Atoi(string(d)) },
		}

		server := httptest.NewServer(handler)
		t.Cleanup(func() { server.Close() })

		req, _ := http.NewRequest("GET", server.URL, nil)
		retry := 10 * time.Millisecond
		es := eventsource.New(req, retry)

		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				ev, err := es.Read()
				switch {
				case err == nil:
					t.Logf("Read: OK: %s", prettyPrint(ev))
					continue
				case errors.Is(err, eventsource.ErrClosed):
					t.Logf("Read: closed (done)")
					return
				case err != nil:
					t.Errorf("Read: error: %v", err)
					return
				}
			}
		}()

		for i := 0; i < 32; i++ {
			stats := broker.Publish(i)
			t.Logf("broker.Publish(%v): %s", i, stats)
			if i == 9 {
				server.CloseClientConnections()
			}
			time.Sleep(2 * retry)
		}

		t.Logf("es.Close")
		es.Close()
		t.Logf("<-done")
		<-done
		t.Logf("finished")
	})

	t.Run("client", func(t *testing.T) {
		broker := ps.NewBroker[int]()

		handler := &pshttp.Handler[int]{
			Broker: broker,
			Encode: func(v int) ([]byte, error) { return json.Marshal(v) },
			Decode: func(d []byte) (int, error) { return strconv.Atoi(string(d)) },
		}

		server := httptest.NewServer(handler)
		t.Cleanup(func() { server.Close() })

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		valc := make(chan int)
		printc := make(chan struct{})
		go func() {
			defer close(printc)
			for i := range valc {
				t.Logf("got: %v", i)
			}
		}()

		subscribec := make(chan error, 1)
		go func() {
			req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
			dec := func(data []byte) (int, error) { return strconv.Atoi(string(data)) }
			subscribec <- pshttp.Subscribe(req, dec, valc)
			close(valc)
		}()

		for i := 0; i < 3; i++ {
			t.Logf("Publish(%d): %v", i, broker.Publish(i))
		}

		t.Logf("cancel")
		cancel()
		t.Logf("subscribec: %v", <-subscribec)
		<-printc
	})
}

func prettyPrint(ev eventsource.Event) string {
	return "[" + strings.Join([]string{
		fmt.Sprintf("Type=%q", ev.Type),
		fmt.Sprintf("ID=%q", ev.ID),
		fmt.Sprintf("Data=%q", string(ev.Data)),
		fmt.Sprintf("ResetID=%v", ev.ResetID),
		fmt.Sprintf("Retry=%q", ev.Retry),
	}, " ") + "]"
}
