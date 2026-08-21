package index

import "sync"

type SegmentCatalog struct {
	mu     sync.RWMutex
	values []int
}

func (s *SegmentCatalog) Store(v int) {
	s.mu.Lock()
	s.values = append(s.values, v)
	s.mu.Unlock()
}
func (s *SegmentCatalog) List() []int {
	s.mu.RLock()
	out := make([]int, len(s.values))
	copy(out, s.values)
	s.mu.RUnlock()
	return out
}
func (s *SegmentCatalog) Generation() uint64 {
	s.mu.RLock()
	n := uint64(len(s.values))
	s.mu.RUnlock()
	return n
}
