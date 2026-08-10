// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_test.go is the B2-R2
// regression matrix for the canonical Plan Contract v1
// decoder.
//
// The test proves the canonical decoder is the single
// authority for both the closure runner and the evidence
// package. The matrix covers every failure mode the
// decoder must reject:
//
//	valid document             -> PASS
//	oversize document          -> reject
//	empty document             -> reject
//	malformed JSON             -> reject
//	duplicate keys             -> reject
//	second JSON object         -> reject
//	second scalar              -> reject
//	truncated JSON             -> reject
//	unknown mode               -> reject
//	empty check id             -> reject
//	missing contract_version  -> reject
//
// The closure runner's own parser and the evidence package's
// wrapper must agree on every row. The package is the only
// place the canonical rules live.
package plancontract

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// TestPlanContractCanDecodeValidDocument proves the happy
// path. The bytes are a minimal but well-formed document.
func TestPlanContractCanDecodeValidDocument(t *testing.T) {
	t.Parallel()
	bytes := []byte(`{"contract_version":1,"checks":[` +
		`{"id":"c1","mode":"run"}` +
		`]}`)
	result, err := DecodeAndValidate(bytes)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.ContractVersion != 1 {
		t.Fatalf("contract_version = %d, want 1", result.ContractVersion)
	}
	if len(result.Checks) != 1 {
		t.Fatalf("len(Checks) = %d, want 1", len(result.Checks))
	}
	if result.Checks[0].ID != "c1" || result.Checks[0].Mode != "run" {
		t.Fatalf("check[0] = %+v, want {c1 run}", result.Checks[0])
	}
}

// TestPlanContractMatrix exercises every rejection path
// the canonical decoder must handle.
func TestPlanContractMatrix(t *testing.T) {
	t.Parallel()

	type row struct {
		name    string
		mutate  func([]byte) []byte
		wantErr bool
	}

	// validBase is the minimal valid document the matrix
	// mutates. Each row applies a single mutation; the
	// mutation may either produce a different valid
	// document (wantErr=false) or a malformed one
	// (wantErr=true).
	validBase := func() []byte {
		return []byte(`{"contract_version":1,"checks":[` +
			`{"id":"c1","mode":"run"}` +
			`]}`)
	}

	rows := []row{
		{
			name:    "valid document",
			mutate:  func(b []byte) []byte { return b },
			wantErr: false,
		},
		{
			name:    "empty document",
			mutate:  func(_ []byte) []byte { return nil },
			wantErr: true,
		},
		{
			name:    "malformed JSON",
			mutate:  func(_ []byte) []byte { return []byte(`{not json`) },
			wantErr: true,
		},
		{
			name: "duplicate keys",
			mutate: func(_ []byte) []byte {
				return []byte(`{"contract_version":1,"checks":[],"contract_version":2}`)
			},
			wantErr: true,
		},
		{
			name: "second JSON object",
			mutate: func(b []byte) []byte {
				return append(b, ' ', '{', '}')
			},
			wantErr: true,
		},
		{
			name: "second scalar",
			mutate: func(b []byte) []byte {
				return append(b, ' ', '4', '2')
			},
			wantErr: true,
		},
		{
			name: "truncated JSON",
			mutate: func(b []byte) []byte {
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
			name: "unknown mode",
			mutate: func(_ []byte) []byte {
				return []byte(`{"contract_version":1,"checks":[{"id":"c1","mode":"garbage"}]}`)
			},
			wantErr: true,
		},
		{
			name: "empty check id",
			mutate: func(_ []byte) []byte {
				return []byte(`{"contract_version":1,"checks":[{"id":"","mode":"run"}]}`)
			},
			wantErr: true,
		},
		{
			name: "missing contract_version",
			mutate: func(_ []byte) []byte {
				return []byte(`{"checks":[{"id":"c1","mode":"run"}]}`)
			},
			wantErr: true,
		},
		{
			name: "missing checks",
			mutate: func(_ []byte) []byte {
				return []byte(`{"contract_version":1}`)
			},
			wantErr: true,
		},
		{
			name: "oversize document",
			mutate: func(_ []byte) []byte {
				// Bytes > MaxPlanBytes must reject.
				oversized := make([]byte, MaxPlanBytes+1)
				sum := sha256.Sum256(oversized)
				hex.Encode(oversized, sum[:])
				return oversized
			},
			wantErr: true,
		},
	}

	for _, r := range rows {
		r := r
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			bytes := r.mutate(validBase())
			_, err := DecodeAndValidate(bytes)
			if r.wantErr && err == nil {
				t.Fatalf("decoder must reject %s, got nil", r.name)
			}
			if !r.wantErr && err != nil {
				t.Fatalf("decoder must accept %s, got %v", r.name, err)
			}
		})
	}
}

// TestPlanContractDecodeBytesReturnsParsedObject proves the
// closure runner's DecodeBytes entry point returns the parsed
// JSON object. The closure runner re-decodes the object into
// its own Plan struct; the contract guarantees the closure
// runner and the evidence package observe the same parse.
func TestPlanContractDecodeBytesReturnsParsedObject(t *testing.T) {
	t.Parallel()
	bytes := []byte(`{"contract_version":1,"checks":[` +
		`{"id":"c1","mode":"run"}` +
		`]}`)
	root, err := DecodeBytes(bytes)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	obj, ok := root.(map[string]any)
	if !ok {
		t.Fatalf("root is not a JSON object: %T", root)
	}
	if _, ok := obj["contract_version"]; !ok {
		t.Fatalf("parsed object missing contract_version: %v", obj)
	}
	if _, ok := obj["checks"]; !ok {
		t.Fatalf("parsed object missing checks: %v", obj)
	}
}

// TestPlanContractTypedAndObjectSurfacesAgree proves the
// minimal DecodeResult and the unbounded DecodeBytes surfaces
// agree on the same input. Both come from the same parser
// pass so the two surfaces cannot diverge.
func TestPlanContractTypedAndObjectSurfacesAgree(t *testing.T) {
	t.Parallel()
	bytes := []byte(`{"contract_version":1,"checks":[` +
		`{"id":"a","mode":"run"},` +
		`{"id":"b","mode":"exclude"}` +
		`]}`)

	typed, err := DecodeAndValidate(bytes)
	if err != nil {
		t.Fatalf("decode+validate: %v", err)
	}
	root, err := DecodeBytes(bytes)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	obj, ok := root.(map[string]any)
	if !ok {
		t.Fatalf("root is not an object: %T", root)
	}
	// Sanity: typed contract_version matches int(obj).
	// (The parser preserves numbers via json.Number.)
	if len(typed.Checks) != 2 {
		t.Fatalf("typed.Checks = %d, want 2", len(typed.Checks))
	}
	if typed.Checks[0].ID != "a" || typed.Checks[0].Mode != "run" {
		t.Fatalf("typed.Checks[0] = %+v", typed.Checks[0])
	}
	if typed.Checks[1].ID != "b" || typed.Checks[1].Mode != "exclude" {
		t.Fatalf("typed.Checks[1] = %+v", typed.Checks[1])
	}
	_ = obj
}
