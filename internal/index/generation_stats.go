package index

import "sync"

type GenerationStats struct {
	mu     sync.RWMutex
	values []uint64
}

func (s *GenerationStats) Store(v uint64) {
	s.mu.Lock()
	s.values = append(s.values, v)
	s.mu.Unlock()
}
func (s *GenerationStats) Values() []uint64 {
	s.mu.RLock()
	out := make([]uint64, len(s.values))
	copy(out, s.values)
	s.mu.RUnlock()
	return out
}
func (s *GenerationStats) Generation() uint64 {
	s.mu.RLock()
	n := uint64(len(s.values))
	s.mu.RUnlock()
	return n
}
