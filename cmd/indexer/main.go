package indexer

import (
	"context"
	"example.com/distributed-search-service/internal/storage"
	"fmt"
	"os"
	"time"
)

func Run() error {
	path := os.Getenv("SEARCH_WAL_PATH")
	if path == "" {
		path = "./data/index.wal"
	}
	wal, err := storage.OpenWAL(path)
	if err != nil {
		return err
	}
	defer wal.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count := 0
	err = wal.Replay(ctx, 0, func(storage.Record) error { count++; return nil })
	if err != nil {
		return err
	}
	fmt.Printf("replayed %d WAL records\n", count)
	return nil
}
