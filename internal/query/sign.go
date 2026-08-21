package query

import (
	"crypto/hmac"
	"crypto/sha256"
)

func sign(value, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(value)
	return h.Sum(nil)
}
func equal(a, b []byte) bool { return hmac.Equal(a, b) }
