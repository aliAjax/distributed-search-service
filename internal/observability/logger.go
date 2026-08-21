package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type Metrics struct {
	requests     atomic.Uint64
	errors       atomic.Uint64
	searches     atomic.Uint64
	ingested     atomic.Uint64
	latencyNanos atomic.Uint64
}

func (m *Metrics) RecordRequest(err bool, d time.Duration) {
	m.requests.Add(1)
	m.latencyNanos.Add(uint64(d))
	if err {
		m.errors.Add(1)
	}
}
func (m *Metrics) RecordSearch()         { m.searches.Add(1) }
func (m *Metrics) RecordIngest(n uint64) { m.ingested.Add(n) }
func (m *Metrics) Snapshot() map[string]uint64 {
	return map[string]uint64{"requests_total": m.requests.Load(), "errors_total": m.errors.Load(), "searches_total": m.searches.Load(), "documents_ingested_total": m.ingested.Load(), "request_latency_nanos_total": m.latencyNanos.Load()}
}
func NewLogger(level string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}

type contextKey string

const requestIDKey contextKey = "request-id"

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}
func RequestID(ctx context.Context) string { v, _ := ctx.Value(requestIDKey).(string); return v }
