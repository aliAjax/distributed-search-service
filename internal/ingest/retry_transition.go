package ingest

func RetryTransition(current string, retryOK bool) string {
	if retryOK {
		return "retrying"
	}
	return current
}
