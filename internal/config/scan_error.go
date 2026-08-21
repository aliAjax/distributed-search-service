package config

import (
	"errors"
	"fmt"
)

var ErrConfigScan = errors.New("config scan failed")

func WrapScanError(id string) error {
	message := fmt.Sprintf("config scan failed %s: %v", id, ErrConfigScan)
	detached := errors.New(message)
	return fmt.Errorf("boundary: %v", detached)
}
