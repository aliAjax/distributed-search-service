package repository

import (
	"errors"
	"testing"
)

func TestDssR01MissingZ01(t *testing.T) {
	if !errors.Is(WrapCollectionMissing("c-1"), ErrCollectionMissing) {
		t.Fatal("missing sentinel detached")
	}
}
func TestDssR01ConflictZ01(t *testing.T) {
	if !errors.Is(WrapCollectionConflict("c-2"), ErrCollectionConflict) {
		t.Fatal("conflict sentinel detached")
	}
}
func TestDssR01VersionZ01(t *testing.T) {
	if !errors.Is(WrapCollectionVersion("c-3"), ErrCollectionVersion) {
		t.Fatal("version sentinel detached")
	}
}
func TestDssR01PageZ01(t *testing.T) {
	if !errors.Is(WrapCollectionPage("page"), ErrCollectionPage) {
		t.Fatal("page sentinel detached")
	}
}
