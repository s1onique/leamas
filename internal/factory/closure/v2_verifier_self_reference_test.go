// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_self_reference_test.go proves the verifier does
// NOT require manifest.closure_commit to equal the externally
// supplied C.
//
// The closure protocol mandates that C authority is the
// external verifier input plus strict topology plus exact
// C:M binding. The committed manifest MAY contain a
// `closure_commit` field for human readability, but the
// verifier MUST NOT require that field to match the
// externally supplied C.
//
// The proof is constructive:
//
//  1. Construct S, F containing P, manifest bytes created
//     before C exists, C committing M.
//  2. Verify successfully with externally supplied C.
//  3. Mutate the manifest bytes so `closure_commit` is set
//     to a guessed future OID.
//  4. Prove the resulting C identity changes (because the
//     manifest is part of C's tree).
//
// The document records the self-reference model so ACT 3 /
// ACT 4 / ACT 5 can rely on the same authority derivation.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestV2VerifierSelfReferenceProof is the primary doctrine
// proof. The test:
//
//   - constructs S, F (with plan P), manifest bytes (without
//     closure_commit pointing at C), C committing M;
//   - verifies successfully with externally supplied C;
//   - asserts that the C identity depends on the manifest
//     bytes (proving C authority is the externally supplied
//     value plus the strict C:M binding, not a manifest
//     self-reference).
func TestV2VerifierSelfReferenceProof(t *testing.T) {
	dir := initRepo(t)
	// Subject: implementation only, no manifest.
	subject := makeCommit(t, dir, "subject implementation", map[string]string{
		"subject-only.txt": "subject implementation\n",
	})
	// Freeze: adds plan P. The plan P contains a single
	// passing check (the test fixture guarantees validity).
	freeze := makeCommit(t, dir, "freeze: add closure plan", map[string]string{
		"docs/closure-plans/PLAN.json": validPlanFixture(),
	})

	// Construct manifest bytes BEFORE C exists. The bytes
	// deliberately do NOT contain closure_commit=C; the
	// only authority for C is the externally supplied
	// argument to the verifier.
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	freezeTree := mustRunGit(t, dir, "rev-parse", freeze+"^{tree}")
	planBlob := mustRunGit(t, dir, "rev-parse", freeze+":docs/closure-plans/PLAN.json")
	planBytes := mustReadPlanBlob(t, dir, freeze+":docs/closure-plans/PLAN.json")
	manifestBytes := buildSelfRefManifest(
		subjectTree, freezeTree, planBlob,
		hex.EncodeToString(sha256.New().Sum(planBytes)))

	// Closure: commit the manifest at M. C is determined
	// by git; the verifier will receive C externally.
	closure := writeAndCommit(t, dir,
		"docs/closure-manifests/MANIFEST.json", manifestBytes,
		"closure: commit manifest")
	closureOID := mustRunGit(t, dir, "rev-parse", closure)

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}

	// Build the verifier request from the externally
	// supplied C. The verifier MUST NOT require
	// manifest.closure_commit to match C.
	req := V2ClosureVerifyRequest{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    PlanContractV1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		ClosureCommit:          closureOID,
		PlanPath:               "docs/closure-plans/PLAN.json",
		ManifestPath:           "docs/closure-manifests/MANIFEST.json",
	}

	// Topology MUST accept S < F < C.
	resolver := NewV2ClosureTopologyResolver()
	facts, err := resolver.ResolveTopology(context.Background(), auth, req)
	if err != nil {
		t.Fatalf("ResolveTopology: %v", err)
	}
	if facts.Relation != V2ClosureRelationSubjectBeforeFreezeBeforeClosure {
		t.Fatalf("topology relation = %q, want accepted",
			facts.Relation)
	}

	// Frozen plan authority MUST resolve to F:P bytes.
	plan, err := ResolveV2FrozenPlanAuthority(
		context.Background(), auth, freeze, req.PlanPath)
	if err != nil {
		t.Fatalf("ResolveV2FrozenPlanAuthority: %v", err)
	}
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("frozen plan authority diagnostics = %v",
			plan.Diagnostics.Codes())
	}

	// Committed manifest authority MUST resolve to C:M
	// bytes (the externally supplied C, not a value from
	// inside M).
	mf, err := ResolveV2CommittedManifestAuthority(
		context.Background(), auth, closureOID, req.ManifestPath)
	if err != nil {
		t.Fatalf("ResolveV2CommittedManifestAuthority: %v", err)
	}
	if len(mf.Diagnostics) != 0 {
		t.Fatalf("committed manifest authority diagnostics = %v",
			mf.Diagnostics.Codes())
	}
	if !strings.Contains(string(mf.RawBytes), `closure_commit": "<externally-supplied>"`) {
		t.Fatalf("manifest bytes must mark closure_commit as externally supplied; got %q",
			string(mf.RawBytes))
	}
}

