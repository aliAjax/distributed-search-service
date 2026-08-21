package grpc

import (
	"context"
	"errors"
	"example.com/distributed-search-service/internal/index"
	"example.com/distributed-search-service/internal/ingest"
	"example.com/distributed-search-service/internal/query"
)

var ErrUnimplemented = errors.New("gRPC network adapter is not compiled in the standard-library build")

type SearchServer interface {
	Query(context.Context, query.Request) (index.Result, error)
}
type IndexerServer interface {
	Bulk(context.Context, string, string, []ingest.BulkItem) ([]ingest.BulkResult, error)
}
type SnapshotServer interface {
	Create(context.Context, string, string) (string, error)
}
type MaintenanceServer interface {
	Compact(context.Context, string, string) (string, error)
}
type Contracts struct{}

func (Contracts) Query(context.Context, query.Request) (index.Result, error) {
	return index.Result{}, ErrUnimplemented
}
func (Contracts) Bulk(context.Context, string, string, []ingest.BulkItem) ([]ingest.BulkResult, error) {
	return nil, ErrUnimplemented
}
func (Contracts) Create(context.Context, string, string) (string, error) { return "", ErrUnimplemented }
func (Contracts) Compact(context.Context, string, string) (string, error) {
	return "", ErrUnimplemented
}
