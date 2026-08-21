package shard

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func waitClosed(t *testing.T, ch <-chan int) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("channel did not close")
	}
}
func TestDssR05ResultsZ05(t *testing.T) {
	waitClosed(t, ProduceResults([]int{1, 2}, 1))
}
func TestDssR05ErrorsZ05(t *testing.T) {
	errorsOut, done := CollectShardError(errors.New("x"))
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("error sender leaked before receiver was ready")
	}
	select {
	case err := <-errorsOut:
		if err == nil {
			t.Fatal("missing error")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("error delivery blocked")
	}
}
func TestDssR05WaitZ05(t *testing.T) {
	gate := make(chan struct{})
	var ran atomic.Bool
	done := make(chan struct{})
	go func() { StartShardWork(gate, func() { ran.Store(true) }); close(done) }()
	select {
	case <-done:
		t.Fatal("wait returned before shard started")
	case <-time.After(20 * time.Millisecond):
	}
	close(gate)
	<-done
	if !ran.Load() {
		t.Fatal("shard not run")
	}
}
func TestDssR05ConsumerZ05(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	input := make(chan int)
	out := ConsumeStream(ctx, input)
	cancel()
	waitClosed(t, out)
}
