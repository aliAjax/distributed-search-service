package index

import (
	"sync"
	"testing"
)

func exerciseStrings(t *testing.T, store func(string), snapshot func() []string) {
	t.Helper()
	store("base")
	held := snapshot()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 200; j++ {
				store("next")
				_ = snapshot()
			}
		}()
	}
	close(start)
	wg.Wait()
	held[0] = "mutated"
	if snapshot()[0] != "base" {
		t.Fatal("snapshot escaped internal storage")
	}
}
func TestDssR03ViewZ03(t *testing.T) {
	var s SnapshotView
	exerciseStrings(t, s.Store, s.Copy)
}
func TestDssR03CacheZ03(t *testing.T) {
	var s DocumentCache
	exerciseStrings(t, s.Store, s.Snapshot)
}
func TestDssR03CatalogZ03(t *testing.T) {
	var s SegmentCatalog
	s.Store(1)
	held := s.List()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			<-start
			for j := 0; j < 200; j++ {
				s.Store(v)
				_ = s.List()
			}
		}(i)
	}
	close(start)
	wg.Wait()
	held[0] = 99
	if s.List()[0] != 1 {
		t.Fatal("catalog snapshot escaped")
	}
}
func TestDssR03StatsZ03(t *testing.T) {
	var s GenerationStats
	s.Store(1)
	held := s.Values()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(v uint64) {
			defer wg.Done()
			<-start
			for j := 0; j < 200; j++ {
				s.Store(v)
				_ = s.Values()
			}
		}(uint64(i))
	}
	close(start)
	wg.Wait()
	held[0] = 99
	if s.Values()[0] != 1 {
		t.Fatal("stats snapshot escaped")
	}
}
