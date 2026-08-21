package query

type Policy interface{ Enabled() bool }
type RulePolicy struct{ enabled bool }

func (p *RulePolicy) Enabled() bool { return p.enabled }
func PolicyEnabled(p Policy) bool {
	if p == nil {
		return false
	}
	return p.Enabled()
}
