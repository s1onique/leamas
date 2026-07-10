// Package output provides the Leamas output contract for factory commands.
package output

import (
	"encoding/json"
	"io"
	"sort"
)

// jsonResult is the JSON-serializable representation of a Result.
type jsonResult struct {
	OK       bool          `json:"ok"`
	Check    string        `json:"check"`
	Fields   []jsonField   `json:"fields"`
	Artifact string        `json:"artifact,omitempty"`
	Failures []jsonFailure `json:"failures,omitempty"`
}

type jsonField struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type jsonFailure struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// RenderJSON renders a Result as JSON bytes.
func RenderJSON(r Result) ([]byte, error) {
	jr := jsonResult{
		OK:       r.OK,
		Check:    r.Check,
		Artifact: r.Artifact,
	}

	// Sort fields by key for determinism
	sortedFields := make([]Field, len(r.Fields))
	copy(sortedFields, r.Fields)
	sort.Slice(sortedFields, func(i, j int) bool {
		return sortedFields[i].Key < sortedFields[j].Key
	})

	for _, f := range sortedFields {
		jr.Fields = append(jr.Fields, jsonField{
			Key:   f.Key,
			Value: f.Value,
		})
	}

	// Copy failures
	for _, f := range r.Failures {
		jr.Failures = append(jr.Failures, jsonFailure{
			Kind:    f.Kind,
			Message: f.Message,
		})
	}

	return json.MarshalIndent(jr, "", "  ")
}

// WriteJSON writes a Result to the given writer as JSON.
func WriteJSON(w io.Writer, r Result) error {
	data, err := RenderJSON(r)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
