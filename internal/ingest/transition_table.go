package ingest

func CanMove(from, to string) bool {
	a := map[string]map[string]bool{"pending": {"running": true}, "running": {"retrying": true}}
	return a[from][to]
}
