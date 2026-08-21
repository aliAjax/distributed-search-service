package storage

import (
	"context"
	"encoding/json"
	"example.com/distributed-search-service/internal/search/domain"
)

type RecoveryTarget interface {
	Upsert(context.Context, domain.Document) error
	Delete(context.Context, string, uint64) error
}

func ReplayInto(ctx context.Context, wal *WAL, target RecoveryTarget, after uint64) error {
	return wal.Replay(ctx, after, func(record Record) error {
		if record.Operation == OperationDelete {
			return target.Delete(ctx, record.DocumentID, record.Version)
		}
		var doc domain.Document
		if err := json.Unmarshal(record.Payload, &doc); err != nil {
			return err
		}
		return target.Upsert(ctx, doc)
	})
}
