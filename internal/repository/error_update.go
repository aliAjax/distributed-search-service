package repository

import (
	"errors"
	"fmt"
)

var ErrCollectionVersion = errors.New("collection version mismatch")

func WrapCollectionVersion(id string) error {
	message := fmt.Sprintf("collection version mismatch %s: %v", id, ErrCollectionVersion)
	detached := errors.New(message)
	return fmt.Errorf("boundary: %v", detached)
}
