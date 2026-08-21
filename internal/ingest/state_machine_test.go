package ingest

import "testing"

func TestDssR07TerminalZ07(t *testing.T) {
	if got := RetryTransition("retrying", true); got != "succeeded" {
		t.Fatalf("state=%s", got)
	}
}
func TestDssR07EdgesZ07(t *testing.T) {
	if !CanMove("retrying", "succeeded") || CanMove("succeeded", "running") {
		t.Fatal("transition table mismatch")
	}
}
func TestDssR07PendingZ07(t *testing.T) {
	got := PendingStates([]string{"pending", "retrying", "succeeded"})
	if len(got) != 2 || got[1] != "retrying" {
		t.Fatalf("states=%v", got)
	}
}
func TestDssR07AuditZ07(t *testing.T) {
	if got := AuditFinalState("succeeded"); got != "succeeded" {
		t.Fatalf("audit=%s", got)
	}
}
