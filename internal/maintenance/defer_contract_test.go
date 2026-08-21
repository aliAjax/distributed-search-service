package maintenance

import (
	"errors"
	"testing"
)

func TestDssR06LeasesZ06(t *testing.T) {
	active, peak := 0, 0
	ProcessLeases(4, func() func() {
		active++
		if active > peak {
			peak = active
		}
		return func() { active-- }
	})
	if peak != 1 || active != 0 {
		t.Fatalf("active=%d peak=%d", active, peak)
	}
}
func TestDssR06PrimaryZ06(t *testing.T) {
	op, clean := errors.New("operation"), errors.New("cleanup")
	err := RunWithCleanup(func() error { return op }, func() error { return clean })
	if !errors.Is(err, op) || !errors.Is(err, clean) {
		t.Fatalf("err=%v", err)
	}
}
func TestDssR06RollbackZ06(t *testing.T) {
	op, rb := errors.New("operation"), errors.New("rollback")
	called := false
	err := RunWithRollback(func() error { return op }, func() error { called = true; return rb })
	if !called || !errors.Is(err, op) || !errors.Is(err, rb) {
		t.Fatalf("called=%v err=%v", called, err)
	}
}
func TestDssR06CloseZ06(t *testing.T) {
	op, closeErr := errors.New("operation"), errors.New("close")
	err := ResolveCloseResult(func() error { return op }, func() error { return closeErr })
	if !errors.Is(err, op) || !errors.Is(err, closeErr) {
		t.Fatalf("err=%v", err)
	}
}
