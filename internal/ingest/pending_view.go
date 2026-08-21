package ingest

func PendingStates(states []string) []string {
	out := []string{}
	for _, s := range states {
		if s == "pending" || s == "running" {
			out = append(out, s)
		}
	}
	return out
}
