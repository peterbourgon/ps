package ps

import (
	"slices"
	"sync"
)

// Broker is a pub/sub coördination point for values of type T. See the Publish,
// Subscribe, and Unsubscribe methods for more information.
type Broker[T any] struct {
	mtx  sync.Mutex
	subs []*subscriber[T]
}

// NewBroker returns a new broker for values of type T.
func NewBroker[T any]() *Broker[T] {
	return &Broker[T]{
		//
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

	for _, s := range b.subs {
		if !s.allow(v) {
			s.stats.Skips++
			stats.Skips++
		} else {
			select {
			case s.c <- v:
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

// Subscribe adds c to the broker, and forwards every published value that
// passes the allow func to c.
func (b *Broker[T]) Subscribe(c chan<- T, allow func(T) bool) error {
	if allow == nil {
		allow = func(T) bool { return true }
	}

	b.mtx.Lock()
	defer b.mtx.Unlock()

	for _, s := range b.subs {
		if s.c == c {
			return ErrAlreadySubscribed
		}
	}

	s := &subscriber[T]{
		c:     c,
		allow: allow,
	}

	b.subs = append(b.subs, s)

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

	var target *subscriber[T]
	for _, s := range b.subs {
		if s.c == c {
			target = s
			break
		}
	}

	if target == nil {
		return Stats{}, ErrNotSubscribed
	}

	b.subs = slices.DeleteFunc(b.subs, func(s *subscriber[T]) bool {
		return s == target
	})

	return target.stats, nil
}

// Stats returns current statistics for the subscription represented by c.
func (b *Broker[T]) Stats(c chan<- T) (Stats, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	for _, s := range b.subs {
		if s.c == c {
			return s.stats, nil
		}
	}

	return Stats{}, ErrNotSubscribed
}

// ActiveSubscribers returns statistics for every active subscriber.
func (b *Broker[T]) ActiveSubscribers() []Stats {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	res := make([]Stats, len(b.subs))
	for i := range b.subs {
		res[i] = b.subs[i].stats
	}

	return res
}

type subscriber[T any] struct {
	c     chan<- T
	allow func(T) bool
	stats Stats
}
