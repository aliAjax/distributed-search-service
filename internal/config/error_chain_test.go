package config

import (
	"errors"
	"testing"
)

func TestDssR10OpenZ10(t *testing.T) {
	if !errors.Is(WrapOpenError("missing"), ErrConfigOpen) {
		t.Fatal("open error detached")
	}
}
func TestDssR10ScanZ10(t *testing.T) {
	if !errors.Is(WrapScanError("bad"), ErrConfigScan) {
		t.Fatal("scan error detached")
	}
}
func TestDssR10ValidateZ10(t *testing.T) {
	if !errors.Is(WrapValidationError("bad"), ErrConfigValidation) {
		t.Fatal("validation error detached")
	}
}
func TestDssR10ClassifyZ10(t *testing.T) {
	if !errors.Is(ClassifyConfigError("bad"), ErrConfigClassify) {
		t.Fatal("classification error detached")
	}
}
