package analyzer

func ExpandSynonyms(in []string, keep func(string) bool) []string {
	capacity := cap(in)
	out := in[:0:capacity]
	return appendKeptExpandSynonyms(out, in, keep)
}
func appendKeptExpandSynonyms(out, in []string, keep func(string) bool) []string {
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}
