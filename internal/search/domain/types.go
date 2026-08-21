package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type FieldType string

const (
	FieldText    FieldType = "text"
	FieldKeyword FieldType = "keyword"
	FieldNumber  FieldType = "number"
	FieldDate    FieldType = "date"
	FieldBool    FieldType = "bool"
)

type CollectionState string

const (
	CollectionCreating CollectionState = "creating"
	CollectionActive   CollectionState = "active"
	CollectionFrozen   CollectionState = "frozen"
	CollectionDeleting CollectionState = "deleting"
)

type Field struct {
	Name       string    `json:"name"`
	Type       FieldType `json:"type"`
	Analyzer   string    `json:"analyzer,omitempty"`
	Searchable bool      `json:"searchable"`
	Stored     bool      `json:"stored"`
	Sortable   bool      `json:"sortable"`
	Required   bool      `json:"required"`
	Boost      float64   `json:"boost,omitempty"`
}

func (f Field) Validate() error {
	if strings.TrimSpace(f.Name) == "" || strings.ContainsAny(f.Name, " ./") {
		return errors.New("field name is required and cannot contain spaces, dots, or slashes")
	}
	switch f.Type {
	case FieldText, FieldKeyword, FieldNumber, FieldDate, FieldBool:
	default:
		return fmt.Errorf("unsupported field type %q", f.Type)
	}
	if f.Type != FieldText && f.Analyzer != "" {
		return errors.New("analyzer is only valid for text fields")
	}
	if f.Boost < 0 || f.Boost > 100 {
		return errors.New("boost must be between 0 and 100")
	}
	return nil
}

type Schema struct {
	Version   uint64    `json:"version"`
	Fields    []Field   `json:"fields"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

func NewSchema(version uint64, fields []Field, actor string, now time.Time) (Schema, error) {
	if version == 0 || len(fields) == 0 || strings.TrimSpace(actor) == "" {
		return Schema{}, errors.New("schema version, fields, and actor are required")
	}
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if err := field.Validate(); err != nil {
			return Schema{}, fmt.Errorf("field %q: %w", field.Name, err)
		}
		if _, ok := seen[field.Name]; ok {
			return Schema{}, fmt.Errorf("duplicate field %q", field.Name)
		}
		seen[field.Name] = struct{}{}
	}
	return Schema{Version: version, Fields: append([]Field(nil), fields...), CreatedAt: now.UTC(), CreatedBy: actor}, nil
}

func (s Schema) Field(name string) (Field, bool) {
	for _, field := range s.Fields {
		if field.Name == name {
			return field, true
		}
	}
	return Field{}, false
}

type Collection struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	Name          string          `json:"name"`
	State         CollectionState `json:"state"`
	Schema        Schema          `json:"schema"`
	Shards        int             `json:"shards"`
	RetentionDays int             `json:"retention_days"`
	Version       uint64          `json:"version"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	CreatedBy     string          `json:"created_by"`
	UpdatedBy     string          `json:"updated_by"`
}

func NewCollection(id, tenant, name string, schema Schema, shards int, actor string, now time.Time) (Collection, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(tenant) == "" || strings.TrimSpace(name) == "" {
		return Collection{}, errors.New("id, tenant, and name are required")
	}
	if shards < 1 || shards > 128 {
		return Collection{}, errors.New("shards must be between 1 and 128")
	}
	return Collection{ID: id, TenantID: tenant, Name: name, State: CollectionActive, Schema: schema, Shards: shards, Version: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(), CreatedBy: actor, UpdatedBy: actor}, nil
}

func (c *Collection) ReplaceSchema(next Schema, expected uint64, actor string, now time.Time) error {
	if c.Version != expected {
		return fmt.Errorf("version conflict: current=%d expected=%d", c.Version, expected)
	}
	if next.Version != c.Schema.Version+1 {
		return errors.New("schema version must increment exactly once")
	}
	if c.State != CollectionActive && c.State != CollectionFrozen {
		return fmt.Errorf("cannot update schema in state %s", c.State)
	}
	c.Schema = next
	c.Version++
	c.UpdatedAt = now.UTC()
	c.UpdatedBy = actor
	return nil
}

type Document struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenant_id"`
	CollectionID    string         `json:"collection_id"`
	Fields          map[string]any `json:"fields"`
	ExternalVersion uint64         `json:"external_version"`
	SchemaVersion   uint64         `json:"schema_version"`
	Deleted         bool           `json:"deleted"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func (d Document) Validate(schema Schema, maxFields int) error {
	if d.ID == "" || d.TenantID == "" || d.CollectionID == "" {
		return errors.New("document id, tenant, and collection are required")
	}
	if len(d.Fields) > maxFields {
		return fmt.Errorf("document has %d fields, limit is %d", len(d.Fields), maxFields)
	}
	for _, field := range schema.Fields {
		value, ok := d.Fields[field.Name]
		if field.Required && (!ok || value == nil) {
			return fmt.Errorf("required field %q is missing", field.Name)
		}
		if ok {
			if err := validateValue(field, value); err != nil {
				return err
			}
		}
	}
	for name := range d.Fields {
		if _, ok := schema.Field(name); !ok {
			return fmt.Errorf("unknown field %q", name)
		}
	}
	return nil
}

func validateValue(field Field, value any) error {
	valid := false
	switch field.Type {
	case FieldText, FieldKeyword:
		_, valid = value.(string)
	case FieldNumber:
		switch value.(type) {
		case float64, float32, int, int64, int32, uint, uint64:
			valid = true
		}
	case FieldBool:
		_, valid = value.(bool)
	case FieldDate:
		switch value.(type) {
		case time.Time, string:
			valid = true
		}
	}
	if !valid {
		return fmt.Errorf("field %q does not match type %s", field.Name, field.Type)
	}
	return nil
}

type Posting struct {
	DocID     string `json:"doc_id"`
	Positions []int  `json:"positions"`
	Frequency int    `json:"frequency"`
}
type SegmentState string

const (
	SegmentBuilding    SegmentState = "building"
	SegmentReady       SegmentState = "ready"
	SegmentQuarantined SegmentState = "quarantined"
	SegmentMerged      SegmentState = "merged"
)

type Segment struct {
	ID            string       `json:"id"`
	CollectionID  string       `json:"collection_id"`
	Shard         int          `json:"shard"`
	Generation    uint64       `json:"generation"`
	State         SegmentState `json:"state"`
	DocumentCount int          `json:"document_count"`
	Checksum      string       `json:"checksum"`
	CreatedAt     time.Time    `json:"created_at"`
}
type Snapshot struct {
	ID           string    `json:"id"`
	CollectionID string    `json:"collection_id"`
	Generation   uint64    `json:"generation"`
	SegmentIDs   []string  `json:"segment_ids"`
	Checksum     string    `json:"checksum"`
	CreatedAt    time.Time `json:"created_at"`
}
