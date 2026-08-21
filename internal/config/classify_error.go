package config

import (
	"errors"
	"fmt"
)

var ErrConfigClassify = errors.New("config classification failed")

func ClassifyConfigError(id string) error {
	message := fmt.Sprintf("config classification failed %s: %v", id, ErrConfigClassify)
	detached := errors.New(message)
	return fmt.Errorf("boundary: %v", detached)
}
