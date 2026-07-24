// SPDX-License-Identifier: Apache-2.0

// Package closure: lifecycle_erratum_test.go asserts the
// requalification contract for the lifecycle erratum that records
// historically invalid closures.
//
// The contract is required by
// ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01.
//
// Tests in this file are independent of the closure plan / manifest
// pipeline so they exercise the validation contract in isolation.
package closure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/authority"
)

// TestLifecycleErratumSchema covers the JSON shape and required
// fields of the lifecycle erratum for the predecessor ACT.
//
// The erratum must be a real file in the repository, not a fabricated
// placeholder, and it must classify the predecessor ACT's lifecycle
// as INVALID rather than as a generic CLOSED or VERIFIED status.
func TestLifecycleErratumSchema(t *testing.T) {
	// Resolve the repository root from this test file's package.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))

	path := filepath.Join(repoRoot, "docs", "lifecycle-errata",
		"ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read lifecycle erratum %s: %v", path, err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("lifecycle erratum is not valid JSON: %v", err)
	}
	if got["kind"] != "lifecycle_erratum" {
		t.Fatalf("kind=%v want lifecycle_erratum", got["kind"])
	}
	if got["act_id"] != "ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01" {
		t.Fatalf("act_id=%v", got["act_id"])
	}
	if got["recorded_by"] != "ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01" {
		t.Fatalf("recorded_by=%v", got["recorded_by"])
	}

	// declared_subject exists-in-repository must be false.
	decl, ok := got["declared_subject"].(map[string]any)
	if !ok {
		t.Fatalf("declared_subject missing or wrong type")
	}
	if exists, ok := decl["exists_in_repository"].(bool); !ok || exists {
		t.Fatalf("declared_subject.exists_in_repository=%v want false", decl["exists_in_repository"])
	}

	// historical_closure.status must be INVALID.
	hc, ok := got["historical_closure"].(map[string]any)
	if !ok {
		t.Fatalf("historical_closure missing or wrong type")
	}
	if hc["status"] != "INVALID" {
		t.Fatalf("historical_closure.status=%v want INVALID", hc["status"])
	}

	// historical_closure.reasons must list the six documented reasons.
	reasons, ok := hc["reasons"].([]any)
	if !ok {
		t.Fatalf("historical_closure.reasons missing or wrong type")
	}
	wantReasons := map[string]bool{
		"declared_subject_object_missing":      false,
		"no_pre_subject_plan_freeze":           false,
		"plan_contains_unresolved_placeholder": false,
		"closure_manifest_missing":             false,
		"attestation_missing":                  false,
		"annotated_tag_missing":                false,
	}
	for _, r := range reasons {
		s, ok := r.(string)
		if !ok {
			continue
		}
		if _, ok := wantReasons[s]; ok {
			wantReasons[s] = true
		}
	}
	for reason, seen := range wantReasons {
		if !seen {
			t.Fatalf("historical_closure.reasons missing %q (got %v)", reason, reasons)
		}
	}

	// historical_verification.status must be UNBOUND.
	hv, ok := got["historical_verification"].(map[string]any)
	if !ok || hv["status"] != "UNBOUND" {
		t.Fatalf("historical_verification.status=%v want UNBOUND", hv)
	}

	// production_implementation.status must be RETAINED.
	pi, ok := got["production_implementation"].(map[string]any)
	if !ok || pi["status"] != "RETAINED" {
		t.Fatalf("production_implementation.status=%v want RETAINED", pi)
	}

	// prior_closure_claim.withdrawn must be true.
	pcc, ok := got["prior_closure_claim"].(map[string]any)
	if !ok {
		t.Fatalf("prior_closure_claim missing")
	}
	if w, ok := pcc["withdrawn"].(bool); !ok || !w {
		t.Fatalf("prior_closure_claim.withdrawn=%v want true", pcc["withdrawn"])
	}

	// do_not_reclassify must contain VERIFIED, CLOSED_LOCAL, PUBLISHED.
	dnr, ok := got["do_not_reclassify"].([]any)
	if !ok {
		t.Fatalf("do_not_reclassify missing")
	}
	wantNot := map[string]bool{"VERIFIED": false, "CLOSED_LOCAL": false, "PUBLISHED": false}
	for _, v := range dnr {
		s, _ := v.(string)
		if _, ok := wantNot[s]; ok {
			wantNot[s] = true
		}
	}
	for v, seen := range wantNot {
		if !seen {
			t.Fatalf("do_not_reclassify missing %q (got %v)", v, dnr)
		}
	}
}

// TestLifecycleErratumSubjectCommitResolvesAsCommit asserts the
// actual S0 OID recorded in the erratum resolves as a real Git
// commit object. This is the key predicate that distinguishes a
// truthful erratum from a fabricated one.
func TestLifecycleErratumSubjectCommitResolvesAsCommit(t *testing.T) {
	const subject = "06c51158d104c20eec389736a2a0bcff06743630"
	if err := assertCommitResolves(subject); err != nil {
		t.Fatalf("%v", err)
	}
}

// TestLifecycleErratumDeclaredSubjectDoesNotResolve asserts the
// declared-but-nonexistent OID fails the same resolver. This is the
// inverse of the previous test and pins the historical defect.
func TestLifecycleErratumDeclaredSubjectDoesNotResolve(t *testing.T) {
	const declared = "06c5115a5d2e7c4f4a26f5c1e3b9a8d7c6e5f4a3"
	_, err := authority.DefaultGitRunner(".", "cat-file", "-e", declared)
	if err == nil {
		t.Fatalf("declared subject %q must not resolve as a commit", declared)
	}
	if !strings.Contains(err.Error(), "exit") {
		t.Fatalf("expected git failure, got %v", err)
	}
}

// TestLifecycleErratumProductionTreeMatches asserts the recorded
// production tree OID matches the actual tree of the recorded
// production subject commit. This pins the relationship between the
// erratum and the real Git history.
func TestLifecycleErratumProductionTreeMatches(t *testing.T) {
	const subject = "06c51158d104c20eec389736a2a0bcff06743630"
	const tree = "897587b88dc06a6f40d68c796f4ed186dbd91b6e"
	out, err := authority.DefaultGitRunner(".", "rev-parse", subject+"^{tree}")
	if err != nil {
		t.Fatalf("rev-parse %s^{tree}: %v", subject, err)
	}
	if strings.TrimSpace(out) != tree {
		t.Fatalf("erratum tree=%s != git %s^{tree}=%s", tree, subject, out)
	}
}

// assertCommitResolves asserts the given 40-char hex commit OID
// resolves to a real Git object. It uses `git cat-file -e`, which
// exits non-zero when the object is missing.
func assertCommitResolves(oid string) error {
	_, err := authority.DefaultGitRunner(".", "cat-file", "-e", oid)
	return err
}
