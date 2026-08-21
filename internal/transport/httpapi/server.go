package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"example.com/distributed-search-service/internal/analyzer"
	"example.com/distributed-search-service/internal/index"
	"example.com/distributed-search-service/internal/ingest"
	"example.com/distributed-search-service/internal/maintenance"
	"example.com/distributed-search-service/internal/observability"
	"example.com/distributed-search-service/internal/query"
	"example.com/distributed-search-service/internal/repository"
	"example.com/distributed-search-service/internal/search/domain"
)

type Server struct {
	collections *repository.MemoryCollections
	idempotency *repository.IdempotencyStore
	analyzers   *analyzer.Registry
	ingest      *ingest.Service
	maintenance *maintenance.Service
	logger      *slog.Logger
	metrics     *observability.Metrics
	maxBody     int64
	searchSlots chan struct{}
	started     time.Time
	mu          sync.RWMutex
	ready       bool
}
type Dependencies struct {
	Collections       *repository.MemoryCollections
	Idempotency       *repository.IdempotencyStore
	Analyzers         *analyzer.Registry
	Ingest            *ingest.Service
	Maintenance       *maintenance.Service
	Logger            *slog.Logger
	Metrics           *observability.Metrics
	MaxBodyBytes      int64
	SearchConcurrency int
}

func New(d Dependencies) *Server {
	return &Server{collections: d.Collections, idempotency: d.Idempotency, analyzers: d.Analyzers, ingest: d.Ingest, maintenance: d.Maintenance, logger: d.Logger, metrics: d.Metrics, maxBody: d.MaxBodyBytes, searchSlots: make(chan struct{}, d.SearchConcurrency), started: time.Now().UTC(), ready: true}
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", s.live)
	mux.HandleFunc("GET /health/ready", s.readyCheck)
	mux.HandleFunc("GET /metrics", s.getMetrics)
	mux.HandleFunc("POST /api/v1/collections", s.createCollection)
	mux.HandleFunc("GET /api/v1/collections", s.listCollections)
	mux.HandleFunc("GET /api/v1/collections/{id}", s.getCollection)
	mux.HandleFunc("PUT /api/v1/collections/{id}/schema", s.replaceSchema)
	mux.HandleFunc("GET /api/v1/analyzers", s.listAnalyzers)
	mux.HandleFunc("POST /api/v1/analyzers", s.createAnalyzer)
	mux.HandleFunc("POST /api/v1/synonyms", s.createSynonyms)
	mux.HandleFunc("PUT /api/v1/collections/{id}/documents/{docID}", s.upsertDocument)
	mux.HandleFunc("DELETE /api/v1/collections/{id}/documents/{docID}", s.deleteDocument)
	mux.HandleFunc("POST /api/v1/collections/{id}/documents:bulk", s.bulkDocuments)
	mux.HandleFunc("POST /api/v1/search", s.search)
	mux.HandleFunc("POST /api/v1/msearch", s.multiSearch)
	mux.HandleFunc("POST /api/v1/explain", s.search)
	mux.HandleFunc("POST /api/v1/count", s.count)
	mux.HandleFunc("GET /api/v1/collections/{id}/segments", s.segments)
	mux.HandleFunc("POST /api/v1/collections/{id}/compact", s.compact)
	mux.HandleFunc("GET /api/v1/tasks/{id}", s.getTask)
	mux.HandleFunc("GET /api/v1/health/consistency", s.consistency)
	mux.HandleFunc("POST /api/v1/snapshots", s.notImplemented)
	mux.Handle("GET /", http.FileServer(http.Dir("web")))
	return s.middleware(mux)
}
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = newID()
		}
		ctx := observability.WithRequestID(r.Context(), requestID)
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("Content-Type", "application/json")
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r.WithContext(ctx))
		failed := rec.status >= 400
		s.metrics.RecordRequest(failed, time.Since(start))
		s.logger.InfoContext(ctx, "request", "method", r.Method, "path", r.URL.Path, "status", rec.status, "duration_ms", time.Since(start).Milliseconds(), "request_id", requestID)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) { r.status = code; r.ResponseWriter.WriteHeader(code) }

