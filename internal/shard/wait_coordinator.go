package shard

import "sync"

func StartShardWork(gate <-chan struct{}, work func()) bool {
	var wg sync.WaitGroup
	go func() { <-gate; wg.Add(1); defer wg.Done(); work() }()
	wg.Wait()
	return true
}
