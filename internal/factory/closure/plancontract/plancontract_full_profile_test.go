// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full_profile_test.go
// is the F1 mandated test matrix for the policy_profile
// authority. The leaf is the canonical semantic
// authority; the closure package's adapter must preserve
// the typed-error contract these tests assert.
package plancontract

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestPlanContractPolicyProfileAuthorityMatrix exercises
// every required F1 row through DecodeAndValidateFull
// rather than through the closure-side adapter. The leaf
// is the authority being repaired; a one-off closure-side
// check would not catch future regressions.
//
// Rows:
//   - F1.1 unknown profile rejected (mystery-profile)
//   - F1.2 disabled profile rejected (unsupported)
//   - F1.3 enabled profile missing required check rejected
//   - F1.4 enabled profile + all required checks accepted
//   - F1.5 omitted policy_profile behaves as legacy
//   - F1.6 invalid type (non-string) rejected
func TestPlanContractPolicyProfileAuthorityMatrix(t *testing.T) {
	canonical := validPlanWithAllLeamasChecks(t)

	t.Run("F1.1_unknown_profile_rejected", func(t *testing.T) {
		doc := canonicalWithPolicyProfile(canonical, `"mystery-profile"`)
		_, err := DecodeAndValidateFull(doc)
		if err == nil {
			t.Fatal("expected error for unknown policy_profile")
		}
		var de *DecodeError
		if !asDecodeError(err, &de) {
			t.Fatalf("error is not *DecodeError: %v", err)
		}
		if de.Code != "unknown_policy_profile" {
			t.Fatalf("code = %q, want unknown_policy_profile", de.Code)
		}
		if !strings.Contains(de.Message, `"mystery-profile"`) {
			t.Fatalf("message %q does not name profile", de.Message)
		}
	})

	t.Run("F1.2_disabled_profile_rejected", func(t *testing.T) {
		doc := canonicalWithPolicyProfile(canonical, `"unsupported"`)
		_, err := DecodeAndValidateFull(doc)
		if err == nil {
			t.Fatal("expected error for disabled policy_profile")
		}
		var de *DecodeError
		if !asDecodeError(err, &de) {
			t.Fatalf("error is not *DecodeError: %v", err)
		}
		if de.Code != "disabled_policy_profile" {
			t.Fatalf("code = %q, want disabled_policy_profile", de.Code)
		}
		if !strings.Contains(de.Message, "not yet implemented") {
			t.Fatalf("message %q does not say 'not yet implemented'", de.Message)
		}
	})

	t.Run("F1.3_enabled_profile_missing_required_check", func(t *testing.T) {
		doc := canonicalWithSingleCheck(canonical, `"leamas-act-v1"`,
			`{"id":"focused-count-1","mode":"run","argv":["go","test","-count=1","./internal/factory/closure/...","./cmd/leamas/..."],"working_directory":".","timeout_seconds":600,"environment":{}}`)
		_, err := DecodeAndValidateFull(doc)
		if err == nil {
			t.Fatal("expected error for missing required check")
		}
		var de *DecodeError
		if !asDecodeError(err, &de) {
			t.Fatalf("error is not *DecodeError: %v", err)
		}
		if de.Code != "missing_policy_profile_check" {
			t.Fatalf("code = %q, want missing_policy_profile_check", de.Code)
		}
	})

	t.Run("F1.4_enabled_profile_all_required_checks_accepted", func(t *testing.T) {
		_, err := DecodeAndValidateFull(canonical)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("F1.5_omitted_policy_profile_accepted", func(t *testing.T) {
		doc := canonicalWithPolicyProfile(canonical, "")
		_, err := DecodeAndValidateFull(doc)
		if err != nil {
			t.Fatalf("expected nil error for omitted policy_profile, got %v", err)
		}
	})

	t.Run("F1.6_non_string_policy_profile_rejected", func(t *testing.T) {
		doc := canonicalWithPolicyProfile(canonical, "42")
		_, err := DecodeAndValidateFull(doc)
		if err == nil {
			t.Fatal("expected error for non-string policy_profile")
		}
		var de *DecodeError
		if !asDecodeError(err, &de) {
			t.Fatalf("error is not *DecodeError: %v", err)
		}
		if de.Code != "invalid_policy_profile_type" {
			t.Fatalf("code = %q, want invalid_policy_profile_type", de.Code)
		}
	})
}

// validPlanWithAllLeamasChecks returns the canonical
// "leamas-act-v1" claims payload as raw JSON bytes. The
// payload declares the policy_profile plus every required
// check the leamas-act-v1 profile mandates so F1.4 can
// accept it without mutation.
func validPlanWithAllLeamasChecks(t *testing.T) []byte {
	t.Helper()
	return []byte(`{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-PROFILE-MATRIX01",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid":   "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "policy_profile": "leamas-act-v1",
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  },
  "artifacts": [
    {"id":"summary","path":".factory/summary.json","required":true,"max_bytes":1024,"media_type":"application/json"}
  ],
  "checks": [
    {"id":"focused-count-1","mode":"run","argv":["go","test","-count=1","./internal/factory/closure/...","./cmd/leamas/..."],"working_directory":".","timeout_seconds":600,"environment":{}},
    {"id":"focused-count-20","mode":"run","argv":["go","test","-count=20","./internal/factory/closure/...","./cmd/leamas/..."],"working_directory":".","timeout_seconds":600,"environment":{}},
    {"id":"focused-race-5","mode":"run","argv":["go","test","-race","-count=5","./internal/factory/closure/...","./cmd/leamas/..."],"working_directory":".","timeout_seconds":600,"environment":{}},
    {"id":"vet","mode":"run","argv":["go","vet","./internal/factory/closure/...","./cmd/leamas/..."],"working_directory":".","timeout_seconds":300,"environment":{}},
    {"id":"build","mode":"run","argv":["go","build","-buildvcs=true","-trimpath","-o","/tmp/leamas-closure-protocol-v1-self","./cmd/leamas"],"working_directory":".","timeout_seconds":600,"environment":{}},
    {"id":"gate-fast","mode":"run","argv":["make","gate-fast"],"working_directory":".","timeout_seconds":600,"environment":{}},
    {"id":"diff-check","mode":"run","argv":["git","diff","--check"],"working_directory":".","timeout_seconds":60,"environment":{}}
  ]
}`)
}

// canonicalWithPolicyProfile swaps the policy_profile
// value to raw. raw is the literal JSON value (with
// quotes for a string, or empty to remove the field).
func canonicalWithPolicyProfile(canonical []byte, raw string) []byte {
	out := string(canonical)
	const marker = `"policy_profile": "leamas-act-v1"`
	if raw == "" {
		return []byte(strings.Replace(out, marker+",", "", 1))
	}
	return []byte(strings.Replace(out, marker, `"policy_profile": `+raw, 1))
}

// canonicalWithSingleCheck swaps the policy_profile and
// replaces the entire /checks array with a single check
// built from raw. raw must be a single JSON object
// representing one check.
//
// Implementation note: the canonical fixture's /checks
// array contains "]>" characters inside argv elements,
// so a naive `strings.Index(out, "]")` would locate a
// `]` inside an argv literal. We instead walk the
// post-`"checks": [` substring and count bracket depth
// to find the matching close.
func canonicalWithSingleCheck(canonical []byte, profileJSON, singleCheckJSON string) []byte {
	out := string(canonical)
	out = strings.Replace(out, `"policy_profile": "leamas-act-v1"`,
		`"policy_profile": `+profileJSON, 1)
	start := strings.Index(out, `"checks": [`)
	if start < 0 {
		panic("checks array not found in canonical fixture")
	}
	// Walk forward from after the opening '['; the first
	// ']' at depth 0 is the array close.
	depth := 1
	i := start + len(`"checks": [`)
	for ; i < len(out); i++ {
		switch out[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return []byte(out[:start] + `"checks": [` + singleCheckJSON + `]` + out[i+1:])
			}
		}
	}
	panic("checks array close not found in canonical fixture")
}

// asDecodeError is a typed-errors helper. The leaf emits
// *DecodeError directly so no errors.As dance is needed.
func asDecodeError(err error, target **DecodeError) bool {
	if err == nil {
		return false
	}
	if t, ok := err.(*DecodeError); ok {
		*target = t
		return true
	}
	return false
}

// TestPlanContractPolicyProfileStructuredContextMatrix
// is the CORRECTION02 adversarial matrix. It proves that
// the leaf's DecodeError.Profile and DecodeError.CheckID
// structured fields carry the ORIGINAL typed values
// even when those values contain characters that Go's
// `%q`-formatted Message would escape. Before CORRECTION02
// the closure adapter reconstructed ProfileName/CheckID
// by scanning the leaf's display prose for quote
// characters; an embedded `"` would corrupt the
// reconstructed value. After CORRECTION02 the adapter
// reads DecodeError.Profile/CheckID directly.
//
// Rows:
//   - A1 unknown profile name with embedded "
//   - A2 unknown profile name with backslash
//   - A3 unknown profile name with non-ASCII (Unicode)
//   - A4 unknown profile name with embedded tab/newline
//   - A5 missing-check carries CheckID structurally
//     (the Expert's required addition: the missing-check
//     path is the second half of the original
//     display-prose roundtrip; the test would catch a
//     fix that handled Profile but left CheckID
//     reconstructed from prose.)
//   - A6 unknown profile name with path separator
//   - A7 unknown profile name with mixed case
func TestPlanContractPolicyProfileStructuredContextMatrix(t *testing.T) {
	canonical := validPlanWithAllLeamasChecks(t)

	// Each row substitutes a profile name containing
	// tricky characters into a copy of the canonical
	// payload. The leaf must report the original
	// verbatim in DecodeError.Profile (no %q-escaping,
	// no truncation, no sanitization).
	//
	// rawValue is the JSON-encoded string the helper will
	// substitute into the payload. For values containing
	// literal control characters (tab/newline) the JSON
	// encoding is non-trivial, so we use json.Marshal for
	// correctness.
	cases := []struct {
		name      string
		wantValue string // expected DecodeError.Profile
	}{
		{
			name:      "A1_unknown_profile_with_embedded_quote",
			wantValue: `foo"bar`,
		},
		{
			name:      "A2_unknown_profile_with_backslash",
			wantValue: `foo\bar`,
		},
		{
			name:      "A3_unknown_profile_with_unicode",
			wantValue: `профиль-неизвестен`,
		},
		{
			name:      "A4_unknown_profile_with_tab_and_newline",
			wantValue: "foo\tbar\nbaz",
		},
		{
			name:      "A6_unknown_profile_with_path_separator",
			wantValue: "path/with/slashes",
		},
		{
			name:      "A7_unknown_profile_with_mixed_case",
			wantValue: "UPPER-lower-Mixed",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.wantValue)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			doc := canonicalWithPolicyProfile(canonical, string(raw))
			_, err = DecodeAndValidateFull(doc)
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
			var de *DecodeError
			if !asDecodeError(err, &de) {
				t.Fatalf("error is not *DecodeError: %v", err)
			}
			if de.Code != "unknown_policy_profile" {
				t.Fatalf("code = %q, want unknown_policy_profile", de.Code)
			}
			if de.Profile != tc.wantValue {
				t.Fatalf("Profile = %q (len=%d), want %q (len=%d)",
					de.Profile, len(de.Profile),
					tc.wantValue, len(tc.wantValue))
			}
		})
	}

	// A5: missing-check carries CheckID structurally.
	// The Expert's required addition: the missing-check
	// path is the second half of the original display-
	// prose roundtrip. The test verifies that when the
	// leaf rejects a plan for missing a required
	// profile check, it carries the missing check ID
	// in DecodeError.CheckID, not just embedded in the
	// human-readable Message.
	t.Run("A5_missing_check_carries_check_id_structurally", func(t *testing.T) {
		// Strip every required check from the canonical
		// payload; the leaf MUST report the FIRST missing
		// check by ID. (Validation iterates required checks
		// in declared order; "focused-count-1" is first.)
		doc := canonicalWithSingleCheck(canonical, `"leamas-act-v1"`,
			`{"id":"unrelated-check","mode":"run","argv":["echo","hi"],"working_directory":".","timeout_seconds":60,"environment":{}}`)
		_, err := DecodeAndValidateFull(doc)
		if err == nil {
			t.Fatal("expected error for missing required profile check")
		}
		var de *DecodeError
		if !asDecodeError(err, &de) {
			t.Fatalf("error is not *DecodeError: %v", err)
		}
		if de.Code != "missing_policy_profile_check" {
			t.Fatalf("code = %q, want missing_policy_profile_check", de.Code)
		}
		if de.CheckID != "focused-count-1" {
			t.Fatalf("CheckID = %q, want focused-count-1", de.CheckID)
		}
		if de.Profile != "leamas-act-v1" {
			t.Fatalf("Profile = %q, want leamas-act-v1", de.Profile)
		}
	})
}
