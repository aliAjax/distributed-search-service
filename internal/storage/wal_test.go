package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestWALAppendReplayAndChecksum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal", "index.wal")
	wal, err := OpenWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	if _, err = wal.Append(context.Background(), Record{Operation: OperationUpsert, TenantID: "t", CollectionID: "c", DocumentID: "d", Version: 1}); err != nil {
		t.Fatal(err)
	}
	var got []Record
	if err = wal.Replay(context.Background(), 0, func(r Record) error { got = append(got, r); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Sequence != 1 {
		t.Fatalf("records=%v", got)
	}
	if err = wal.Close(); err != nil {
		t.Fatal(err)
	}
	wal, err = OpenWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	if _, err = wal.Append(context.Background(), Record{Operation: OperationDelete, TenantID: "t", CollectionID: "c", DocumentID: "d", Version: 2}); err != nil {
		t.Fatal(err)
	}
	if wal.sequence != 2 {
		t.Fatalf("sequence=%d", wal.sequence)
	}
}
