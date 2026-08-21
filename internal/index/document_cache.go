package index

import "sync"

type DocumentCache struct {
	mu     sync.RWMutex
	values []string
}

func (s *DocumentCache) Store(v string) {
	s.mu.Lock()
	s.values = append(s.values, v)
	s.mu.Unlock()
}
func (s *DocumentCache) Snapshot() []string {
	s.mu.RLock()
	out := make([]string, len(s.values))
	copy(out, s.values)
	s.mu.RUnlock()
	return out
}
func (s *DocumentCache) Generation() uint64 {
	s.mu.RLock()
	n := uint64(len(s.values))
	s.mu.RUnlock()
	return n
}
