package ps_test

import (
	"strconv"
	"testing"

	"github.com/peterbourgon/ps"
)

func BenchmarkBrokerMethods(b *testing.B) {
	subscribers := []int{
		0,
		1,
		100,
		10000,
	}

	tcs := []struct {
		name string
		fn   func(*ps.Broker[int])
	}{
		{"Publish", func(b *ps.Broker[int]) { b.Publish(123) }},
		{"Sub-Unsub", func(b *ps.Broker[int]) { c := make(chan int); b.SubscribeAll(c); b.Unsubscribe(c) }},
		{"ActiveSubscribers", func(b *ps.Broker[int]) { b.ActiveSubscribers() }},
	}

	for _, tc := range tcs {
		b.Run(tc.name, func(b *testing.B) {
			for _, nsubs := range subscribers {
				b.Run(strconv.Itoa(nsubs), func(b *testing.B) {
					broker := ps.NewBroker[int]()
					for i := 0; i < nsubs; i++ {
						broker.Subscribe(make(chan int), func(int) bool { return true })
					}
					b.ResetTimer()
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						tc.fn(broker)
					}
				})
			}
		})
	}
}
