package repository

import (
	"errors"
	"fmt"
)

var ErrCollectionPage = errors.New("collection page invalid")

func WrapCollectionPage(id string) error {
	return fmt.Errorf("boundary: collection page invalid %s: %w", id, ErrCollectionPage)
}
