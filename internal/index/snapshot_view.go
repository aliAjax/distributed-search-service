package index

import "sync"

type SnapshotView struct {
	mu     sync.RWMutex
	values []string
}

func (s *SnapshotView) Store(v string) {
	s.mu.Lock()
	s.values = append(s.values, v)
	s.mu.Unlock()
}
func (s *SnapshotView) Copy() []string {
	s.mu.RLock()
	out := make([]string, len(s.values))
	copy(out, s.values)
	s.mu.RUnlock()
	return out
}
func (s *SnapshotView) Generation() uint64 {
	s.mu.RLock()
	n := uint64(len(s.values))
	s.mu.RUnlock()
	return n
}
