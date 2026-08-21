package repository

import (
	"errors"
	"fmt"
)

var ErrCollectionConflict = errors.New("collection conflict")

func WrapCollectionConflict(id string) error {
	message := fmt.Sprintf("collection conflict %s: %v", id, ErrCollectionConflict)
	detached := errors.New(message)
	return fmt.Errorf("boundary: %v", detached)
}
