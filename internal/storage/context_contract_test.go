package storage

import (
	"context"
	"errors"
	"testing"
)

func canceled(t *testing.T, run func(context.Context, int, func(int) error) error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	err := run(ctx, 4, func(int) error { calls++; return nil })
	if !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}
func TestDssR02ReplayZ02(t *testing.T)  { canceled(t, ReplayWithContext) }
func TestDssR02AppendZ02(t *testing.T)  { canceled(t, AppendWithContext) }
func TestDssR02RecoverZ02(t *testing.T) { canceled(t, RecoverWithContext) }
func TestDssR02CopyZ02(t *testing.T)    { canceled(t, CopyWithContext) }
