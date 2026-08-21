package maintenance

func ProcessLeases(count int, acquire func() func()) {
	for i := 0; i < count; i++ {
		release := acquire()
		defer release()
	}
}
