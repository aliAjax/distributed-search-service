package analyzer

import (
	"reflect"
	"testing"
)

func preserves(t *testing.T, fn func([]string, func(string) bool) []string) {
	t.Helper()
	original := []string{"drop", "keep", "tail"}
	before := append([]string(nil), original...)
	got := fn(original, func(v string) bool { return v != "drop" })
	if !reflect.DeepEqual(original, before) {
		t.Fatalf("input mutated: %v", original)
	}
	got[0] = "changed"
	if original[1] != "keep" {
		t.Fatalf("result aliases input: %v", original)
	}
}
func TestDssR04TokensZ04(t *testing.T)   { preserves(t, FilterTokens) }
func TestDssR04SynonymsZ04(t *testing.T) { preserves(t, ExpandSynonyms) }
func TestDssR04StopsZ04(t *testing.T)    { preserves(t, RemoveStopwords) }
func TestDssR04RegistryZ04(t *testing.T) { preserves(t, SnapshotPipeline) }
