// Package output provides the Leamas output contract for factory commands.
// All factory commands must emit bounded, deterministic, parseable output.
package output

import "fmt"

// Field represents a key-value field in output.
type Field struct {
	Key   string
	Value any
}

// Failure represents a single failure in a check result.
type Failure struct {
	Kind    string
	Message string
}

// Result represents the canonical output for a factory check command.
// All factory commands should produce output conforming to this structure.
type Result struct {
	OK       bool      // true if check passed
	Check    string    // check name (e.g., "coverage", "dupcode")
	Fields   []Field   // key-value fields (sorted by Key)
	Artifact string    // optional path to detailed artifact file
	Failures []Failure // bounded list of failures (max 5)
}

// ExitCode returns the canonical exit code for this result.
func (r Result) ExitCode() int {
	if r.OK {
		return 0
	}
	return 1
}

// Summary returns a one-line summary suitable for human output.
func (r Result) Summary() string {
	if r.Artifact != "" {
		return fmt.Sprintf("FAIL %s artifact=%s", r.Check, r.Artifact)
	}
	return fmt.Sprintf("%s: FAIL", r.Check)
}

// String implements fmt.Stringer for the human-readable line output.
func (r Result) String() string {
	return RenderLine(r)
}

// JSON returns the JSON representation suitable for --json output.
func (r Result) JSON() ([]byte, error) {
	return RenderJSON(r)
}

// NewResult creates a new Result with the given check name.
func NewResult(check string) *Result {
	return &Result{
		Check:    check,
		Fields:   make([]Field, 0),
		Artifact: "",
		Failures: make([]Failure, 0),
	}
}

// SetOK marks the result as passing.
func (r *Result) SetOK() {
	r.OK = true
}

// AddField adds a field to the result. Fields are sorted by key for determinism.
func (r *Result) AddField(key string, value any) {
	r.Fields = append(r.Fields, Field{Key: key, Value: value})
}

// SetArtifact sets the artifact path for detailed failure output.
func (r *Result) SetArtifact(path string) {
	r.Artifact = path
}

// AddFailure adds a failure to the result. Only the first 5 failures are kept.
func (r *Result) AddFailure(kind, message string) {
	if len(r.Failures) >= 5 {
		return // Bounded failure list
	}
	r.Failures = append(r.Failures, Failure{Kind: kind, Message: message})
	r.OK = false
}
