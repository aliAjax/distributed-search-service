package config

import (
	"errors"
	"fmt"
)

var ErrConfigOpen = errors.New("config open failed")

func WrapOpenError(id string) error {
	message := fmt.Sprintf("config open failed %s: %v", id, ErrConfigOpen)
	detached := errors.New(message)
	return fmt.Errorf("boundary: %v", detached)
}
