package repository

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"example.com/distributed-search-service/internal/search/domain"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

type CollectionRepository interface {
	Create(context.Context, domain.Collection) error
	Get(context.Context, string, string) (domain.Collection, error)
	List(context.Context, string, int, string) ([]domain.Collection, string, error)
	Update(context.Context, domain.Collection, uint64) error
}
type MemoryCollections struct {
	mu    sync.RWMutex
	items map[string]domain.Collection
}

func NewMemoryCollections() *MemoryCollections {
	return &MemoryCollections{items: map[string]domain.Collection{}}
}
func key(tenant, id string) string { return tenant + "\x00" + id }
func (r *MemoryCollections) Create(_ context.Context, c domain.Collection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(c.TenantID, c.ID)
	if _, ok := r.items[k]; ok {
		return ErrConflict
	}
	r.items[k] = cloneCollection(c)
	return nil
}
func (r *MemoryCollections) Get(_ context.Context, tenant, id string) (domain.Collection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.items[key(tenant, id)]
	if !ok {
		return domain.Collection{}, ErrNotFound
	}
	return cloneCollection(c), nil
}
func (r *MemoryCollections) List(_ context.Context, tenant string, limit int, after string) ([]domain.Collection, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var items []domain.Collection
	for _, c := range r.items {
		if c.TenantID == tenant && c.ID > after {
			items = append(items, cloneCollection(c))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	next := ""
	if len(items) > limit {
		next = items[limit-1].ID
		items = items[:limit]
	}
	return items, next, nil
}
func (r *MemoryCollections) Update(_ context.Context, c domain.Collection, expected uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[key(c.TenantID, c.ID)]
	if !ok {
		return ErrNotFound
	}
	if current.Version != expected {
		return ErrConflict
	}
	r.items[key(c.TenantID, c.ID)] = cloneCollection(c)
	return nil
}
func cloneCollection(c domain.Collection) domain.Collection {
	c.Schema.Fields = append([]domain.Field(nil), c.Schema.Fields...)
	return c
}

type IdempotencyRecord struct {
	TenantID    string
	Key         string
	RequestHash string
	ResourceID  string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}
type IdempotencyStore struct {
	mu      sync.Mutex
	records map[string]IdempotencyRecord
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{records: map[string]IdempotencyRecord{}}
}
func (s *IdempotencyStore) Reserve(tenant, keyValue, hash, resource string, now time.Time) (string, bool, error) {
	if keyValue == "" {
		return resource, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(tenant, keyValue)
	if old, ok := s.records[k]; ok && old.ExpiresAt.After(now) {
		if old.RequestHash != hash {
			return "", false, ErrConflict
		}
		return old.ResourceID, true, nil
	}
	s.records[k] = IdempotencyRecord{TenantID: tenant, Key: keyValue, RequestHash: hash, ResourceID: resource, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
	return resource, false, nil
}
