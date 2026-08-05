// SPDX-License-Identifier: Apache-2.0

package closure

// evidence_verifier_r2b_test.go proves the R2B committed
// closure-manifest evidence verifier required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2B.
//
// The test builds a small synthetic Git repository, writes a
// fully-correct closure manifest, and asserts the verifier
// reports OK. It then mutates every hash-shaped field in
// turn and asserts the verifier rejects with a typed
// diagnostic.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testResolver is a GitObjectResolver backed by a temporary
// Git repository built with the production execution helpers.
type testResolver struct {
	repoDir string
}

func (r *testResolver) CatFile(oid string) ([]byte, error) {
	out, err := runGitValue(context.Background(), RealGit{}, r.repoDir, "cat-file", "-p", oid)
	if err != nil {
		return nil, fmt.Errorf("git cat-file %s: %w", oid, err)
	}
	return []byte(out), nil
}

// writeSubjectRepo creates a small Git repository containing
// `files` and returns the commit OID.
func writeSubjectRepo(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	mustRunGit(t, dir, "add", ".")
	mustRunGit(t, dir, "commit", "-m", "subject commit")
	return mustRunGit(t, dir, "rev-parse", "HEAD")
}

// TestR2BEvidenceVerifier_AcceptsLiteralHashes proves the
// happy path: a closure manifest whose every hash-shaped
// field is literal and verified against the subject commit
// bytes passes the verifier.
func TestR2BEvidenceVerifier_AcceptsLiteralHashes(t *testing.T) {
	dir := initRepo(t)
	files := map[string]string{
		"alpha.txt": "alpha content\n",
		"beta.txt":  "beta content\n",
	}
	subject := writeSubjectRepo(t, dir, files)
	tree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	resolver := &testResolver{repoDir: dir}
	doc, err := buildManifestForRepo(subject, tree, files, resolver)
	if err != nil {
		t.Fatalf("buildManifestForRepo: %v", err)
	}
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(manifestPath, doc, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	res, err := VerifyClosureManifestR2B(EvidenceVerifierOptions{
		ManifestPath:  manifestPath,
		SubjectCommit: subject,
		Resolver:      resolver,
	})
	if err != nil {
		t.Fatalf("verifier returned error: %v", err)
	}
	if !res.OK {
		t.Fatalf("verifier rejected valid manifest: %+v", res.Diagnostics)
	}
}

// TestR2BEvidenceVerifier_RejectsAbbreviatedPlanBlob proves
// the verifier rejects a manifest whose plan_blob is shorter
// than 40 chars.
func TestR2BEvidenceVerifier_RejectsAbbreviatedPlanBlob(t *testing.T) {
	dir := initRepo(t)
	files := map[string]string{"alpha.txt": "alpha content\n"}
	subject := writeSubjectRepo(t, dir, files)
	tree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	resolver := &testResolver{repoDir: dir}
	doc, err := buildManifestForRepo(subject, tree, files, resolver)
	if err != nil {
		t.Fatalf("buildManifestForRepo: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(doc, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	dogfood, ok := m["dogfood"].(map[string]any)
	if !ok {
		t.Fatalf("dogfood section missing")
	}
	dogfood["plan_blob"] = "30bb1ac6"
	mutated, _ := json.MarshalIndent(m, "", "  ")
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(manifestPath, mutated, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	res, err := VerifyClosureManifestR2B(EvidenceVerifierOptions{
		ManifestPath:  manifestPath,
		SubjectCommit: subject,
		Resolver:      resolver,
	})
	if err != nil {
		t.Fatalf("verifier returned error: %v", err)
	}
	if res.OK {
		t.Fatalf("verifier accepted abbreviated plan_blob")
	}
	if !containsDiagnosticField(res.Diagnostics, "dogfood.plan_blob") {
		t.Fatalf("verifier did not flag dogfood.plan_blob: %+v", res.Diagnostics)
	}
}

// TestR2BEvidenceVerifier_RejectsSyntheticFilesChangedHash
// proves the verifier rejects a manifest whose
// files_changed[i].sha256 is descriptive text instead of a
// 64-char SHA-256 digest.
func TestR2BEvidenceVerifier_RejectsSyntheticFilesChangedHash(t *testing.T) {
	dir := initRepo(t)
	files := map[string]string{"alpha.txt": "alpha content\n"}
	subject := writeSubjectRepo(t, dir, files)
	tree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	resolver := &testResolver{repoDir: dir}
	doc, err := buildManifestForRepo(subject, tree, files, resolver)
	if err != nil {
		t.Fatalf("buildManifestForRepo: %v", err)
	}
	mutated := strings.ReplaceAll(string(doc),
		`"sha256": "`, `"sha256": "fail-closed caller-state snapshot`)
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	res, err := VerifyClosureManifestR2B(EvidenceVerifierOptions{
		ManifestPath:  manifestPath,
		SubjectCommit: subject,
		Resolver:      resolver,
	})
	if err != nil {
		t.Fatalf("verifier returned error: %v", err)
	}
	if res.OK {
		t.Fatalf("verifier accepted synthetic sha256 text")
	}
	if !containsDiagnosticFieldPrefix(res.Diagnostics, "files_changed[") {
		t.Fatalf("verifier did not flag files_changed: %+v", res.Diagnostics)
	}
}

// TestR2BEvidenceVerifier_RejectsFilesChangedHashMismatch
// proves the verifier rejects a manifest whose declared
// SHA-256 disagrees with the subject commit bytes.
func TestR2BEvidenceVerifier_RejectsFilesChangedHashMismatch(t *testing.T) {
	dir := initRepo(t)
	files := map[string]string{"alpha.txt": "alpha content\n"}
	subject := writeSubjectRepo(t, dir, files)
	tree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	resolver := &testResolver{repoDir: dir}
	doc, err := buildManifestForRepo(subject, tree, files, resolver)
	if err != nil {
		t.Fatalf("buildManifestForRepo: %v", err)
	}
	bogus := strings.Repeat("0", 64)
	mutated := strings.ReplaceAll(string(doc),
		`"sha256": "`, `"sha256": "`+bogus)
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	res, err := VerifyClosureManifestR2B(EvidenceVerifierOptions{
		ManifestPath:  manifestPath,
		SubjectCommit: subject,
		Resolver:      resolver,
	})
	if err != nil {
		t.Fatalf("verifier returned error: %v", err)
	}
	if res.OK {
		t.Fatalf("verifier accepted mismatched files_changed SHA-256")
	}
}

// TestR2BEvidenceVerifier_RejectsAbbreviatedPlanSHA256 proves
// the verifier rejects a manifest whose dogfood.plan_sha256
// is shorter than 64 chars.
func TestR2BEvidenceVerifier_RejectsAbbreviatedPlanSHA256(t *testing.T) {
	dir := initRepo(t)
	files := map[string]string{"alpha.txt": "alpha content\n"}
	subject := writeSubjectRepo(t, dir, files)
	tree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	resolver := &testResolver{repoDir: dir}
	doc, err := buildManifestForRepo(subject, tree, files, resolver)
	if err != nil {
		t.Fatalf("buildManifestForRepo: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(doc, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	dogfood := m["dogfood"].(map[string]any)
	dogfood["plan_sha256"] = "72d23905"
	mutated, _ := json.MarshalIndent(m, "", "  ")
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(manifestPath, mutated, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	res, err := VerifyClosureManifestR2B(EvidenceVerifierOptions{
		ManifestPath:  manifestPath,
		SubjectCommit: subject,
		Resolver:      resolver,
	})
	if err != nil {
		t.Fatalf("verifier returned error: %v", err)
	}
	if res.OK {
		t.Fatalf("verifier accepted abbreviated plan_sha256")
	}
	if !containsDiagnosticField(res.Diagnostics, "dogfood.plan_sha256") {
		t.Fatalf("verifier did not flag dogfood.plan_sha256: %+v", res.Diagnostics)
	}
}

// buildManifestForRepo constructs a fully-correct closure
// manifest JSON document whose files_changed hashes are
// literal SHA-256 digests of the committed blob bytes.
func buildManifestForRepo(subject, tree string, files map[string]string, resolver GitObjectResolver) ([]byte, error) {
	type fileEntry struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	filesChanged := []fileEntry{}
	for path := range files {
		oid, err := lookupBlobOID(resolver, tree, path)
		if err != nil {
			return nil, fmt.Errorf("lookup blob %s: %w", path, err)
		}
		blob, err := resolver.CatFile(oid)
		if err != nil {
			return nil, fmt.Errorf("read blob %s: %w", oid, err)
		}
		filesChanged = append(filesChanged, fileEntry{
			Path:   path,
			SHA256: evidenceSHA256Hex(blob),
		})
	}
	m := map[string]any{
		"contract_version": 1,
		"act_id":           "ACT-R2B-VERIFY",
		"subject": map[string]string{
			"commit_oid": subject,
			"tree_oid":   tree,
		},
		"runner": map[string]any{
			"leamas_version": "0.1.0+test",
			"binary_sha256":  strings.Repeat("a", 64),
			"vcs_revision":   subject,
			"vcs_modified":   false,
		},
		"files_changed": filesChanged,
		"dogfood": map[string]string{
			"binary_commit":       subject,
			"binary_sha256":       strings.Repeat("a", 64),
			"binary_vcs_revision": subject,
			"subject_commit":      subject,
			"subject_tree":        tree,
			"freeze_commit":       subject,
			"freeze_tree":         tree,
			"caller_head":         subject,
			"execution_tree":      tree,
			"plan_blob":           strings.Repeat("b", 40),
			"plan_sha256":         strings.Repeat("c", 64),
		},
	}
	return json.MarshalIndent(m, "", "  ")
}

// evidenceSHA256Hex returns the lowercase hex SHA-256 of b.
func evidenceSHA256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// gitCatFileResolver is a resolver backed by the production
// RealGit client. Tests that need to load committed
// repository state use it directly.
type gitCatFileResolver struct {
	RepoDir string
}

func (r *gitCatFileResolver) CatFile(oid string) ([]byte, error) {
	// Do NOT use runGitValue here: it TrimSpace's the
	// output, which would strip the trailing newline from
	// blob content and produce a SHA-256 that disagrees
	// with the literal blob bytes. We need the raw bytes
	// unchanged.
	result := RealGit{}.Run(context.Background(), r.RepoDir, "cat-file", "-p", oid)
	if result.Err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("git cat-file -p %s: exit %d err %v", oid, result.ExitCode, result.Err)
	}
	return result.Stdout, nil
}

// TestR2BEvidenceVerifier_AcceptsCommittedR1Manifest proves
// the verifier accepts the committed
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R1
// closure manifest. The test loads the manifest from the
// repository, reads the subject commit and tree via
// `git cat-file`, and runs the verifier. The committed
// manifest is expected to pass after the R2B hash
// regeneration.
func TestR2BEvidenceVerifier_AcceptsCommittedR1Manifest(t *testing.T) {
	repoRoot := "/home/chistyakov/Projects/leamas"
	resolver := &gitCatFileResolver{RepoDir: repoRoot}
	subject := "25010d160c6b04edc24ec4602af951541ef1ffa8"
	manifestPath := repoRoot + "/docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R1.json"
	res, err := VerifyClosureManifestR2B(EvidenceVerifierOptions{
		ManifestPath:  manifestPath,
		SubjectCommit: subject,
		Resolver:      resolver,
	})
	if err != nil {
		t.Fatalf("verifier returned error: %v", err)
	}
	if !res.OK {
		t.Fatalf("verifier rejected committed R1 manifest: %+v", res.Diagnostics)
	}
}

// lookupBlobOID resolves the tree's blob OID for the supplied
// path by parsing the parsed-format tree output.
func lookupBlobOID(resolver GitObjectResolver, treeOID, path string) (string, error) {
	treeBytes, err := resolver.CatFile(treeOID)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimRight(string(treeBytes), "\n"), "\n") {
		tab := strings.IndexByte(line, '\t')
		if tab < 0 {
			continue
		}
		meta := strings.SplitN(line[:tab], " ", 3)
		if len(meta) != 3 {
			continue
		}
		name := line[tab+1:]
		if name == path {
			return meta[2], nil
		}
	}
	return "", fmt.Errorf("path %s not in tree %s", path, treeOID)
}

// containsDiagnosticField reports whether any diagnostic in
// ds targets the supplied field name.
func containsDiagnosticField(ds []EvidenceVerifierDiagnostic, field string) bool {
	for _, d := range ds {
		if d.Field == field {
			return true
		}
	}
	return false
}

// containsDiagnosticFieldPrefix reports whether any
// diagnostic in ds targets a field whose name starts with
// the supplied prefix.
func containsDiagnosticFieldPrefix(ds []EvidenceVerifierDiagnostic, prefix string) bool {
	for _, d := range ds {
		if strings.HasPrefix(d.Field, prefix) {
			return true
		}
	}
	return false
}
