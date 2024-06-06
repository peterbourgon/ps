package ps_test

import (
	"testing"

	"github.com/peterbourgon/ps"
)

func TestBasics(t *testing.T) {
	t.Parallel()

	t.Run("no subscribers", func(t *testing.T) {
		broker := ps.NewBroker[int]()

		compareStats(t, broker.Publish(1), ps.Stats{})
		compareStats(t, broker.Publish(2), ps.Stats{})
		compareStats(t, broker.Publish(3), ps.Stats{})
	})

	t.Run("skip subscriber", func(t *testing.T) {
		broker := ps.NewBroker[int]()

		c := make(chan int)
		broker.Subscribe(c, func(int) bool { return false })

		compareStats(t, broker.Publish(1), ps.Stats{Skips: 1})
		compareStats(t, broker.Publish(2), ps.Stats{Skips: 1})
		compareStats(t, broker.Publish(3), ps.Stats{Skips: 1})

		stats, err := broker.Unsubscribe(c)
		requireNoError(t, err)
		compareStats(t, stats, ps.Stats{Skips: 3})
	})

	t.Run("slow subscriber", func(t *testing.T) {
		broker := ps.NewBroker[int]()

		c1, c3 := make(chan int, 1), make(chan int, 3)
		broker.Subscribe(c1, func(int) bool { return true })
		broker.Subscribe(c3, func(int) bool { return true })

		compareStats(t, broker.Publish(1), ps.Stats{Sends: 2})
		compareStats(t, broker.Publish(2), ps.Stats{Sends: 1, Drops: 1})
		compareStats(t, broker.Publish(3), ps.Stats{Sends: 1, Drops: 1})
		compareStats(t, broker.Publish(4), ps.Stats{Sends: 0, Drops: 2})
		compareStats(t, broker.Publish(5), ps.Stats{Sends: 0, Drops: 2})

		expectEqual(t, 1, <-c1)
		expectEqual(t, 1, <-c3)

		compareStats(t, broker.Publish(6), ps.Stats{Sends: 2, Drops: 0})
		compareStats(t, broker.Publish(7), ps.Stats{Sends: 0, Drops: 2})

		c1s, err := broker.Unsubscribe(c1)
		requireNoError(t, err)
		compareStats(t, c1s, ps.Stats{Skips: 0, Sends: 2, Drops: 5})

		c3s, err := broker.Unsubscribe(c3)
		requireNoError(t, err)
		compareStats(t, c3s, ps.Stats{Skips: 0, Sends: 4, Drops: 3})
	})

	t.Run("subscriber stats", func(t *testing.T) {
		broker := ps.NewBroker[int]()

		mod2, mod3 := make(chan int, 100), make(chan int, 100)
		broker.Subscribe(mod2, func(i int) bool { return i%2 == 0 })
		broker.Subscribe(mod3, func(i int) bool { return i%3 == 0 })

		compareStats(t, broker.Publish(1), ps.Stats{Skips: 2, Sends: 0})
		compareStats(t, broker.Publish(2), ps.Stats{Skips: 1, Sends: 1})
		compareStats(t, broker.Publish(3), ps.Stats{Skips: 1, Sends: 1})
		compareStats(t, broker.Publish(4), ps.Stats{Skips: 1, Sends: 1})
		compareStats(t, broker.Publish(5), ps.Stats{Skips: 2, Sends: 0})
		compareStats(t, broker.Publish(6), ps.Stats{Skips: 0, Sends: 2})

		mod2stats, err := broker.Stats(mod2)
		requireNoError(t, err)
		compareStats(t, mod2stats, ps.Stats{Skips: 3, Sends: 3})

		mod3stats, err := broker.Stats(mod3)
		requireNoError(t, err)
		compareStats(t, mod3stats, ps.Stats{Skips: 4, Sends: 2})

		allstats := broker.ActiveSubscribers()
		expectEqual(t, 2, len(allstats))
		compareStats(t, allstats[0], mod2stats)
		compareStats(t, allstats[1], mod3stats)

		mod2unsub, err := broker.Unsubscribe(mod2)
		requireNoError(t, err)
		compareStats(t, mod2stats, mod2unsub)

		mod3unsub, err := broker.Unsubscribe(mod3)
		requireNoError(t, err)
		compareStats(t, mod3stats, mod3unsub)
	})
}

func compareStats(tb testing.TB, have, want ps.Stats) {
	tb.Helper()
	if have.String() != want.String() {
		tb.Errorf("stats: have %+v, want %+v", have, want)
	}
}

func requireNoError(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("fatal error: %v", err)
	}
}

func expectEqual[T comparable](tb testing.TB, want, have T) {
	tb.Helper()
	if want != have {
		tb.Errorf("want %+v, have %+v", want, have)
	}
}
