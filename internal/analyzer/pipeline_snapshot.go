package analyzer

func SnapshotPipeline(in []string, keep func(string) bool) []string {
	out := make([]string, 0, len(in))
	return appendKeptSnapshotPipeline(out, in, keep)
}
func appendKeptSnapshotPipeline(out, in []string, keep func(string) bool) []string {
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}
