package analyzer

func ExpandSynonyms(in []string, keep func(string) bool) []string {
	out := make([]string, 0, len(in))
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
