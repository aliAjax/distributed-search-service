package index

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
)

type SegmentHeader struct {
	Magic         string `json:"magic"`
	Version       uint16 `json:"version"`
	Generation    uint64 `json:"generation"`
	DocumentCount int    `json:"document_count"`
}

func EncodeHeader(w io.Writer, header SegmentHeader) error { return json.NewEncoder(w).Encode(header) }
func DecodeHeader(r io.Reader) (SegmentHeader, error) {
	var header SegmentHeader
	err := json.NewDecoder(bufio.NewReader(r)).Decode(&header)
	return header, err
}
func ChecksumBytes(data []byte) string                 { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func VerifyChecksum(data []byte, expected string) bool { return ChecksumBytes(data) == expected }
