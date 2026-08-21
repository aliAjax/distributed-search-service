package analyzer

func SnapshotPipeline(in []string, keep func(string) bool) []string {
	capacity := cap(in)
	out := in[:0:capacity]
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
