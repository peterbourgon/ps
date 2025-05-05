package pshttp_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/peterbourgon/ps"
	"github.com/peterbourgon/ps/pshttp"
)

func TestBasics(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	broker := ps.NewBroker[int64]()

	handler := pshttp.NewHandler(broker, pshttp.EncodeJSON, pshttp.DecodeJSON, newTestWriter(t))
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := pshttp.NewDefaultClient[int64](server.URL)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	spawn := func() (valc chan int64, stop func(), errc chan error) {
		ctx, cancel := context.WithCancel(ctx)
		valc = make(chan int64, 1)
		errc = make(chan error, 1)
		go func() { defer close(valc); errc <- client.Subscribe(ctx, valc, 100*time.Millisecond) }()
		time.Sleep(100 * time.Millisecond)
		return valc, cancel, errc
	}

	publish := func(v int64) {
		t.Helper()
		stats, err := client.Publish(ctx, v)
		if err != nil {
			t.Fatalf("publish: %v", err)
		}
		t.Logf("publish(%v): %v", v, stats)
	}

	recvAndCheck := func(valc <-chan int64, want int64) {
		t.Helper()
		select {
		case v := <-valc:
			if want != v {
				t.Errorf("want %v, have %v", want, v)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout waiting for value")
		}
	}

	stopAndCheck := func(stopfunc func(), errc <-chan error) {
		t.Helper()
		stopfunc()
		if err := <-errc; errors.Is(err, context.Canceled) {
			err = nil
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	v1, s1, e1 := spawn()

	publish(1)
	recvAndCheck(v1, 1)

	v2, s2, e2 := spawn()

	publish(2)
	recvAndCheck(v1, 2)
	recvAndCheck(v2, 2)

	stopAndCheck(s1, e1)

	publish(3)
	recvAndCheck(v1, 0)
	recvAndCheck(v2, 3)

	stopAndCheck(s2, e2)

	publish(4)
	recvAndCheck(v1, 0)
	recvAndCheck(v2, 0)
}

type testWriter struct {
	tb testing.TB
}

func newTestWriter(tb testing.TB) *testWriter {
	return &testWriter{
		tb: tb,
	}
}

func (tw *testWriter) Write(p []byte) (int, error) {
	tw.tb.Logf("%s", string(p))
	return len(p), nil
}
