package index

import "sort"

type MergePolicy struct {
	MaxSegments int
	TargetDocs  int
}

func DefaultMergePolicy() MergePolicy { return MergePolicy{MaxSegments: 8, TargetDocs: 10000} }
func (p MergePolicy) Candidates(segments []*Segment) []*Segment {
	if len(segments) <= p.MaxSegments {
		return nil
	}
	copySegments := append([]*Segment(nil), segments...)
	sort.Slice(copySegments, func(i, j int) bool { return len(copySegments[i].docs) < len(copySegments[j].docs) })
	count := len(copySegments) - p.MaxSegments + 1
	if count < 2 {
		count = 2
	}
	if count > len(copySegments) {
		count = len(copySegments)
	}
	return copySegments[:count]
}
