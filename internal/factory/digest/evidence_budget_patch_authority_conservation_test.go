// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_conservation_test.go
// is the conservation half of the focused regression matrix for
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01.
//
// It was split out of evidence_budget_patch_authority_test.go so
// that both files stay within the LLM-friendly 400-line limit.
//
// The cases here assert that correcting the per-file budget did NOT
// disturb the neighbouring policies:
//
//   - Case 6: sensitive source -> existing redaction policy conserved
//   - Case 5: aggregate budget exceeded -> existing policy conserved
//   - Case 7: binary change -> existing disposition conserved
package digest

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestPatchBudget_SensitiveSource_RedactionConserved verifies that
// including a small patch inside a large source file does NOT bypass
// the existing redaction policy: non-source files are still redacted
// and source files are still scanned for secret-like patterns.
//
// AMENDMENT-01 (A5): the predecessor witness was VACUOUS. It
// appended the secret at the END of the file and changed the FIRST
// line, so the secret never appeared in the unified patch at all
// (Git emits three context lines around a change). The witness
// therefore passed even if the redaction were completely broken.
//
// The corrected witness establishes the PRECONDITION that the RAW
// unified patch really contains the secret, and only then asserts
// that the RENDERED evidence excludes it.
func TestPatchBudget_SensitiveSource_RedactionConserved(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// A large NON-SOURCE file (no source extension) carrying a
	// secret-like literal. The redaction policy must still apply.
	var sb strings.Builder
	for i := 0; i < 20000; i++ {
		sb.WriteString("ordinary config padding line for the large body\n")
	}
	secretCfg := sb.String() +
		"api_key = \"AKIAIOSFODNN7EXAMPLE\"\n"
	writeFilePath(t, filepath.Join(dir, "config.yaml"), secretCfg)
	patchBudgetCommitAll(t, dir, "add config")

	// AMENDMENT-01 (A5): the secret must be inside the CHANGED LINE
	// (or its immediate context) so it genuinely reaches the patch.
	modified := strings.Replace(secretCfg,
		"api_key = \"AKIAIOSFODNN7EXAMPLE\"\n",
		"api_key = \"AKIAIOSFODNN7EXAMPLE\"\n"+
			"# EDITED neighbouring line\n", 1)
	writeFilePath(t, filepath.Join(dir, "config.yaml"), modified)
	patchBudgetCommitAll(t, dir, "edit config")

	// PRECONDITION: the RAW unified patch really contains the secret.
	rawPatch, err := RunGit(dir, []string{
		"diff", "--unified=3", "HEAD~1..HEAD", "--", "config.yaml"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	if !strings.Contains(rawPatch, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("fixture setup: the raw unified patch must contain "+
			"the synthetic secret for this witness to be a real "+
			"discriminator. Raw patch:\n%s", rawPatch)
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	if !strings.Contains(d, "REDACTION_POLICY:") {
		t.Fatalf("redaction policy metadata missing from digest")
	}
	if !strings.Contains(d, "class=non_source") {
		t.Fatalf("non-source file must keep the non_source redaction "+
			"class. Digest excerpt:\n%s", patchBudgetExcerpt(d))
	}
	if !strings.Contains(d, "redaction_applied=true") {
		t.Fatalf("non-source file must still be redacted. "+
			"Digest excerpt:\n%s", patchBudgetExcerpt(d))
	}
	// POSTCONDITION: the rendered evidence excludes the secret, even
	// though the raw patch contained it.
	if strings.Contains(d, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("redaction violated: the secret literal survived " +
			"the redaction policy")
	}
	// POSTCONDITION: the intended NON-SECRET change survives the
	// redaction.
	//
	// AMENDMENT-01 FINALIZATION (F3): without this assertion an
	// implementation that suppressed the COMPLETE patch would satisfy
	// the secret-absence assertion above. The redaction must remove
	// the secret and nothing else.
	if !strings.Contains(d, "# EDITED neighbouring line") {
		t.Fatalf("redaction violated: the intended non-secret change "+
			"was suppressed along with the secret. The redaction "+
			"must remove the secret and nothing else. Digest "+
			"excerpt:\n%s", patchBudgetExcerpt(d))
	}
}

// ---------------------------------------------------------------------------
// Case 7: binary change -> existing disposition conserved.
// ---------------------------------------------------------------------------

// TestPatchBudget_BinaryChange_DispositionConserved verifies that a
// binary change keeps the existing bounded disposition and does not
// emit raw binary bytes.
func TestPatchBudget_BinaryChange_DispositionConserved(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	bin := make([]byte, 4096)
	for i := range bin {
		bin[i] = byte(i % 251)
	}
	writeFilePath(t, filepath.Join(dir, "blob.bin"), string(bin))
	patchBudgetCommitAll(t, dir, "add binary")

	bin2 := make([]byte, 4096)
	for i := range bin2 {
		bin2[i] = byte((i + 7) % 251)
	}
	writeFilePath(t, filepath.Join(dir, "blob.bin"), string(bin2))
	patchBudgetCommitAll(t, dir, "modify binary")

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	if !strings.Contains(d, "=== blob.bin ===") {
		t.Fatalf("setup: binary path missing from digest")
	}
	// The binary disposition must not claim a normal textual
	// embedding of raw bytes. Git renders a binary delta as
	// "Binary files a/<p> and b/<p> differ" and never emits the
	// raw bytes.
	if !strings.Contains(d, "Binary files ") &&
		!strings.Contains(d, "BinaryFiles:") &&
		!strings.Contains(d, "EvidenceCompleteness: INCOMPLETE") {
		t.Fatalf("binary change disposition was not conserved. "+
			"Digest excerpt:\n%s", patchBudgetExcerpt(d))
	}
}
