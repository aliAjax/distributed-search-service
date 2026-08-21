package storage

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Operation string

const (
	OperationUpsert Operation = "upsert"
	OperationDelete Operation = "delete"
)

type Record struct {
	Sequence     uint64          `json:"sequence"`
	Operation    Operation       `json:"operation"`
	TenantID     string          `json:"tenant_id"`
	CollectionID string          `json:"collection_id"`
	DocumentID   string          `json:"document_id"`
	Version      uint64          `json:"version"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	Timestamp    time.Time       `json:"timestamp"`
	Checksum     string          `json:"checksum"`
}

func (r *Record) calculateChecksum() string {
	copy := *r
	copy.Checksum = ""
	raw, _ := json.Marshal(copy)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (r *Record) Seal() { r.Checksum = r.calculateChecksum() }
func (r Record) Validate() error {
	if r.Sequence == 0 || r.TenantID == "" || r.CollectionID == "" || r.DocumentID == "" {
		return errors.New("invalid WAL record identity")
	}
	if r.Operation != OperationUpsert && r.Operation != OperationDelete {
		return errors.New("invalid WAL operation")
	}
	if r.Checksum != r.calculateChecksum() {
		return errors.New("WAL checksum mismatch")
	}
	return nil
}

type WAL struct {
	mu       sync.Mutex
	file     *os.File
	path     string
	sequence uint64
}

func OpenWAL(path string) (*WAL, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	w := &WAL{file: file, path: path}
	if err = w.scanSequence(); err != nil {
		file.Close()
		return nil, err
	}
	return w, nil
}
func (w *WAL) scanSequence() error {
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	scanner := bufio.NewScanner(w.file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var rec Record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			return fmt.Errorf("decode WAL: %w", err)
		}
		if err := rec.Validate(); err != nil {
			return err
		}
		if rec.Sequence > w.sequence {
			w.sequence = rec.Sequence
		}
	}
	_, err := w.file.Seek(0, io.SeekEnd)
	return err
}
func (w *WAL) Append(ctx context.Context, record Record) (uint64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	w.sequence++
	record.Sequence = w.sequence
	record.Timestamp = time.Now().UTC()
	record.Seal()
	raw, err := json.Marshal(record)
	if err != nil {
		return 0, err
	}
	raw = append(raw, '\n')
	if _, err = w.file.Write(raw); err != nil {
		return 0, err
	}
	if err = w.file.Sync(); err != nil {
		return 0, err
	}
	return record.Sequence, nil
}
func (w *WAL) Replay(ctx context.Context, after uint64, apply func(Record) error) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	file, err := os.Open(w.path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var rec Record
		if err = json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			return err
		}
		if err = rec.Validate(); err != nil {
			return err
		}
		if rec.Sequence > after {
			if err = apply(rec); err != nil {
				return fmt.Errorf("apply WAL sequence %d: %w", rec.Sequence, err)
			}
		}
	}
	return scanner.Err()
}
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

type SnapshotManifest struct {
	ID         string         `json:"id"`
	Generation uint64         `json:"generation"`
	Files      []SnapshotFile `json:"files"`
	CreatedAt  time.Time      `json:"created_at"`
	Checksum   string         `json:"checksum"`
}
type SnapshotFile struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type ObjectStore interface {
	Put(context.Context, string, io.Reader) error
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}
type FileStore struct{ Root string }

func (s FileStore) path(key string) (string, error) {
	clean := filepath.Clean(key)
	if clean == "." || filepath.IsAbs(clean) || clean[:1] == "." {
		return "", errors.New("unsafe object key")
	}
	return filepath.Join(s.Root, clean), nil
}
func (s FileStore) Put(ctx context.Context, key string, reader io.Reader) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	tmp := path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, copyErr := copyContext(ctx, file, reader)
	syncErr := file.Sync()
	closeErr := file.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp, path)
}
func (s FileStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}
func (s FileStore) Delete(_ context.Context, key string) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
func copyContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 64*1024)
	var total int64
	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}
		n, readErr := src.Read(buf)
		if n > 0 {
			written, err := dst.Write(buf[:n])
			total += int64(written)
			if err != nil {
				return total, err
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}
