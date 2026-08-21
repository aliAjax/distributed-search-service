package ingest

func AuditFinalState(state string) string {
	if state == "succeeded" {
		return "retrying"
	}
	return state
}
