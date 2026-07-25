// SPDX-License-Identifier: Apache-2.0

// Package closure: lifecycle_erratum_test.go asserts the
// requalification contract for the lifecycle erratum that records
// historically invalid closures.
//
// The contract is required by
// ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01.
//
// Tests in this file validate the canonical declared facts in the
// erratum document. They do NOT depend on the continued reachability
// of the predecessor historical Git objects; the erratum itself
// classifies the original binding as UNBOUND/UNAVAILABLE.
package closure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resolveErratumPath returns the canonical path to the recorded
// lifecycle erratum.
func resolveErratumPath(t *testing.T) string {
	t.Helper()
	root := findLeamasRepoRoot(t)
	return filepath.Join(root, "docs", "lifecycle-errata",
		"ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01.json")
}

// findLeamasRepoRoot locates the Leamas repository root by walking
// up from this test file's package directory until it finds the
// first directory containing `docs/lifecycle-errata`.
func findLeamasRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "docs", "lifecycle-errata")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate Leamas repository root from %s", wd)
	return ""
}

// loadErratumDocument reads and unmarshals the canonical erratum.
func loadErratumDocument(t *testing.T) map[string]any {
	t.Helper()
	path := resolveErratumPath(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read lifecycle erratum %s: %v", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("lifecycle erratum is not valid JSON: %v", err)
	}
	return doc
}

