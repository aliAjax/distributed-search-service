package shard

func ProduceResults(values []int, failAt int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for i, v := range values {
			if i == failAt {
				return
			}
			out <- v
		}
	}()
	return out
}
