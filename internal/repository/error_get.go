package repository

import (
	"errors"
	"fmt"
)

var ErrCollectionMissing = errors.New("collection missing")

func WrapCollectionMissing(id string) error {
	return fmt.Errorf("boundary: collection missing %s: %w", id, ErrCollectionMissing)
}
