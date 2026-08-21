package query

type Facets struct{ Values map[string][]string }

func (f *Facets) Add(k, v string) { f.Values[k] = append(f.Values[k], v) }
