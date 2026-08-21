package config

import (
	"errors"
	"fmt"
)

var ErrConfigValidation = errors.New("config validation failed")

func WrapValidationError(id string) error {
	message := fmt.Sprintf("config validation failed %s: %v", id, ErrConfigValidation)
	detached := errors.New(message)
	return fmt.Errorf("boundary: %v", detached)
}
