// Package runbundle provides local run bundle creation and validation for
// Leamas verification witness evidence.
package runbundle

import (
	"bytes"
	"encoding/json"
	"time"
)

// SchemaVersion is the current metadata schema version.
const SchemaVersion = "leamas.runbundle.v1"

// RunID identifies a run bundle.
type RunID string

// Metadata holds the structured metadata for a run bundle.
type Metadata struct {
	SchemaVersion string    `json:"schema_version"`
	RunID         RunID     `json:"run_id"`
	CreatedAt     time.Time `json:"created_at"`
	Tool          ToolInfo  `json:"tool"`
	Doctrine      Doctrine  `json:"doctrine"`
}

// ToolInfo describes the tool that created the run bundle.
type ToolInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// Doctrine records the doctrine flags at bundle creation time.
type Doctrine struct {
	LocalOnly  bool `json:"local_only"`
	ReadOnly   bool `json:"read_only"`
	NoDatabase bool `json:"no_database"`
}

// NewMetadata creates a new Metadata with default doctrine flags.
func NewMetadata(runID RunID, now time.Time, toolName, toolVersion string) Metadata {
	return Metadata{
		SchemaVersion: SchemaVersion,
		RunID:         runID,
		CreatedAt:     now,
		Tool: ToolInfo{
			Name:    toolName,
			Version: toolVersion,
		},
		Doctrine: Doctrine{
			LocalOnly:  true,
			ReadOnly:   true,
			NoDatabase: true,
		},
	}
}

// MarshalJSON serializes Metadata to JSON with deterministic ordering.
func (m Metadata) MarshalJSON() ([]byte, error) {
	type metadataAlias Metadata
	return json.Marshal(struct {
		CreatedAt string `json:"created_at"`
		*metadataAlias
	}{
		CreatedAt:     m.CreatedAt.Format(time.RFC3339Nano),
		metadataAlias: (*metadataAlias)(&m),
	})
}

// StrictDecode decodes JSON into Metadata, rejecting unknown fields.
func StrictDecode(data []byte) (*Metadata, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var m Metadata
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}
