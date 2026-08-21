package repository

import (
	"errors"
	"fmt"
)

var ErrCollectionVersion = errors.New("collection version mismatch")

func WrapCollectionVersion(id string) error {
	return fmt.Errorf("boundary: collection version mismatch %s: %w", id, ErrCollectionVersion)
}
