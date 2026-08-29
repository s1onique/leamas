// SPDX-License-Identifier: Apache-2.0

// Path-safety, redaction, and immutability tests for the local
// evidence overlay MVP. The "happy path" collection tests live
// in evidence_overlay_test.go; this file concentrates tests that
// protect the user from misconfiguration, repository escape,
// unauthorized disclosure, and unwanted Git state mutation.
package digest

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/redact"
)

// ----- M09: symlink rejected ----------------------------------------

// TestEvidence_M09a_RootIsSymlink_Rejected covers the case
// where the user-provided --evidence-path argument names a
// symlink itself. The MVP refuses to follow it. C2 establishes
// the fail-closed whole-request contract: any rejected root
// returns a non-nil error.
func TestEvidence_M09a_RootIsSymlink_Rejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	dir := makeEvidenceFixture(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"),
		[]byte("ext\n"), 0644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "linked")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	gi := filepath.Join(dir, ".gitignore")
	b, _ := os.ReadFile(gi)
	if err := os.WriteFile(gi, append(b, []byte("linked/\n")...), 0644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	roots, err := ValidateEvidenceRoots(dir, []string{"linked"})
	if err == nil {
		t.Fatalf("expected error (fail-closed); roots=%+v", roots)
	}
	if len(roots) == 0 || !roots[0].Rejected {
		t.Fatalf("diagnostic slice should mark root rejected, got %+v", roots)
	}
}

