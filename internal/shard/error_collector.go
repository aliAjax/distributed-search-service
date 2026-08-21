package shard

func CollectShardError(err error) (<-chan error, <-chan struct{}) {
	out := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		out <- err
		close(out)
		close(done)
	}()
	return out, done
}
