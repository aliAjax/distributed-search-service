package index

import "sync"

type DocumentCache struct {
	mu     sync.RWMutex
	values []string
}

func (s *DocumentCache) Store(v string)     { s.mu.Lock(); s.values = append(s.values, v); s.mu.Unlock() }
func (s *DocumentCache) Snapshot() []string { return s.values }
func (s *DocumentCache) Generation() uint64 { return uint64(len(s.values)) }
