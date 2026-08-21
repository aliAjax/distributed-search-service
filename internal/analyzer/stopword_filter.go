package analyzer

func RemoveStopwords(in []string, keep func(string) bool) []string {
	out := make([]string, 0, len(in))
	return appendKeptRemoveStopwords(out, in, keep)
}
func appendKeptRemoveStopwords(out, in []string, keep func(string) bool) []string {
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}
