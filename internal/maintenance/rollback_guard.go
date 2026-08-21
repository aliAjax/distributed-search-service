package maintenance

import "errors"

func RunWithRollback(operation, rollback func() error) error {
	if err := operation(); err != nil {
		return errors.Join(err, rollback())
	}
	return nil
}
