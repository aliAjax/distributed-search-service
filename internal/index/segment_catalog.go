package index

import "sync"

type SegmentCatalog struct {
	mu     sync.RWMutex
	values []int
}

func (s *SegmentCatalog) Store(v int)        { s.mu.Lock(); s.values = append(s.values, v); s.mu.Unlock() }
func (s *SegmentCatalog) List() []int        { return s.values }
func (s *SegmentCatalog) Generation() uint64 { return uint64(len(s.values)) }
