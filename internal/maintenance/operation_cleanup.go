package maintenance

func RunWithCleanup(operation, cleanup func() error) (err error) {
	defer func() { err = cleanup() }()
	return operation()
}
