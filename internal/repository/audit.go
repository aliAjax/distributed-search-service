package repository

import (
	"sync"
	"time"
)

type AuditEntry struct {
	ID        string            `json:"id"`
	TenantID  string            `json:"tenant_id"`
	Actor     string            `json:"actor"`
	Action    string            `json:"action"`
	Resource  string            `json:"resource"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
}
type AuditLog struct {
	mu      sync.RWMutex
	entries []AuditEntry
}

func (a *AuditLog) Append(entry AuditEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry.CreatedAt = entry.CreatedAt.UTC()
	a.entries = append(a.entries, entry)
}
func (a *AuditLog) List(tenant string, limit int) []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]AuditEntry, 0, limit)
	for i := len(a.entries) - 1; i >= 0 && len(out) < limit; i-- {
		if a.entries[i].TenantID == tenant {
			out = append(out, a.entries[i])
		}
	}
	return out
}