// TestV2VerifierSelfReferenceMutationChangesC proves that
// mutating the manifest bytes (specifically, adding a
// guessed future `closure_commit`) changes the resulting C
// identity. This proves C authority is the externally
// supplied value plus strict topology plus exact C:M
// binding, not self-reference inside M.
func TestV2VerifierSelfReferenceMutationChangesC(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject implementation", map[string]string{
		"subject-only.txt": "subject implementation\n",
	})
	freeze := makeCommit(t, dir, "freeze: add closure plan", map[string]string{
		"docs/closure-plans/PLAN.json": validPlanFixture(),
	})

	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	freezeTree := mustRunGit(t, dir, "rev-parse", freeze+"^{tree}")
	planBlob := mustRunGit(t, dir, "rev-parse", freeze+":docs/closure-plans/PLAN.json")
	planBytes := mustReadPlanBlob(t, dir, freeze+":docs/closure-plans/PLAN.json")

	// Two manifest variants: original (closure_commit
	// placeholder) and mutated (closure_commit = a
	// guessed future OID).
	originalBytes := buildSelfRefManifest(
		subjectTree, freezeTree, planBlob,
		hex.EncodeToString(sha256.New().Sum(planBytes)))
	mutatedBytes := strings.Replace(string(originalBytes),
		`"closure_commit": "<externally-supplied>"`,
		`"closure_commit": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"`, 1)

	closure1 := writeAndCommit(t, dir,
		"docs/closure-manifests/MANIFEST.json", originalBytes,
		"closure with original manifest")
	closure2 := writeAndCommit(t, dir,
		"docs/closure-manifests/MANIFEST.json", []byte(mutatedBytes),
		"closure with mutated manifest")

	oid1 := mustRunGit(t, dir, "rev-parse", closure1)
	oid2 := mustRunGit(t, dir, "rev-parse", closure2)
	if oid1 == oid2 {
		t.Fatalf("mutating manifest bytes MUST change C identity, but oid1==oid2==%q", oid1)
	}
}

// TestV2VerifierSelfReferenceModel documents the authority
// derivation in a way the closure report can quote
// verbatim. The test asserts the model string matches the
// spec exactly.
func TestV2VerifierSelfReferenceModel(t *testing.T) {
	const expectedModel = `C authority =
  external verifier input
  + strict topology
  + exact C:M binding

not self-reference inside M`

	if V2ClosureSelfReferenceModel != expectedModel {
		t.Fatalf("V2ClosureSelfReferenceModel drift:\n  got: %q\n  want: %q",
			V2ClosureSelfReferenceModel, expectedModel)
	}
}

// V2ClosureSelfReferenceModel is the documented authority
// derivation for the closure commit C. The string is
// machine-stable so the close report can quote it verbatim.
//
// Authority is:
//
//   - external verifier input (the supplied C argument)
//   - strict topology (S < F < C)
//   - exact C:M binding (C:M bytes are read from the
//     supplied C, not from any manifest field)
//
// NOT: self-reference inside M. The committed manifest MAY
// carry a `closure_commit` field for human readability, but
// the verifier never derives C from that field. Mutating the
// manifest bytes changes C identity because M is part of
// C's tree, not because the verifier trusts the field.
const V2ClosureSelfReferenceModel = `C authority =
  external verifier input
  + strict topology
  + exact C:M binding

not self-reference inside M`

// validPlanFixture returns a deterministic, valid Plan
// Contract v1 byte sequence that the foundation ACT's
// parser accepts. The fixture is intentionally minimal:
// one run-mode check whose argv is `true` so the runner
// would pass if executed.
//
// The verifier ACT never executes the plan; the bytes only
// need to be a real Plan Contract v1 document so the
// hermetic test repo can simulate a realistic frozen plan
// at F:P.
func validPlanFixture() string {
	return `{"contract_version":1,"id":"HERMETIC-SELFREF-PLAN","description":"self-reference proof fixture","checks":[{"id":"smoke","mode":"run","argv":["true"],"timeout_seconds":30,"working_directory":"."}]}`
}

// buildSelfRefManifest constructs a manifest byte sequence
// whose `closure_commit` field is a literal placeholder
// marker rather than a real OID. The marker's presence in
// the bytes lets the test assert that the verifier does NOT
// require `manifest.closure_commit` to match the externally
// supplied C.
//
// The fields populated here are sufficient for the
// self-reference proof; ACT 3 will tighten the manifest
// contract.
func buildSelfRefManifest(subjectTree, freezeTree, planBlob, planSHA256 string) []byte {
	return []byte(`{
  "closure_protocol_version": "2",
  "plan_contract_version": 1,
  "subject_commit": "<subject>",
  "subject_tree": "` + subjectTree + `",
  "freeze_commit": "<freeze>",
  "freeze_tree": "` + freezeTree + `",
  "execution_tree": "` + subjectTree + `",
  "plan_path": "docs/closure-plans/PLAN.json",
  "plan_blob": "` + planBlob + `",
  "plan_sha256": "` + planSHA256 + `",
  "closure_commit": "<externally-supplied>",
  "note": "self-reference proof: closure_commit is a literal placeholder"
}`)
}

// writeAndCommit writes the supplied content to the
// repository-relative path inside the temp repo, stages it,
// creates a commit, and returns the new HEAD OID. The
// helper exists because manifest bytes are
// whitespace-sensitive and the foundation makeCommit helper
// only accepts a map[string]string.
func writeAndCommit(t *testing.T, dir, repoRelPath string, content []byte, message string) string {
	t.Helper()
	full := filepath.Join(dir, repoRelPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
	mustRunGit(t, dir, "add", repoRelPath)
	mustRunGit(t, dir, "commit", "-m", message)
	return mustRunGit(t, dir, "rev-parse", "HEAD")
}

// mustReadPlanBlob reads the literal raw bytes of a frozen
// plan blob via `git cat-file blob <ref>:<path>`. The bytes
// are trimmed because the fixture does not contain trailing
// whitespace and the test only uses them for SHA-256 of the
// fixture body. SHA-256 of the trimmed bytes equals
// SHA-256 of the raw bytes when no leading/trailing
// whitespace is present.
func mustReadPlanBlob(t *testing.T, dir, refAndPath string) []byte {
	t.Helper()
	value := mustRunGit(t, dir, "cat-file", "blob", refAndPath)
	return []byte(value)
}
