package query

import "testing"

import "runtime/debug"

func noPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if v := recover(); v != nil {
			t.Fatalf("panic: %v\n%s", v, debug.Stack())
		}
	}()
	fn()
}
func TestDssR08OptionsZ08(t *testing.T) {
	noPanic(t, func() {
		o := NewOptions()
		o.AddLabel("a", "b")
		if o.Labels["a"] != "b" {
			t.Fatal("missing label")
		}
	})
}
func TestDssR08TypedNilZ08(t *testing.T) {
	var concrete *RulePolicy
	var policy Policy = concrete
	noPanic(t, func() {
		if PolicyEnabled(policy) {
			t.Fatal("typed nil enabled")
		}
	})
}
func TestDssR08FacetsZ08(t *testing.T) {
	noPanic(t, func() {
		var f Facets
		f.Add("brand", "acme")
		if len(f.Values["brand"]) != 1 {
			t.Fatal("missing facet")
		}
	})
}
func TestDssR08ValidatorsZ08(t *testing.T) {
	noPanic(t, func() {
		var c ValidationChain
		c.Add("required", nil)
		if len(c.Items) != 0 {
			t.Fatal("nil validator stored")
		}
	})
}