// TestLifecycleErratumSchema covers the JSON shape and required
// fields of the lifecycle erratum for the predecessor ACT.
func TestLifecycleErratumSchema(t *testing.T) {
	doc := loadErratumDocument(t)
	if doc["kind"] != "lifecycle_erratum" {
		t.Fatalf("kind=%v want lifecycle_erratum", doc["kind"])
	}
	if doc["act_id"] != "ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01" {
		t.Fatalf("act_id=%v", doc["act_id"])
	}
	if doc["recorded_by"] != "ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01" {
		t.Fatalf("recorded_by=%v", doc["recorded_by"])
	}
	decl, ok := doc["declared_subject"].(map[string]any)
	if !ok {
		t.Fatalf("declared_subject missing or wrong type")
	}
	if exists, ok := decl["exists_in_repository"].(bool); !ok || exists {
		t.Fatalf("declared_subject.exists_in_repository=%v want false", decl["exists_in_repository"])
	}
	hc, ok := doc["historical_closure"].(map[string]any)
	if !ok {
		t.Fatalf("historical_closure missing or wrong type")
	}
	if hc["status"] != "INVALID" {
		t.Fatalf("historical_closure.status=%v want INVALID", hc["status"])
	}
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
	hv, ok := doc["historical_verification"].(map[string]any)
	if !ok || hv["status"] != "UNBOUND" {
		t.Fatalf("historical_verification.status=%v want UNBOUND", hv)
	}
	pi, ok := doc["production_implementation"].(map[string]any)
	if !ok || pi["status"] != "RETAINED" {
		t.Fatalf("production_implementation.status=%v want RETAINED", pi["status"])
	}
	pcc, ok := doc["prior_closure_claim"].(map[string]any)
	if !ok {
		t.Fatalf("prior_closure_claim missing")
	}
	if w, ok := pcc["withdrawn"].(bool); !ok || !w {
		t.Fatalf("prior_closure_claim.withdrawn=%v want true", pcc["withdrawn"])
	}
	dnr, ok := doc["do_not_reclassify"].([]any)
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

// TestLifecycleErratumRecordsActualSubjectIdentity covers the
// document-contract claim that the actual implementation commit
// (the production subject retained after the historical correction)
// is recorded as a structured field. The test does not assert that
// the historical Git object remains reachable in a fresh or shallow
// clone; it validates that the erratum JSON retains the recorded
// identity unchanged.
func TestLifecycleErratumRecordsActualSubjectIdentity(t *testing.T) {
	doc := loadErratumDocument(t)
	impl, ok := doc["implementation"].(map[string]any)
	if !ok {
		t.Fatalf("implementation block missing")
	}
	commit, ok := impl["commit"].(string)
	if !ok || commit == "" {
		t.Fatalf("implementation.commit missing or empty")
	}
	if !looksLikeOID(commit) {
		t.Fatalf("implementation.commit %q is not a 40-char hex OID", commit)
	}
	tree, ok := impl["tree"].(string)
	if !ok || tree == "" {
		t.Fatalf("implementation.tree missing or empty")
	}
	if !looksLikeOID(tree) {
		t.Fatalf("implementation.tree %q is not a 40-char hex OID", tree)
	}
	if status, _ := impl["status"].(string); !containsAnyLower(status,
		"retained", "implemented", "implementation") {
		t.Fatalf("implementation.status=%q must acknowledge retained/implemented state", status)
	}
}

// TestLifecycleErratumRecordsProductionTree covers the assertion
// that the recorded production tree OID is committed text inside
// the erratum document and matches the documented 40-char hex shape.
// The test does NOT invoke `git rev-parse` against any unreachable
// Git object; that contract is delegated to the optional retention
// audit defined in ACT-LEAMAS-FACTORY-GATE-FAST-CI-PORTABILITY01 §8.5.
func TestLifecycleErratumRecordsProductionTree(t *testing.T) {
	doc := loadErratumDocument(t)
	impl, ok := doc["implementation"].(map[string]any)
	if !ok {
		t.Fatalf("implementation block missing")
	}
	gotTree, ok := impl["tree"].(string)
	if !ok {
		t.Fatalf("implementation.tree missing")
	}
	wantTree := "897587b88dc06a6f40d68c796f4ed186dbd91b6e"
	if gotTree != wantTree {
		t.Fatalf("implementation.tree=%q want=%q", gotTree, wantTree)
	}
	hc, ok := doc["historical_closure"].(map[string]any)
	if !ok {
		t.Fatalf("historical_closure missing")
	}
	firstTree, ok := hc["first_plan_appearance_tree"].(string)
	if !ok || !looksLikeOID(firstTree) {
		t.Fatalf("historical_closure.first_plan_appearance_tree=%v want OID", firstTree)
	}
}

// TestLifecycleErratumRecordsUnavailableOriginalBinding covers
// the contract that the canonical verification status of the
// predecessor is UNBOUND, and the recorded note states that the
// historical verification claim is unavailable.
func TestLifecycleErratumRecordsUnavailableOriginalBinding(t *testing.T) {
	doc := loadErratumDocument(t)
	hv, ok := doc["historical_verification"].(map[string]any)
	if !ok {
		t.Fatalf("historical_verification missing")
	}
	if hv["status"] != "UNBOUND" {
		t.Fatalf("historical_verification.status=%q want UNBOUND", hv["status"])
	}
	note, _ := hv["note"].(string)
	if note == "" {
		t.Fatalf("historical_verification.note missing")
	}
	if !containsAnyLower(note, "unavailable", "no verifiable") {
		t.Fatalf("historical_verification.note=%q must declare unavailable binding", note)
	}
	decl, ok := doc["declared_subject"].(map[string]any)
	if !ok {
		t.Fatalf("declared_subject missing")
	}
	commit, ok := decl["commit"].(string)
	if !ok || !looksLikeOID(commit) {
		t.Fatalf("declared_subject.commit missing or wrong shape")
	}
}

// TestLifecycleErratumRecordsPlanAppearanceCommit covers the
// recorded first-appearance commit for the predecessor plan.
func TestLifecycleErratumRecordsPlanAppearanceCommit(t *testing.T) {
	doc := loadErratumDocument(t)
	hc, ok := doc["historical_closure"].(map[string]any)
	if !ok {
		t.Fatalf("historical_closure missing")
	}
	want := "d20fc2c0f856b8a99330b626cd87fd256dc0a931"
	if got, _ := hc["first_plan_appearance_commit"].(string); got != want {
		t.Fatalf("first_plan_appearance_commit=%q want=%q", got, want)
	}
	if got, _ := hc["first_plan_appearance_tree"].(string); !looksLikeOID(got) {
		t.Fatalf("first_plan_appearance_tree=%v want OID", got)
	}
	placeholderField, _ := hc["plan_placeholder_field"].(string)
	if placeholderField != "baseline.tree_oid" {
		t.Fatalf("plan_placeholder_field=%q want=baseline.tree_oid", placeholderField)
	}
	if got, _ := hc["plan_placeholder_value"].(string); !strings.Contains(got, "TO_BE") {
		t.Fatalf("plan_placeholder_value=%q must record the unresolved token", got)
	}
}

// TestLifecycleErratumRecordsForwardRequalification covers the
// recorded forward-requalification identity.
func TestLifecycleErratumRecordsForwardRequalification(t *testing.T) {
	doc := loadErratumDocument(t)
	fr, ok := doc["forward_requalification"].(map[string]any)
	if !ok {
		t.Fatalf("forward_requalification missing")
	}
	wantActID := "ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01"
	if got, _ := fr["act_id"].(string); got != wantActID {
		t.Fatalf("act_id=%q want=%q", got, wantActID)
	}
	wantTag := "act/leamas-factory-self-hosted-entrypoint-authority01-correction01"
	if got, _ := fr["tag_name"].(string); got != wantTag {
		t.Fatalf("tag_name=%q want=%q", got, wantTag)
	}
	note, _ := fr["note"].(string)
	if !strings.Contains(note, "manifest") {
		t.Fatalf("forward_requalification.note=%q must defer identities to manifest", note)
	}
}

// looksLikeOID returns true when s is a 40-char lower/upper-case
// hex string.
func looksLikeOID(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// containsAnyLower reports whether s (lowercased) contains any of
// the supplied substrings (also lowercased).
func containsAnyLower(s string, subs ...string) bool {
	ls := strings.ToLower(s)
	for _, sub := range subs {
		if strings.Contains(ls, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}
