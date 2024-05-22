package ps

import (
	"sync"
)

type Broker[T any] struct {
	mtx  sync.Mutex
	subs map[chan<- T]*subscriber[T]
}

// NewBroker returns a new broker for values of type T.
func NewBroker[T any]() *Broker[T] {
	return &Broker[T]{
		subs: map[chan<- T]*subscriber[T]{},
	}
}

// Publish the given value to all active and matching subscribers. Each send is
// non-blocking, so values are dropped when subscribers aren't keeping up. Also,
// values are sent directly, so be mindful of copy costs and semantics. Returned
// stats reflect the outcome for all active subscribers at the time of the
// publish.
func (b *Broker[T]) Publish(v T) Stats {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	var stats Stats

	for c, s := range b.subs {
		if !s.allow(v) {
			s.stats.Skips++
			stats.Skips++
		} else {
			select {
			case c <- v:
				s.stats.Sends++
				stats.Sends++
			default:
				s.stats.Drops++
				stats.Drops++
			}
		}
	}

	return stats
}

// Subscribe adds c to the broker, and forwards every published value that passes the allow func to c.
func (b *Broker[T]) Subscribe(c chan<- T, allow func(T) bool) error {
	if allow == nil {
		allow = func(T) bool { return true }
	}

	b.mtx.Lock()
	defer b.mtx.Unlock()

	if _, ok := b.subs[c]; ok {
		return ErrAlreadySubscribed
	}

	b.subs[c] = &subscriber[T]{
		allow: allow,
	}

	return nil
}

// SubscribeAll subscribes to every published value.
func (b *Broker[T]) SubscribeAll(c chan<- T) error {
	return b.Subscribe(c, nil)
}

// Unsubscribe removes the given channel from the broker.
func (b *Broker[T]) Unsubscribe(c chan<- T) (Stats, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	s, ok := b.subs[c]
	if !ok {
		return Stats{}, ErrNotSubscribed
	}

	delete(b.subs, c)

	return s.stats, nil
}

// Stats returns current statistics for the subscription represented by c.
func (b *Broker[T]) Stats(c chan<- T) (Stats, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	s, ok := b.subs[c]
	if !ok {
		return Stats{}, ErrNotSubscribed
	}

	return s.stats, nil
}

type subscriber[T any] struct {
	allow func(T) bool
	stats Stats
}
