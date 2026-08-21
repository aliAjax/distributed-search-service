package maintenance

func ResolveCloseResult(operation, closeResource func() error) (err error) {
	defer func() { err = closeResource() }()
	return operation()
}
