package analyzer

func FilterTokens(in []string, keep func(string) bool) []string {
	out := make([]string, 0, len(in))
	return appendKeptFilterTokens(out, in, keep)
}
func appendKeptFilterTokens(out, in []string, keep func(string) bool) []string {
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}
