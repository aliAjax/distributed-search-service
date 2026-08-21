package shard

import (
	"context"
	"errors"
	"example.com/distributed-search-service/internal/index"
	"example.com/distributed-search-service/internal/query"
	"hash/fnv"
	"sync"
)

type Coordinator struct {
	mu     sync.RWMutex
	shards map[int]*index.Engine
	count  int
}

func NewCoordinator(count int) *Coordinator {
	if count < 1 {
		count = 1
	}
	return &Coordinator{shards: map[int]*index.Engine{}, count: count}
}
func (c *Coordinator) Register(id int, engine *index.Engine) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shards[id] = engine
}
func (c *Coordinator) Route(key string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % uint32(c.count))
}
func (c *Coordinator) Query(ctx context.Context, req query.Request) ([]index.Result, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.shards) == 0 {
		return nil, errors.New("no shards registered")
	}
	out := make([]index.Result, 0, len(c.shards))
	for _, engine := range c.shards {
		result, err := engine.Search(ctx, req)
		if err != nil {
			return out, err
		}
		out = append(out, result)
	}
	return out, nil
}
