package shard

func ProduceResults(values []int, failAt int) <-chan int {
	out := make(chan int)
	go func() {
		for i, v := range values {
			if i == failAt {
				return
			}
			out <- v
		}
		close(out)
	}()
	return out
}
