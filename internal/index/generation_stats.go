package index

import "sync"

type GenerationStats struct {
	mu     sync.RWMutex
	values []uint64
}

func (s *GenerationStats) Store(v uint64)     { s.mu.Lock(); s.values = append(s.values, v); s.mu.Unlock() }
func (s *GenerationStats) Values() []uint64   { return s.values }
func (s *GenerationStats) Generation() uint64 { return uint64(len(s.values)) }
