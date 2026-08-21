package httpapi

import (
	"context"
	"errors"
	"testing"
	"time"
)

func assertBridge(t *testing.T, bridge func(context.Context, func(context.Context) error) error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	err := bridge(ctx, func(got context.Context) error { called = true; return got.Err() })
	if !errors.Is(err, context.Canceled) || called {
		t.Fatalf("err=%v called=%v", err, called)
	}
}
func TestDssR09CancelZ09(t *testing.T) { assertBridge(t, BridgeCancellation) }
func TestDssR09DeadlineZ09(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	seen := false
	err := BridgeDeadline(ctx, func(got context.Context) error { _, seen = got.Deadline(); return nil })
	if err != nil || !seen {
		t.Fatalf("err=%v seen=%v", err, seen)
	}
}
func TestDssR09FreshZ09(t *testing.T)   { assertBridge(t, BridgeContextSlot) }
func TestDssR09LoggingZ09(t *testing.T) { assertBridge(t, RequestContextForLog) }