// TestEvidence_M09b_SymlinkFileDropped covers a non-symlink
// evidence root that contains a symlink under it. The root is
// accepted (the user explicitly authorized it), but the
// symlink inside is silently dropped at collection time so no
// out-of-repository content ever enters the rendered digest.
func TestEvidence_M09b_SymlinkFileDropped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	dir := makeEvidenceFixture(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"),
		[]byte("ext\n"), 0644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "ignored-sym"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	target := filepath.Join(outside, "secret.txt")
	if err := os.Symlink(target,
		filepath.Join(dir, "ignored-sym", "link")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	gi := filepath.Join(dir, ".gitignore")
	b, _ := os.ReadFile(gi)
	if err := os.WriteFile(gi,
		append(b, []byte("ignored-sym/\n")...), 0644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-sym"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(roots) == 0 || roots[0].Rejected {
		t.Fatalf("expected root to be accepted, got %+v", roots)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	for _, f := range overlay.Files {
		if strings.HasSuffix(f.Path, "link") {
			t.Errorf("M09b violation: symlink file appears in Files")
		}
	}
	// C3: the symlink is recorded as an omission rather than
	// silently dropped, so reviewers can see what happened.
	recorded := false
	for _, om := range overlay.Omitted {
		if strings.HasSuffix(om.Path, "link") && om.Reason == OmitSymlink {
			recorded = true
		}
	}
	if !recorded {
		t.Errorf("M09b: symlink not recorded as OmitSymlink omission: %+v", overlay.Omitted)
	}
}

// ----- M10: redaction applies ---------------------------------------

func TestEvidence_Render_M10_RedactionReusesEngine(t *testing.T) {
	dir := makeEvidenceFixture(t)
	secretContent := "leak: sk-1234567890abcdefghijklmnopqrstuvwxyz\n"
	if err := os.WriteFile(filepath.Join(dir, "ignored-a", "token.txt"),
		[]byte(secretContent), 0644); err != nil {
		t.Fatalf("write token: %v", err)
	}
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	rendered := RenderEvidenceOverlay(overlay)
	out := redact.RedactDigest(rendered)
	if strings.Contains(out, "sk-1234567890") {
		t.Errorf("M10: evidence content leaked an OpenAI-shaped secret")
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("M10: expected [REDACTED] marker")
	}
}

// TestEvidence_Write_M10b_RedactionViaWritePath proves the
// production Write() pipeline applies the same redaction
// policy to evidence content that it applies to ordinary
// digest content.
func TestEvidence_Write_M10b_RedactionViaWritePath(t *testing.T) {
	dir := makeEvidenceFixture(t)
	secretContent := "leak: sk-1234567890abcdefghijklmnopqrstuvwxyz\n"
	if err := os.WriteFile(filepath.Join(dir, "ignored-a", "token2.txt"),
		[]byte(secretContent), 0644); err != nil {
		t.Fatalf("write token: %v", err)
	}
	output := filepath.Join(dir, "digest.md")
	if err := Write(Options{
		RepoRoot:      dir,
		Mode:          ModeDirty,
		Output:        output,
		EvidencePaths: []string{"ignored-a"},
	}); err != nil {
		t.Fatalf("write: %v", err)
	}
	written, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	body := string(written)
	if strings.Contains(body, "sk-1234567890") {
		t.Errorf("M10b: production Write() leaked an OpenAI-shaped secret")
	}
	if !strings.Contains(body, "[REDACTED]") {
		t.Errorf("M10b: production Write() must apply redaction to evidence")
	}
}

// ----- M11: no-flag compatibility -----------------------------------

func TestEvidence_NoFlag_M11_Compatibility(t *testing.T) {
	dir := makeEvidenceFixture(t)
	baseOut, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("base generate: %v", err)
	}
	emptyOut, err := Generate(Options{
		RepoRoot:      dir,
		Mode:          ModeDirty,
		EvidencePaths: []string{},
	})
	if err != nil {
		t.Fatalf("empty generate: %v", err)
	}
	if baseOut != emptyOut {
		t.Errorf("M11 violation: empty evidence slice changed digest output")
	}
}

// ----- M12: collection does not mutate Git -------------------------

func TestEvidence_M12_GitImmutability(t *testing.T) {
	dir := makeEvidenceFixture(t)
	beforeHead := RunGitForTest(dir, []string{"rev-parse", "HEAD"})
	beforeTree := RunGitForTest(dir, []string{"rev-parse", "HEAD^{tree}"})
	beforePorcelain := RunGitForTest(dir, []string{"status", "--porcelain=v1",
		"--untracked-files=all"})
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if _, err := CollectEvidenceOverlay(dir, roots); err != nil {
		t.Fatalf("collect: %v", err)
	}
	afterHead := RunGitForTest(dir, []string{"rev-parse", "HEAD"})
	afterTree := RunGitForTest(dir, []string{"rev-parse", "HEAD^{tree}"})
	afterPorcelain := RunGitForTest(dir, []string{"status", "--porcelain=v1",
		"--untracked-files=all"})
	if beforeHead != afterHead {
		t.Errorf("M12: HEAD moved")
	}
	if beforeTree != afterTree {
		t.Errorf("M12: tree moved")
	}
	if beforePorcelain != afterPorcelain {
		t.Errorf("M12: porcelain status changed")
	}
}

// ----- M13..M22 highly-desirable matrix rows ------------------------

func TestEvidence_M13_DuplicateRootsDeterministic(t *testing.T) {
	dir := makeEvidenceFixture(t)
	roots, err := ValidateEvidenceRoots(dir,
		[]string{"ignored-a", "ignored-a", "ignored-a"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(roots) != 1 {
		t.Errorf("expected dedup to 1, got %d", len(roots))
	}
}

func TestEvidence_M14_OverlappingRootsNoDoubleCount(t *testing.T) {
	dir := makeEvidenceFixture(t)
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a", "ignored-a"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	seen := make(map[string]int)
	for _, f := range overlay.Files {
		seen[f.Path]++
	}
	for p, n := range seen {
		if n > 1 {
			t.Errorf("M14 violation: %s appeared %d times", p, n)
		}
	}
}

func TestEvidence_M15_DeterministicOrder(t *testing.T) {
	dir := makeEvidenceFixture(t)
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	first, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("first collect: %v", err)
	}
	second, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("second collect: %v", err)
	}
	if len(first.Files) != len(second.Files) {
		t.Fatalf("len mismatch: %d vs %d", len(first.Files), len(second.Files))
	}
	for i := range first.Files {
		if first.Files[i].Path != second.Files[i].Path {
			t.Errorf("ordering non-deterministic at %d", i)
		}
	}
}

func TestEvidence_M20_TrackedFileNotPromotedToEvidence(t *testing.T) {
	dir := makeEvidenceFixture(t)
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	for _, f := range overlay.Files {
		if f.Path == "tracked-evidence.log" {
			t.Errorf("M20 violation: tracked file promoted to evidence")
		}
	}
}

func TestEvidence_M22_EmptyOverlayTruthful(t *testing.T) {
	rendered := RenderEvidenceOverlay(EvidenceOverlay{})
	if !strings.Contains(rendered, "(no evidence roots requested)") {
		t.Errorf("expected truthful empty-overlay message, got: %s", rendered)
	}
}

func TestEvidence_Options_SignalsRequest(t *testing.T) {
	if IsEvidenceRequested(Options{}) {
		t.Errorf("empty Options should not signal evidence request")
	}
	if !IsEvidenceRequested(Options{EvidencePaths: []string{"x"}}) {
		t.Errorf("non-empty EvidencePaths should signal evidence request")
	}
}
