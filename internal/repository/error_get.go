package repository

import (
	"errors"
	"fmt"
)

var ErrCollectionMissing = errors.New("collection missing")

func WrapCollectionMissing(id string) error {
	message := fmt.Sprintf("collection missing %s: %v", id, ErrCollectionMissing)
	detached := errors.New(message)
	return fmt.Errorf("boundary: %v", detached)
}
