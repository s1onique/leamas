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
// The strict decoder uses json.Decoder.DisallowUnknownFields
// for "STRICT_UNKNOWN_FIELDS=true" and a second Decode that
// MUST return io.EOF for "STRICT_SINGLE_DOCUMENT=true".
// Decoder.More() is the wrong primitive for the second check
// because it is defined for arrays and objects, not for the
// top-level document.
package evidence

import (
	"encoding/json"
	"testing"
)

// TestClosureEvidenceStrictSingleDocumentDecode is the
// matrix required by the B2-R2 strict single-document
// contract. Each row asserts either the decoder accepts the
// input (PASS) or that it returns a non-nil error (reject).
func TestClosureEvidenceStrictSingleDocumentDecode(t *testing.T) {
	t.Parallel()

	// validMarshalableCandidate returns a fully valid
	// closure-evidence JSON document ready for round-trip
	// decode. The same base is mutated for the negative
	// rows.
	validBytes := func() []byte {
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
			name: "whitespace after one document",
			mutate: func(b []byte) []byte {
				// Trailing whitespace is permitted by
				// the io.EOF contract: a second Decode
				// against whitespace-only input still
				// returns io.EOF.
				return append(b, ' ', '\n', '\t')
			},
			wantErr: false,
		},
	}

	for _, r := range rows {
		r := r
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			bytes := r.mutate(validBytes())
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
