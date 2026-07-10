// Package output provides the Leamas output contract for factory commands.
package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// RenderLine renders a Result as a one-line human-readable output.
// Format: "check: key=value key=value ... OK" or "check: key=value FAIL\nFAIL key=value\nFAIL ..."
func RenderLine(r Result) string {
	if r.OK {
		return renderOKLine(r)
	}
	return renderFailLines(r)
}

func renderOKLine(r Result) string {
	var parts []string

	// Check name as prefix
	parts = append(parts, r.Check+":")

	// Add sorted fields
	sortedFields := make([]Field, len(r.Fields))
	copy(sortedFields, r.Fields)
	sort.Slice(sortedFields, func(i, j int) bool {
		return sortedFields[i].Key < sortedFields[j].Key
	})

	for _, f := range sortedFields {
		parts = append(parts, formatField(f))
	}

	parts = append(parts, "OK")
	return strings.Join(parts, " ")
}

func renderFailLines(r Result) string {
	var parts []string

	// Check name as prefix
	parts = append(parts, r.Check+":")

	// Add sorted fields
	sortedFields := make([]Field, len(r.Fields))
	copy(sortedFields, r.Fields)
	sort.Slice(sortedFields, func(i, j int) bool {
		return sortedFields[i].Key < sortedFields[j].Key
	})

	for _, f := range sortedFields {
		parts = append(parts, formatField(f))
	}

	parts = append(parts, "FAIL")
	var lines []string
	lines = append(lines, strings.Join(parts, " "))

	// Render bounded failures
	for _, f := range r.Failures {
		lines = append(lines, fmt.Sprintf("FAIL %s %s", f.Kind, f.Message))
	}

	// If there's an artifact, mention it
	if r.Artifact != "" {
		lines = append(lines, fmt.Sprintf("artifact=%s", r.Artifact))
	}

	return strings.Join(lines, "\n")
}

func formatField(f Field) string {
	return fmt.Sprintf("%s=%v", f.Key, formatValue(f.Value))
}

func formatValue(v any) string {
	switch val := v.(type) {
	case float64:
		// Format floats consistently
		if val == float64(int(val)) {
			return fmt.Sprintf("%.0f", val)
		}
		return fmt.Sprintf("%.1f", val)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// WriteLine writes a Result to the given writer in human-readable format.
func WriteLine(w io.Writer, r Result) {
	fmt.Fprintln(w, RenderLine(r))
}
