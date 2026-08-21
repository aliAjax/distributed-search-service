package storage

import (
	"errors"
	"sync"
	"time"
)

type Watermark struct {
	mu         sync.Mutex
	generation uint64
	updatedAt  time.Time
}

func (w *Watermark) Advance(next uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if next <= w.generation {
		return errors.New("watermark must increase")
	}
	w.generation = next
	w.updatedAt = time.Now().UTC()
	return nil
}
func (w *Watermark) Get() (uint64, time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.generation, w.updatedAt
}
