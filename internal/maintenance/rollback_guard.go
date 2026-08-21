package maintenance

func RunWithRollback(operation, rollback func() error) error {
	if err := operation(); err != nil {
		return err
	}
	return nil
}
