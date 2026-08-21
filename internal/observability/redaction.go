package observability

import "strings"

func Redact(value string) string {
	if value == "" {
		return value
	}
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}
func SafeFields(fields map[string]any, sensitive map[string]bool) map[string]any {
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		if sensitive[key] {
			out[key] = "[REDACTED]"
		} else {
			out[key] = value
		}
	}
	return out
}
