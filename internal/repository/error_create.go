package repository

import (
	"errors"
	"fmt"
)

var ErrCollectionConflict = errors.New("collection conflict")

func WrapCollectionConflict(id string) error {
	return fmt.Errorf("boundary: collection conflict %s: %w", id, ErrCollectionConflict)
}
