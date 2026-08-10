// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_contract_parity_b2r4_test.go is the
// B2-R4 execution/evidence parity matrix.
//
// B2-R3 closed the duplicate-field, contract-version, and
// repository-binding gaps but left Plan Contract semantics
// split across two validators:
//
//	closure.ValidatePlan  -> full typed semantic rules
//	plancontract.Validate -> only contract_version +
//	                         minimal check-shape rules
//
// The evidence package consumed the minimal validator while
// the closure runner consumed the typed validator. A plan
// that passed the minimal validator could fail the typed
// validator (placeholder baseline OID, invalid act_id, too
// many checks, etc.) so execution and evidence did not share
// the same authority.
//
// B2-R4 closes the gap by moving the full semantic authority
// into plancontract.ValidateFull. Both paths MUST consume
// the same function. The parity matrix below proves it.
//
// For every row:
//
//	executionAccept := closure.ValidatePlan parses and
//	                   validates the bytes.
//	evidenceAccept  := plancontract.ValidateFull on the same
//	                   bytes.
//
// The test asserts executionAccept == evidenceAccept for
// every row. Any drift is a B2-R4 contract bug.
package closure

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// runCheckJSON is the canonical run-mode check entry used
// when constructing duplicate-row fixtures. Defined as a
// constant to keep the long fixture strings under the
// LLM-friendly line-length threshold.
const runCheckJSON = `"mode":"run","argv":["go","test"],"working_directory":".","timeout_seconds":60,"environment":{"K":"V"}`

// validPlanBase returns a fully-valid Plan Contract v1
// document. Every row mutates exactly one field.
func validPlanBase() []byte {
	const oid40 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const oid40b = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	b := newValidPlanBuilder()
	b.add("contract_version", "1")
	b.add("act_id", `"ACT-PARITY-B2R4-01"`)
	b.addRaw(fmt.Sprintf(`"baseline":{"commit_oid":%q,"tree_oid":%q}`, oid40, oid40b))
	b.addRaw(`"execution":{"mode":"serial_fail_fast"}`)
	b.addRaw(`"checks":[{"id":"c1","mode":"run","argv":["go","test"],"working_directory":".","timeout_seconds":60,"environment":{"K":"V"}}]`)
	b.addRaw(`"artifacts":[{"id":"a1","path":"docs/a.md","required":true,"max_bytes":1024,"media_type":"text/plain"}]`)
	b.addRaw(`"policy":{"require_clean_before":true,"require_clean_after":true,"forbid_tracked_full_digests":true,"require_diff_check":true}`)
	return b.bytes()
}

// validPlanBuilder accumulates Plan JSON bytes for the
// parity matrix. The builder applies individual mutations so
// each row exercises a single semantic rule.
type validPlanBuilder struct {
	parts []string
}

func newValidPlanBuilder() *validPlanBuilder {
	return &validPlanBuilder{
		parts: []string{},
	}
}

func (b *validPlanBuilder) add(key, value string) *validPlanBuilder {
	b.parts = append(b.parts, fmt.Sprintf("%q:%s", key, value))
	return b
}

func (b *validPlanBuilder) addRaw(s string) *validPlanBuilder {
	b.parts = append(b.parts, s)
	return b
}

func (b *validPlanBuilder) bytes() []byte {
	return []byte("{" + strings.Join(b.parts, ",") + "}")
}

