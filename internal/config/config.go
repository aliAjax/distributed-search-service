package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTP          HTTPConfig
	Storage       StorageConfig
	Limits        LimitsConfig
	Security      SecurityConfig
	Observability ObservabilityConfig
}
type HTTPConfig struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}
type StorageConfig struct {
	DataDir string
	WALPath string
}
type LimitsConfig struct {
	MaxBodyBytes        int64
	MaxFields           int
	MaxBulkItems        int
	MaxMutableDocuments int
	SearchConcurrency   int
}
type SecurityConfig struct{ CursorSigningKey string }
type ObservabilityConfig struct {
	LogLevel    string
	EnablePprof bool
}

func Default() Config {
	return Config{HTTP: HTTPConfig{Address: ":8080", ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, ShutdownTimeout: 10 * time.Second}, Storage: StorageConfig{DataDir: "./data", WALPath: "./data/index.wal"}, Limits: LimitsConfig{MaxBodyBytes: 8 << 20, MaxFields: 128, MaxBulkItems: 1000, MaxMutableDocuments: 100, SearchConcurrency: 32}, Security: SecurityConfig{CursorSigningKey: "development-only-change-me-32bytes"}, Observability: ObservabilityConfig{LogLevel: "info"}}
}
func Load(path string) (Config, error) {
	cfg := Default()
	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return Config{}, err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		section := ""
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasSuffix(line, ":") && !strings.Contains(strings.TrimSuffix(line, ":"), " ") {
				section = strings.TrimSuffix(line, ":")
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				return Config{}, fmt.Errorf("invalid config line %q", line)
			}
			apply(&cfg, section+"."+strings.TrimSpace(parts[0]), strings.Trim(strings.TrimSpace(parts[1]), "\"'"))
		}
		if err = scanner.Err(); err != nil {
			return Config{}, err
		}
	}
	applyEnv(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
func apply(c *Config, key, value string) {
	switch key {
	case "http.address":
		c.HTTP.Address = value
	case "http.read_timeout":
		c.HTTP.ReadTimeout = parseDuration(value, c.HTTP.ReadTimeout)
	case "http.write_timeout":
		c.HTTP.WriteTimeout = parseDuration(value, c.HTTP.WriteTimeout)
	case "http.shutdown_timeout":
		c.HTTP.ShutdownTimeout = parseDuration(value, c.HTTP.ShutdownTimeout)
	case "storage.data_dir":
		c.Storage.DataDir = value
	case "storage.wal_path":
		c.Storage.WALPath = value
	case "limits.max_body_bytes":
		c.Limits.MaxBodyBytes = parseInt64(value, c.Limits.MaxBodyBytes)
	case "limits.max_fields":
		c.Limits.MaxFields = parseInt(value, c.Limits.MaxFields)
	case "limits.max_bulk_items":
		c.Limits.MaxBulkItems = parseInt(value, c.Limits.MaxBulkItems)
	case "limits.max_mutable_documents":
		c.Limits.MaxMutableDocuments = parseInt(value, c.Limits.MaxMutableDocuments)
	case "limits.search_concurrency":
		c.Limits.SearchConcurrency = parseInt(value, c.Limits.SearchConcurrency)
	case "security.cursor_signing_key":
		c.Security.CursorSigningKey = value
	case "observability.log_level":
		c.Observability.LogLevel = value
	case "observability.enable_pprof":
		c.Observability.EnablePprof = value == "true"
	}
}
func applyEnv(c *Config) {
	mapping := map[string]string{"SEARCH_HTTP_ADDRESS": "http.address", "SEARCH_DATA_DIR": "storage.data_dir", "SEARCH_WAL_PATH": "storage.wal_path", "SEARCH_CURSOR_SIGNING_KEY": "security.cursor_signing_key", "SEARCH_LOG_LEVEL": "observability.log_level"}
	for env, key := range mapping {
		if value, ok := os.LookupEnv(env); ok {
			apply(c, key, value)
		}
	}
}
func (c Config) Validate() error {
	if c.HTTP.Address == "" {
		return errors.New("HTTP address required")
	}
	if c.Storage.WALPath == "" {
		return errors.New("WAL path required")
	}
	if len(c.Security.CursorSigningKey) < 24 {
		return errors.New("cursor signing key must be at least 24 characters")
	}
	if c.Limits.MaxFields < 1 || c.Limits.MaxBulkItems < 1 || c.Limits.SearchConcurrency < 1 {
		return errors.New("limits must be positive")
	}
	return nil
}
func parseDuration(v string, fallback time.Duration) time.Duration {
	n, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return n
}
func parseInt(v string, fallback int) int {
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
func parseInt64(v string, fallback int64) int64 {
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}
