package index

import "sync"

type SnapshotView struct {
	mu     sync.RWMutex
	values []string
}

func (s *SnapshotView) Store(v string)     { s.mu.Lock(); s.values = append(s.values, v); s.mu.Unlock() }
func (s *SnapshotView) Copy() []string     { return s.values }
func (s *SnapshotView) Generation() uint64 { return uint64(len(s.values)) }