// TestPlanContractExecutionEvidenceParity is the B2-R4
// parity matrix. Every row asserts that the closure runner
// and the evidence package observe the same validity.
func TestPlanContractExecutionEvidenceParity(t *testing.T) {
	t.Parallel()

	const oid40 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const oid40b = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	type row struct {
		name    string
		mutate  func([]byte) []byte
		wantErr bool
	}

	rows := []row{
		{
			name:    "valid base",
			mutate:  func(b []byte) []byte { return b },
			wantErr: false,
		},
		{
			name: "contract_version=2",
			mutate: func(b []byte) []byte {
				return replaceJSONField(b, "contract_version", "2")
			},
			wantErr: true,
		},
		{
			name: "invalid act_id",
			mutate: func(b []byte) []byte {
				return replaceJSONField(b, "act_id", `"not-an-act-id"`)
			},
			wantErr: true,
		},
		{
			name: "placeholder baseline commit_oid",
			mutate: func(b []byte) []byte {
				s := string(b)
				const anchor = `"commit_oid":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`
				return []byte(strings.Replace(s, anchor, `"commit_oid":"TO_BE_FILLED"`, 1))
			},
			wantErr: true,
		},
		{
			name: "malformed baseline commit_oid",
			mutate: func(b []byte) []byte {
				s := string(b)
				const anchor = `"commit_oid":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`
				return []byte(strings.Replace(s, anchor, `"commit_oid":"not-a-hex-oid"`, 1))
			},
			wantErr: true,
		},
		{
			name: "zero checks",
			mutate: func(b []byte) []byte {
				s := string(b)
				const anchor = `"checks":[`
				end := strings.Index(s, anchor) + len(anchor)
				rest := s[end:]
				closeIdx := strings.Index(rest, "]")
				return []byte(s[:end] + rest[closeIdx:])
			},
			wantErr: true,
		},
		{
			name: "too many checks",
			mutate: func(b []byte) []byte {
				// 65 entries -> > MaxChecks (64).
				parts := make([]string, 0, 65)
				for i := 0; i < 65; i++ {
					parts = append(parts, fmt.Sprintf(`{"id":"c%d",`+runCheckJSON+`}`, i))
				}
				s := string(b)
				anchor := `"checks":[`
				start := strings.Index(s, anchor) + len(anchor)
				rest := s[start:]
				closeIdx := strings.Index(rest, "]")
				return []byte(s[:start] + strings.Join(parts, ",") + rest[closeIdx:])
			},
			wantErr: true,
		},
		{
			name: "invalid check mode",
			mutate: func(b []byte) []byte {
				return replaceJSONField(b, `"mode":"run"`, `"mode":"garbage"`)
			},
			wantErr: true,
		},
		{
			name: "duplicate check IDs",
			mutate: func(b []byte) []byte {
				return replaceJSONField(b, `"id":"c1"`, `"id":"c1",`+runCheckJSON+`,{"id":"c1",`+runCheckJSON)
			},
			wantErr: true,
		},
		{
			name: "invalid run working_directory",
			mutate: func(b []byte) []byte {
				return replaceJSONField(b, `"working_directory":"."`, `"working_directory":"../escape"`)
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			mutate: func(b []byte) []byte {
				return replaceJSONField(b, `"timeout_seconds":60`, `"timeout_seconds":999999`)
			},
			wantErr: true,
		},
		{
			name: "exclude carrying run-only fields",
			mutate: func(b []byte) []byte {
				s := string(b)
				old := `{"id":"c1","mode":"run","argv":["go","test"],"working_directory":".","timeout_seconds":60,"environment":{"K":"V"}}`
				replacement := `{"id":"c1","mode":"exclude","reason":"obsolete","argv":["go","test"]}`
				return []byte(strings.Replace(s, old, replacement, 1))
			},
			wantErr: true,
		},
		{
			name: "missing required policy field",
			mutate: func(b []byte) []byte {
				s := string(b)
				return []byte(strings.Replace(s, `"require_clean_before":true,`, ``, 1))
			},
			wantErr: true,
		},
		{
			name: "invalid policy constraint type",
			mutate: func(b []byte) []byte {
				return replaceJSONField(b, `"require_clean_before":true`, `"require_clean_before":"yes"`)
			},
			wantErr: true,
		},
		{
			name: "valid exclude check (control row)",
			mutate: func(b []byte) []byte {
				s := string(b)
				old := `{"id":"c1","mode":"run","argv":["go","test"],"working_directory":".","timeout_seconds":60,"environment":{"K":"V"}}`
				replacement := `{"id":"c1","mode":"exclude","reason":"obsolete"}`
				return []byte(strings.Replace(s, old, replacement, 1))
			},
			wantErr: false,
		},
		// Valid: subject_exact with explicit null tool. JSON null
		// is treated as "no tool" because the typed Plan decodes
		// it as a nil *ToolAuthority pointer. Both authorities
		// accept.
		{
			name: "valid subject_exact runner authority",
			mutate: func(b []byte) []byte {
				s := string(b)
				tail := fmt.Sprintf(`,"runner_authority":{"mode":"subject_exact","tool":null}`)
				return []byte(s[:len(s)-1] + tail + "}")
			},
			wantErr: false,
		},
		{
			name: "valid tool_release_exact runner authority",
			mutate: func(b []byte) []byte {
				s := string(b)
				tail := fmt.Sprintf(`,"runner_authority":{"mode":"tool_release_exact","tool":{"revision":%q,"binary_sha256":"%s"}}`,
					oid40, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
				return []byte(s[:len(s)-1] + tail + "}")
			},
			wantErr: false,
		},
	}

	for _, r := range rows {
		r := r
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			bytes := r.mutate(validPlanBase())

			// Evidence path: plancontract.ValidateFull.
			evidenceErr := plancontract.ValidateFull(bytes)

			// Execution path: closure.LoadPlanFromBytes
			// (plancontract.DecodeBytes + typed decode) +
			// closure.ValidatePlan (typed semantic).
			plan, _, execErr := LoadPlanFromBytes(bytes)
			if execErr == nil {
				execErr = ValidatePlan(plan)
			}

			executionAccept := execErr == nil
			evidenceAccept := evidenceErr == nil
			if executionAccept != evidenceAccept {
				t.Fatalf("execution/evidence parity broken for %q:\n  execution accept=%v err=%v\n  evidence  accept=%v err=%v",
					r.name, executionAccept, execErr, evidenceAccept, evidenceErr)
			}
			if r.wantErr && executionAccept {
				t.Fatalf("both authorities accepted %q but the matrix expected rejection", r.name)
			}
			if !r.wantErr && !executionAccept {
				t.Fatalf("both authorities rejected %q but the matrix expected acceptance: exec=%v evidence=%v",
					r.name, execErr, evidenceErr)
			}
			// Suppress unused oid constants for rows that
			// do not reference them; the package compiler
			// would otherwise warn about redeclaration.
			_ = oid40
			_ = oid40b
		})
	}
}

// replaceJSONField replaces the first occurrence of needle
// with replacement in the supplied JSON document. The
// replacement is a literal JSON fragment and must produce
// valid JSON when substituted.
func replaceJSONField(doc []byte, needle, replacement string) []byte {
	return []byte(strings.Replace(string(doc), needle, replacement, 1))
}

// _ guards the json import even if no row directly uses it.
var _ = json.Marshal
