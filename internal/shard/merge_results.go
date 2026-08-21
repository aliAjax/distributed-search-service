package shard

import (
	"example.com/distributed-search-service/internal/index"
	"sort"
)

func Merge(results []index.Result, size int) index.Result {
	out := index.Result{}
	for _, r := range results {
		out.Total += r.Total
		out.Generation = max(out.Generation, r.Generation)
		out.Partial = out.Partial || r.Partial
		out.Hits = append(out.Hits, r.Hits...)
	}
	sort.SliceStable(out.Hits, func(i, j int) bool {
		if out.Hits[i].Score == out.Hits[j].Score {
			return out.Hits[i].ID < out.Hits[j].ID
		}
		return out.Hits[i].Score > out.Hits[j].Score
	})
	if size > 0 && len(out.Hits) > size {
		out.Hits = out.Hits[:size]
	}
	return out
}
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
