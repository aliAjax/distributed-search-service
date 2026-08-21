package shard

func CollectShardError(err error) (<-chan error, <-chan struct{}) {
	out := make(chan error)
	done := make(chan struct{})
	go func() { out <- err; close(out); close(done) }()
	return out, done
}
