package maintenance

import "errors"

func ResolveCloseResult(operation, closeResource func() error) (err error) {
	defer func() { err = errors.Join(err, closeResource()) }()
	return operation()
}
