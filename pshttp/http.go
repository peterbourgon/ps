package pshttp

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/bernerdschaefer/eventsource"
	"github.com/peterbourgon/ps"
)

// Handler provides an HTTP interface to a pub/sub broker.
type Handler[T any] struct {
	Encode func(T) ([]byte, error)          // required
	Decode func([]byte) (T, error)          // required
	Broker *ps.Broker[T]                    // optional
	Filter func(*http.Request) func(T) bool // optional
	Logger io.Writer                        // optional

	once   sync.Once
	fatal  func(http.ResponseWriter)
	logger *log.Logger
}

const (
	EventTypeData      = "data"
	EventTypeHeartbeat = "heartbeat"
)

// ServeHTTP publishes on POST and subscribes (via SSE) on GET.
func (h *Handler[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(func() {
		if h.Encode == nil {
			h.fatal = func(w http.ResponseWriter) {
				http.Error(w, "invalid handler: no encode", http.StatusInternalServerError)
			}
			return
		}

		if h.Decode == nil {
			h.fatal = func(w http.ResponseWriter) {
				http.Error(w, "invalid handler: no decode", http.StatusInternalServerError)
			}
			return
		}

		if h.Broker == nil {
			h.Broker = ps.NewBroker[T]()
		}

		if h.Filter == nil {
			h.Filter = func(*http.Request) func(T) bool {
				return func(T) bool { return true }
			}
		}

		if h.Logger == nil {
			h.Logger = io.Discard
		}

		h.logger = log.New(h.Logger, "", 0)
	})

	switch {
	case h.fatal != nil:
		h.fatal(w)
	case r.Method == "POST":
		h.HandlePublish(w, r)
	case r.Method == "GET":
		h.HandleSubscribe(w, r)
	default:
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
	}
}

func (h *Handler[T]) HandlePublish(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Errorf("read body: %w", err).Error(), http.StatusBadRequest)
		return
	}

	v, err := h.Decode(body)
	if err != nil {
		http.Error(w, fmt.Errorf("decode body: %w", err).Error(), http.StatusBadRequest)
		return
	}

	stats := h.Broker.Publish(v)

	h.logger.Printf("publish: RemoteAddr=%s payload=%dB stats=%s", r.RemoteAddr, len(body), stats)
}

func (h *Handler[T]) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	if !requestExplicitlyAccepts(r, "text/event-stream") {
		http.Error(w, "subscribe requests must Accept: text/event-stream", http.StatusUnsupportedMediaType)
		return
	}

	valc := make(chan T)
	if err := h.Broker.Subscribe(valc, h.Filter(r)); err != nil {
		http.Error(w, fmt.Errorf("subscribe: %w", err).Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Printf("subscribe: RemoteAddr=%s", r.RemoteAddr)

	defer func() {
		if stats, err := h.Broker.Unsubscribe(valc); err != nil {
			h.logger.Printf("unsubscribe: error: %v", err)
		} else {
			h.logger.Printf("unsubscribe: OK: %s", stats)
		}
	}()

	eventsource.Handler(func(lastID string, enc *eventsource.Encoder, stop <-chan bool) {
		heartbeat := time.NewTicker(3 * time.Second)
		defer heartbeat.Stop()

		var (
			recvValueCount uint64
			sendValueCount uint64
			heartbeatCount uint64
		)
		defer func() {
			log.Printf("recv %d, send %d, heartbeat %d", recvValueCount, sendValueCount, heartbeatCount)
		}()

		for {
			select {
			case <-stop:
				log.Printf("eventsource: stop")
				return

			case v := <-valc:
				recvValueCount++
				data, err := h.Encode(v)
				if err != nil {
					log.Printf("eventsource: encode data value: %v", err)
					continue
				}
				if err := enc.Encode(eventsource.Event{
					Type: EventTypeData,
					Data: data,
				}); err != nil {
					log.Printf("eventsource: encode data event: %v", err)
					continue
				}
				sendValueCount++

			case ts := <-heartbeat.C:
				if err := enc.Encode(eventsource.Event{
					Type: EventTypeHeartbeat,
					Data: []byte(fmt.Sprintf(`{"ts":"%s"}`, ts.UTC().Format(time.RFC3339))),
				}); err != nil {
					log.Printf("eventsource: encode heartbeat event: %v", err)
					continue
				}
				heartbeatCount++

			case <-r.Context().Done():
				log.Printf("context done (%v)", r.Context().Err())
				return
			}
		}
	}).ServeHTTP(w, r)
}

// Subscribe to the broker represented by req, decoding event payloads via dec,
// and sending values to dst. Runs until a fatal error is encountered, or the
// request context is canceled.
func Subscribe[T any](req *http.Request, dec func([]byte) (T, error), dst chan<- T) error {
	ctx := req.Context()

	es := eventsource.New(req, time.Second)
	defer es.Close()

	{
		bugfix := make(chan struct{})
		go func() { <-ctx.Done(); es.Close(); close(bugfix) }()
		defer func() { <-bugfix }()
	}

	for {
		event, err := es.Read()
		if err != nil {
			return fmt.Errorf("read event: %w", err)
		}

		switch event.Type {
		case EventTypeHeartbeat:
			continue

		case EventTypeData:
			val, err := dec(event.Data)
			if err != nil {
				return fmt.Errorf("decode data: %w", err)
			}
			select {
			case dst <- val:
				continue
			case <-ctx.Done():
				return ctx.Err()
			}

		default:
			continue

		}
	}
}
