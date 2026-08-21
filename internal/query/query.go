package query

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Kind string

const (
	KindMatch  Kind = "match"
	KindPhrase Kind = "phrase"
	KindPrefix Kind = "prefix"
	KindFuzzy  Kind = "fuzzy"
	KindRange  Kind = "range"
	KindBool   Kind = "bool"
)

type Clause struct {
	Kind    Kind     `json:"kind"`
	Field   string   `json:"field,omitempty"`
	Value   string   `json:"value,omitempty"`
	From    any      `json:"from,omitempty"`
	To      any      `json:"to,omitempty"`
	Must    []Clause `json:"must,omitempty"`
	Should  []Clause `json:"should,omitempty"`
	MustNot []Clause `json:"must_not,omitempty"`
	Boost   float64  `json:"boost,omitempty"`
}

type Sort struct {
	Field string `json:"field"`
	Desc  bool   `json:"desc"`
}
type Request struct {
	TenantID     string   `json:"-"`
	CollectionID string   `json:"collection_id"`
	Query        Clause   `json:"query"`
	Filters      []Clause `json:"filters,omitempty"`
	Sort         []Sort   `json:"sort,omitempty"`
	Size         int      `json:"size,omitempty"`
	SearchAfter  string   `json:"search_after,omitempty"`
	Explain      bool     `json:"explain,omitempty"`
	TimeoutMS    int      `json:"timeout_ms,omitempty"`
	AllowPartial bool     `json:"allow_partial,omitempty"`
}

func (r *Request) Defaults() {
	if r.Size == 0 {
		r.Size = 10
	}
	if r.TimeoutMS == 0 {
		r.TimeoutMS = 1000
	}
}
func (r Request) Validate() error {
	if r.TenantID == "" || r.CollectionID == "" {
		return errors.New("tenant and collection are required")
	}
	if r.Size < 1 || r.Size > 1000 {
		return errors.New("size must be between 1 and 1000")
	}
	if r.TimeoutMS < 10 || r.TimeoutMS > 30000 {
		return errors.New("timeout_ms must be between 10 and 30000")
	}
	if err := validateClause(r.Query, 0); err != nil {
		return fmt.Errorf("query: %w", err)
	}
	for i, c := range r.Filters {
		if err := validateClause(c, 0); err != nil {
			return fmt.Errorf("filter %d: %w", i, err)
		}
	}
	return nil
}
func validateClause(c Clause, depth int) error {
	if depth > 10 {
		return errors.New("query nesting exceeds 10")
	}
	switch c.Kind {
	case KindMatch, KindPhrase, KindPrefix, KindFuzzy:
		if c.Field == "" || strings.TrimSpace(c.Value) == "" {
			return errors.New("text clause requires field and value")
		}
	case KindRange:
		if c.Field == "" || (c.From == nil && c.To == nil) {
			return errors.New("range requires field and bound")
		}
	case KindBool:
		if len(c.Must)+len(c.Should)+len(c.MustNot) == 0 {
			return errors.New("bool query must contain clauses")
		}
		for _, child := range append(append(append([]Clause{}, c.Must...), c.Should...), c.MustNot...) {
			if err := validateClause(child, depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported query kind %q", c.Kind)
	}
	return nil
}

type cursorPayload struct {
	Values     []any  `json:"values"`
	DocID      string `json:"doc_id"`
	Generation uint64 `json:"generation"`
	Expires    int64  `json:"expires"`
}
type CursorCodec struct{ key []byte }

func NewCursorCodec(key string) *CursorCodec { return &CursorCodec{key: []byte(key)} }
func (c *CursorCodec) Encode(values []any, docID string, generation uint64, expiry time.Time) (string, error) {
	p := cursorPayload{Values: values, DocID: docID, Generation: generation, Expires: expiry.Unix()}
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	sig := sign(raw, c.key)
	return base64.RawURLEncoding.EncodeToString(append(raw, sig...)), nil
}
func (c *CursorCodec) Decode(value string, now time.Time) ([]any, string, uint64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(raw) < 32 {
		return nil, "", 0, errors.New("invalid search-after cursor")
	}
	body, sig := raw[:len(raw)-32], raw[len(raw)-32:]
	expected := sign(body, c.key)
	if !equal(sig, expected) {
		return nil, "", 0, errors.New("search-after cursor signature mismatch")
	}
	var p cursorPayload
	if err = json.Unmarshal(body, &p); err != nil {
		return nil, "", 0, errors.New("invalid search-after cursor payload")
	}
	if now.Unix() > p.Expires {
		return nil, "", 0, errors.New("search-after cursor expired")
	}
	return p.Values, p.DocID, p.Generation, nil
}
