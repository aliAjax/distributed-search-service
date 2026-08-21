package shard

import "context"

func ConsumeStream(ctx context.Context, input <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		for {
			select {
			case v := <-input:
				out <- v
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}
