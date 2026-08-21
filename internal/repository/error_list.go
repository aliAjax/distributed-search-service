package repository

import (
	"errors"
	"fmt"
)

var ErrCollectionPage = errors.New("collection page invalid")

func WrapCollectionPage(id string) error {
	message := fmt.Sprintf("collection page invalid %s: %v", id, ErrCollectionPage)
	detached := errors.New(message)
	return fmt.Errorf("boundary: %v", detached)
}
