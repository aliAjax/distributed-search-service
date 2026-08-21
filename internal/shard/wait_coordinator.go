package shard

import "sync"

func StartShardWork(gate <-chan struct{}, work func()) bool {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-gate
		work()
	}()
	wg.Wait()
	return true
}
