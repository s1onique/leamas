// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence_strict_decode_test.go
// is the B2-R2 strict single-document decoder matrix.
//
// The matrix proves the B2-R2 contract:
//
//	valid document                 -> PASS
//	unknown field                  -> reject
//	second JSON object             -> reject
//	second scalar                  -> reject
//	trailing garbage               -> reject
//	truncated JSON                 -> reject
//	whitespace after one document  -> PASS
//
// B2-R3 additions:
//
//	duplicate top-level known field -> reject
//	duplicate nested known field   -> reject
//
// The strict decoder uses Decoder.DisallowUnknownFields
// for unknown-field rejection and a custom duplicate-field
// scanner (in plancontract.StrictDecode) for duplicate-key
// rejection. The Go team's v1 encoding/json decoder does not
// reject duplicate keys, so the scanner catches the third
// class of malformed input.
package evidence

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestClosureEvidenceStrictSingleDocumentDecode is the
// matrix required by the B2-R2 strict single-document
// contract. Each row asserts either the decoder accepts the
// input (PASS) or that it returns a non-nil error (reject).
func TestClosureEvidenceStrictSingleDocumentDecode(t *testing.T) {
	t.Parallel()

	// validBase returns a fully valid closure-evidence
	// JSON document that the matrix mutates. The same
	// base is used for every row; only the mutation
	// function changes.
	validBase := func() []byte {
		c := validCandidate()
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatalf("marshal valid candidate: %v", err)
		}
		return b
	}

	type row struct {
		name    string
		mutate  func([]byte) []byte
		wantErr bool
	}

	rows := []row{
		{
			name:    "valid document",
			mutate:  func(b []byte) []byte { return b },
			wantErr: false,
		},
		{
			name: "unknown field",
			mutate: func(b []byte) []byte {
				// Inject an unknown top-level field.
				return []byte(`{"undeclared":"x",` + string(b[1:]))
			},
			wantErr: true,
		},
		{
			name: "second JSON object",
			mutate: func(b []byte) []byte {
				// Two objects separated by a single space.
				return append(b, ' ', '{', '}')
			},
			wantErr: true,
		},
		{
			name: "second scalar",
			mutate: func(b []byte) []byte {
				// Document followed by a stray scalar.
				return append(b, ' ', '4', '2')
			},
			wantErr: true,
		},
		{
			name: "trailing garbage",
			mutate: func(b []byte) []byte {
				// Non-JSON characters after the document.
				return append(b, ' ', 'g', 'a', 'r', 'b', 'a', 'g', 'e')
			},
			wantErr: true,
		},
		{
			name: "truncated JSON",
			mutate: func(b []byte) []byte {
				// Drop the closing brace.
				return b[:len(b)-1]
			},
			wantErr: true,
		},
		{
			name:    "whitespace after one document",
			mutate:  func(b []byte) []byte { return append(b, ' ', '\n', '\t') },
			wantErr: false,
		},
		{
			name: "duplicate top-level known field",
			mutate: func(b []byte) []byte {
				// Two schema_version keys at the top level.
				// The B2-R3 strict scanner rejects duplicates
				// even when the keys are well-known. We replace
				// the closing brace of the document with another
				// schema_version key.
				b[len(b)-1] = ' '
				return append(b, []byte(`"schema_version":4}`)...)
			},
			wantErr: true,
		},
		{
			name: "duplicate nested known field",
			mutate: func(b []byte) []byte {
				// Two checks keys inside the runtime sub-object.
				// Pick a candidate that has a checks-shaped field
				// and inject a duplicate.
				b2 := strings.Replace(string(b), `"runtime":{`, `"runtime":{"checks":null,`, 1)
				return []byte(b2)
			},
			wantErr: true,
		},
	}

	for _, r := range rows {
		r := r
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			bytes := r.mutate(validBase())
			_, err := UnmarshalClosureEvidence(bytes)
			if r.wantErr && err == nil {
				t.Fatalf("strict decoder must reject %s, got nil", r.name)
			}
			if !r.wantErr && err != nil {
				t.Fatalf("strict decoder must accept %s, got %v", r.name, err)
			}
		})
	}
}
