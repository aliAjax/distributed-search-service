package maintenance

import "errors"

func RunWithCleanup(operation, cleanup func() error) (err error) {
	defer func() { err = errors.Join(err, cleanup()) }()
	return operation()
}
