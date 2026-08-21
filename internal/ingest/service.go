package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"example.com/distributed-search-service/internal/index"
	"example.com/distributed-search-service/internal/search/domain"
	"example.com/distributed-search-service/internal/storage"
)

type CollectionSource interface {
	Get(context.Context, string, string) (domain.Collection, error)
}
type Service struct {
	collections CollectionSource
	wal         *storage.WAL
	enginesMu   sync.RWMutex
	engines     map[string]*index.Engine
	maxFields   int
	maxBulk     int
}

func New(collections CollectionSource, wal *storage.WAL, maxFields, maxBulk int) *Service {
	return &Service{collections: collections, wal: wal, engines: map[string]*index.Engine{}, maxFields: maxFields, maxBulk: maxBulk}
}
func (s *Service) RegisterEngine(tenant, collection string, engine *index.Engine) {
	s.enginesMu.Lock()
	defer s.enginesMu.Unlock()
	s.engines[tenant+"\x00"+collection] = engine
}
func (s *Service) Engine(tenant, collection string) (*index.Engine, bool) {
	s.enginesMu.RLock()
	defer s.enginesMu.RUnlock()
	e, ok := s.engines[tenant+"\x00"+collection]
	return e, ok
}
func (s *Service) Upsert(ctx context.Context, tenant, collection, id string, fields map[string]any, version uint64) (domain.Document, error) {
	c, err := s.collections.Get(ctx, tenant, collection)
	if err != nil {
		return domain.Document{}, err
	}
	engine, ok := s.Engine(tenant, collection)
	if !ok {
		return domain.Document{}, errors.New("collection engine unavailable")
	}
	now := time.Now().UTC()
	doc := domain.Document{ID: id, TenantID: tenant, CollectionID: collection, Fields: fields, ExternalVersion: version, SchemaVersion: c.Schema.Version, CreatedAt: now, UpdatedAt: now}
	if err = doc.Validate(c.Schema, s.maxFields); err != nil {
		return domain.Document{}, err
	}
	payload, err := json.Marshal(doc)
	if err != nil {
		return domain.Document{}, err
	}
	if _, err = s.wal.Append(ctx, storage.Record{Operation: storage.OperationUpsert, TenantID: tenant, CollectionID: collection, DocumentID: id, Version: version, Payload: payload}); err != nil {
		return domain.Document{}, fmt.Errorf("append WAL: %w", err)
	}
	if err = engine.Upsert(doc); err != nil {
		return domain.Document{}, err
	}
	return doc, nil
}
func (s *Service) Delete(ctx context.Context, tenant, collection, id string, version uint64) error {
	engine, ok := s.Engine(tenant, collection)
	if !ok {
		return errors.New("collection engine unavailable")
	}
	if _, err := s.wal.Append(ctx, storage.Record{Operation: storage.OperationDelete, TenantID: tenant, CollectionID: collection, DocumentID: id, Version: version}); err != nil {
		return err
	}
	return engine.Delete(id, version)
}

type BulkItem struct {
	ID      string         `json:"id"`
	Version uint64         `json:"version"`
	Fields  map[string]any `json:"fields"`
}
type BulkResult struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
	Error  string `json:"error,omitempty"`
}

func (s *Service) Bulk(ctx context.Context, tenant, collection string, items []BulkItem) []BulkResult {
	if len(items) > s.maxBulk {
		return []BulkResult{{Status: 413, Error: "bulk item limit exceeded"}}
	}
	results := make([]BulkResult, len(items))
	for i, item := range items {
		select {
		case <-ctx.Done():
			results[i] = BulkResult{ID: item.ID, Status: 499, Error: ctx.Err().Error()}
			continue
		default:
		}
		_, err := s.Upsert(ctx, tenant, collection, item.ID, item.Fields, item.Version)
		if err != nil {
			results[i] = BulkResult{ID: item.ID, Status: 422, Error: err.Error()}
		} else {
			results[i] = BulkResult{ID: item.ID, Status: 200}
		}
	}
	return results
}
func HashRequest(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
