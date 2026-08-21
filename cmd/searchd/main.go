package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"example.com/distributed-search-service/internal/analyzer"
	"example.com/distributed-search-service/internal/config"
	"example.com/distributed-search-service/internal/ingest"
	"example.com/distributed-search-service/internal/maintenance"
	"example.com/distributed-search-service/internal/observability"
	"example.com/distributed-search-service/internal/repository"
	"example.com/distributed-search-service/internal/storage"
	"example.com/distributed-search-service/internal/transport/httpapi"
)

func main() {
	if err := run(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.Load(os.Getenv("SEARCH_CONFIG"))
	if err != nil {
		return err
	}
	if err = os.MkdirAll(cfg.Storage.DataDir, 0750); err != nil {
		return err
	}
	wal, err := storage.OpenWAL(cfg.Storage.WALPath)
	if err != nil {
		return err
	}
	defer wal.Close()
	logger := observability.NewLogger(cfg.Observability.LogLevel)
	metrics := &observability.Metrics{}
	collections := repository.NewMemoryCollections()
	ids := repository.NewIdempotencyStore()
	analyzers := analyzer.NewRegistry()
	if err = analyzers.Put(analyzer.Pipeline{Name: "standard", Version: 1, Lowercase: true, Stopwords: []string{"the", "a", "an", "的", "了"}, Synonyms: map[string][]string{"golang": {"go"}, "汽车": {"车辆"}}}); err != nil {
		return err
	}
	ingestService := ingest.New(collections, wal, cfg.Limits.MaxFields, cfg.Limits.MaxBulkItems)
	maintenanceService := maintenance.New()
	api := httpapi.New(httpapi.Dependencies{Collections: collections, Idempotency: ids, Analyzers: analyzers, Ingest: ingestService, Maintenance: maintenanceService, Logger: logger, Metrics: metrics, MaxBodyBytes: cfg.Limits.MaxBodyBytes, SearchConcurrency: cfg.Limits.SearchConcurrency})
	server := &http.Server{Addr: cfg.HTTP.Address, Handler: api.Handler(), ReadTimeout: cfg.HTTP.ReadTimeout, WriteTimeout: cfg.HTTP.WriteTimeout, IdleTimeout: 60 * 1e9}
	go func() {
		logger.Info("searchd_started", "address", cfg.HTTP.Address)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("searchd_failed", "error", serveErr)
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	api.SetReady(false)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	return server.Shutdown(ctx)
}
