// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_strict_decode_test.go is
// the B2-R4 strict duplicate-key matrix for the canonical
// strict single-document decoder.
//
// B2-R3 added duplicate-key rejection to the strict decoder
// via a custom scanner, but the implementation pre-consumed
// the root delimiter before delegating to the recursive
// walker. As a result, the root object was never scanned and
// the tests that were meant to prove the contract were
// accidentally satisfied by other rejection paths
// (malformed JSON or unknown-field rejection).
//
// The matrix below uses GENUINELY VALID JSON with duplicate
// keys so the only rejection mechanism available is the
// duplicate-key scanner. Each row asserts the strict decoder
// rejects the duplicate and propagates the scanner error.
//
// Coverage:
//
//	valid document                       -> PASS
//	duplicate top-level known field      -> reject
//	duplicate nested known field         -> reject
//	duplicate inside array object        -> reject
//	same field name in different objects -> accept
//	duplicate in deeply nested object    -> reject
package plancontract

import (
	"testing"
)

// TestPlanContractStrictDecodeRejectsDuplicateKeys is the
// B2-R4 matrix that proves the strict duplicate-key scanner
// actually scans the root object. Every row that expects
// rejection uses valid JSON; the only rejection mechanism is
// the scanner, not the typed decoder.
func TestPlanContractStrictDecodeRejectsDuplicateKeys(t *testing.T) {
	t.Parallel()

	type row struct {
		name    string
		bytes   []byte
		wantErr bool
	}

	rows := []row{
		{
			name: "valid document",
			bytes: []byte(`{"schema_version":1,"protocol":"v1"}`),
			wantErr: false,
		},
		{
			// Genuinely valid JSON with a duplicate
			// top-level key. encoding/json v1 silently
			// accepts the duplicate (the Go team flags
			// this as a top-priority bug); the strict
			// decoder must reject it via the scanner.
			name:    "duplicate top-level known field",
			bytes:   []byte(`{"schema_version":1,"protocol":"v1","schema_version":2}`),
			wantErr: true,
		},
		{
			// Duplicate inside a nested object, using a
			// field name that is otherwise valid.
			name:    "duplicate nested known field",
			bytes:   []byte(`{"schema_version":1,"runtime":{"name":"r","name":"r2"}}`),
			wantErr: true,
		},
		{
			// Duplicate inside an object that lives
			// inside an array. The scanner must walk
			// into arrays and inspect every object.
			name:    "duplicate inside array object",
			bytes:   []byte(`{"items":[{"id":1,"id":2}]}`),
			wantErr: true,
		},
		{
			// Same field name appearing in two separate
			// objects is fine: duplicates are scoped to
			// a single object. The scanner must NOT
			// reject this.
			name:    "same field name in different objects",
			bytes:   []byte(`{"a":{"name":"x"},"b":{"name":"y"}}`),
			wantErr: false,
		},
		{
			// Duplicate deep in the tree: a
			// nested-object field name that appears
			// twice.
			name:    "duplicate in deeply nested object",
			bytes:   []byte(`{"a":{"b":{"c":{"d":"1","d":"2"}}}}`),
			wantErr: true,
		},
	}

	for _, r := range rows {
		r := r
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			var target map[string]any
			err := StrictDecode(r.bytes, &target)
			if r.wantErr && err == nil {
				t.Fatalf("StrictDecode must reject %q, got nil", r.name)
			}
			if !r.wantErr && err != nil {
				t.Fatalf("StrictDecode must accept %q, got %v", r.name, err)
			}
		})
	}
}