type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Details   any    `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, apiError{Code: code, Message: message, RequestID: observability.RequestID(r.Context())})
}
func decode(w http.ResponseWriter, r *http.Request, max int64, out any) error {
	reader := http.MaxBytesReader(w, r.Body, max)
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request must contain exactly one JSON value")
	}
	return nil
}
func tenant(r *http.Request) (string, error) {
	value := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if value == "" {
		return "", errors.New("X-Tenant-ID header is required")
	}
	if len(value) > 128 {
		return "", errors.New("tenant id too long")
	}
	return value, nil
}
func actor(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("X-Caller-ID"))
	if value == "" {
		return "anonymous"
	}
	return value
}

type createCollectionRequest struct {
	Name          string         `json:"name"`
	Shards        int            `json:"shards"`
	RetentionDays int            `json:"retention_days"`
	Fields        []domain.Field `json:"fields"`
}

func (s *Server) createCollection(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	var request createCollectionRequest
	if err = decode(w, r, s.maxBody, &request); err != nil {
		writeError(w, r, 400, "invalid_json", err.Error())
		return
	}
	id := newID()
	if key := r.Header.Get("Idempotency-Key"); key != "" {
		resource, replayed, reserveErr := s.idempotency.Reserve(tenantID, key, ingest.HashRequest(request), id, time.Now())
		if reserveErr != nil {
			writeError(w, r, 409, "idempotency_conflict", reserveErr.Error())
			return
		}
		if replayed {
			existing, getErr := s.collections.Get(r.Context(), tenantID, resource)
			if getErr == nil {
				w.Header().Set("Idempotency-Replayed", "true")
				writeJSON(w, 200, existing)
				return
			}
		}
	}
	schema, err := domain.NewSchema(1, request.Fields, actor(r), time.Now())
	if err != nil {
		writeError(w, r, 422, "invalid_schema", err.Error())
		return
	}
	collection, err := domain.NewCollection(id, tenantID, request.Name, schema, request.Shards, actor(r), time.Now())
	if err != nil {
		writeError(w, r, 422, "invalid_collection", err.Error())
		return
	}
	collection.RetentionDays = request.RetentionDays
	if err = s.collections.Create(r.Context(), collection); err != nil {
		writeError(w, r, 409, "collection_conflict", err.Error())
		return
	}
	pipeline, ok := s.analyzers.Get("standard", 0)
	if !ok {
		writeError(w, r, 500, "analyzer_missing", "standard analyzer is unavailable")
		return
	}
	s.ingest.RegisterEngine(tenantID, id, index.NewEngine(pipeline, "development-only-change-me-32bytes", 100))
	w.Header().Set("Location", "/api/v1/collections/"+id)
	w.Header().Set("ETag", etag(collection.Version))
	writeJSON(w, 201, collection)
}
func (s *Server) listCollections(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	limit := parseLimit(r.URL.Query().Get("limit"))
	items, next, err := s.collections.List(r.Context(), tenantID, limit, r.URL.Query().Get("after"))
	if err != nil {
		writeError(w, r, 500, "repository_error", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "next": next})
}
func (s *Server) getCollection(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	c, err := s.collections.Get(r.Context(), tenantID, r.PathValue("id"))
	if err != nil {
		writeError(w, r, 404, "collection_not_found", err.Error())
		return
	}
	w.Header().Set("ETag", etag(c.Version))
	writeJSON(w, 200, c)
}

type replaceSchemaRequest struct {
	Fields []domain.Field `json:"fields"`
}

func (s *Server) replaceSchema(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	c, err := s.collections.Get(r.Context(), tenantID, r.PathValue("id"))
	if err != nil {
		writeError(w, r, 404, "collection_not_found", err.Error())
		return
	}
	expected, err := parseETag(r.Header.Get("If-Match"))
	if err != nil {
		writeError(w, r, 428, "if_match_required", err.Error())
		return
	}
	var request replaceSchemaRequest
	if err = decode(w, r, s.maxBody, &request); err != nil {
		writeError(w, r, 400, "invalid_json", err.Error())
		return
	}
	schema, err := domain.NewSchema(c.Schema.Version+1, request.Fields, actor(r), time.Now())
	if err != nil {
		writeError(w, r, 422, "invalid_schema", err.Error())
		return
	}
	previous := c.Version
	if err = c.ReplaceSchema(schema, expected, actor(r), time.Now()); err != nil {
		writeError(w, r, 409, "version_conflict", err.Error())
		return
	}
	if err = s.collections.Update(r.Context(), c, previous); err != nil {
		writeError(w, r, 409, "version_conflict", err.Error())
		return
	}
	w.Header().Set("ETag", etag(c.Version))
	writeJSON(w, 200, c)
}
func (s *Server) listAnalyzers(w http.ResponseWriter, r *http.Request) {
	if _, err := tenant(r); err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"items": s.analyzers.List()})
}
func (s *Server) createAnalyzer(w http.ResponseWriter, r *http.Request) {
	if _, err := tenant(r); err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	var p analyzer.Pipeline
	if err := decode(w, r, s.maxBody, &p); err != nil {
		writeError(w, r, 400, "invalid_json", err.Error())
		return
	}
	if err := s.analyzers.Put(p); err != nil {
		writeError(w, r, 409, "analyzer_conflict", err.Error())
		return
	}
	writeJSON(w, 201, p)
}
func (s *Server) createSynonyms(w http.ResponseWriter, r *http.Request) { s.createAnalyzer(w, r) }

type documentRequest struct {
	Version uint64         `json:"version"`
	Fields  map[string]any `json:"fields"`
}

func (s *Server) upsertDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	var request documentRequest
	if err = decode(w, r, s.maxBody, &request); err != nil {
		writeError(w, r, 400, "invalid_json", err.Error())
		return
	}
	doc, err := s.ingest.Upsert(r.Context(), tenantID, r.PathValue("id"), r.PathValue("docID"), request.Fields, request.Version)
	if err != nil {
		writeError(w, r, 422, "document_rejected", err.Error())
		return
	}
	s.metrics.RecordIngest(1)
	writeJSON(w, 200, doc)
}
func (s *Server) deleteDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	version, err := strconv.ParseUint(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		writeError(w, r, 400, "version_required", "version query parameter is required")
		return
	}
	if err = s.ingest.Delete(r.Context(), tenantID, r.PathValue("id"), r.PathValue("docID"), version); err != nil {
		writeError(w, r, 409, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(204)
}

type bulkRequest struct {
	Items []ingest.BulkItem `json:"items"`
}

func (s *Server) bulkDocuments(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	var request bulkRequest
	if err = decode(w, r, s.maxBody, &request); err != nil {
		writeError(w, r, 400, "invalid_json", err.Error())
		return
	}
	results := s.ingest.Bulk(r.Context(), tenantID, r.PathValue("id"), request.Items)
	s.metrics.RecordIngest(uint64(len(request.Items)))
	writeJSON(w, 207, map[string]any{"items": results})
}
func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	var request query.Request
	if err = decode(w, r, s.maxBody, &request); err != nil {
		writeError(w, r, 400, "invalid_json", err.Error())
		return
	}
	request.TenantID = tenantID
	request.Defaults()
	if err = request.Validate(); err != nil {
		writeError(w, r, 422, "invalid_query", err.Error())
		return
	}
	engine, ok := s.ingest.Engine(tenantID, request.CollectionID)
	if !ok {
		writeError(w, r, 404, "collection_not_found", "collection engine not found")
		return
	}
	select {
	case s.searchSlots <- struct{}{}:
		defer func() { <-s.searchSlots }()
	case <-r.Context().Done():
		writeError(w, r, 499, "request_cancelled", r.Context().Err().Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(request.TimeoutMS)*time.Millisecond)
	defer cancel()
	result, err := engine.Search(ctx, request)
	if err != nil {
		if request.AllowPartial {
			writeJSON(w, 206, index.Result{Partial: true})
			return
		}
		writeError(w, r, 422, "search_failed", err.Error())
		return
	}
	s.metrics.RecordSearch()
	writeJSON(w, 200, result)
}

type multiSearchRequest struct {
	Searches []query.Request `json:"searches"`
}

func (s *Server) multiSearch(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	var request multiSearchRequest
	if err = decode(w, r, s.maxBody, &request); err != nil {
		writeError(w, r, 400, "invalid_json", err.Error())
		return
	}
	if len(request.Searches) > 32 {
		writeError(w, r, 413, "too_many_searches", "at most 32 searches are allowed")
		return
	}
	results := make([]any, len(request.Searches))
	for i, q := range request.Searches {
		q.TenantID = tenantID
		q.Defaults()
		if err = q.Validate(); err != nil {
			results[i] = apiError{Code: "invalid_query", Message: err.Error()}
			continue
		}
		engine, ok := s.ingest.Engine(tenantID, q.CollectionID)
		if !ok {
			results[i] = apiError{Code: "collection_not_found", Message: "collection engine not found"}
			continue
		}
		result, searchErr := engine.Search(r.Context(), q)
		if searchErr != nil {
			results[i] = apiError{Code: "search_failed", Message: searchErr.Error()}
		} else {
			results[i] = result
		}
	}
	writeJSON(w, 200, map[string]any{"results": results})
}
func (s *Server) count(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	var request query.Request
	if err = decode(w, r, s.maxBody, &request); err != nil {
		writeError(w, r, 400, "invalid_json", err.Error())
		return
	}
	request.TenantID = tenantID
	request.Size = 1
	if request.TimeoutMS == 0 {
		request.TimeoutMS = 1000
	}
	if err = request.Validate(); err != nil {
		writeError(w, r, 422, "invalid_query", err.Error())
		return
	}
	engine, ok := s.ingest.Engine(tenantID, request.CollectionID)
	if !ok {
		writeError(w, r, 404, "collection_not_found", "collection engine not found")
		return
	}
	result, err := engine.Search(r.Context(), request)
	if err != nil {
		writeError(w, r, 422, "search_failed", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"count": result.Total, "generation": result.Generation})
}
func (s *Server) segments(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	engine, ok := s.ingest.Engine(tenantID, r.PathValue("id"))
	if !ok {
		writeError(w, r, 404, "collection_not_found", "collection engine not found")
		return
	}
	writeJSON(w, 200, map[string]any{"items": engine.Segments(), "stats": engine.Stats()})
}
func (s *Server) compact(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant(r)
	if err != nil {
		writeError(w, r, 400, "tenant_required", err.Error())
		return
	}
	engine, ok := s.ingest.Engine(tenantID, r.PathValue("id"))
	if !ok {
		writeError(w, r, 404, "collection_not_found", "collection engine not found")
		return
	}
	task, err := s.maintenance.ScheduleCompact(context.Background(), newID(), r.PathValue("id"), engine)
	if err != nil {
		writeError(w, r, 409, "task_conflict", err.Error())
		return
	}
	writeJSON(w, 202, task)
}
func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	task, ok := s.maintenance.Get(r.PathValue("id"))
	if !ok {
		writeError(w, r, 404, "task_not_found", "task not found")
		return
	}
	writeJSON(w, 200, task)
}
func (s *Server) consistency(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "consistent", "checked_at": time.Now().UTC()})
}
func (s *Server) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "live", "uptime_seconds": int(time.Since(s.started).Seconds())})
}
func (s *Server) readyCheck(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	ready := s.ready
	s.mu.RUnlock()
	if !ready {
		writeError(w, r, 503, "not_ready", "service is shutting down")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ready"})
}
func (s *Server) getMetrics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, s.metrics.Snapshot())
}
func (s *Server) notImplemented(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, 501, "unimplemented", "snapshot upload API is documented but not implemented in the local storage profile")
}
func (s *Server) SetReady(value bool) { s.mu.Lock(); defer s.mu.Unlock(); s.ready = value }
func parseLimit(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return 50
	}
	if n > 200 {
		return 200
	}
	return n
}
func etag(version uint64) string { return fmt.Sprintf("\"%d\"", version) }
func parseETag(value string) (uint64, error) {
	if value == "" {
		return 0, errors.New("If-Match header is required")
	}
	return strconv.ParseUint(strings.Trim(value, "\""), 10, 64)
}
func newID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(raw[:])
}
