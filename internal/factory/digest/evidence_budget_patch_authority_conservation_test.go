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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	modified := strings.Replace(secretCfg,
		"ordinary config padding line for the large body\n",
		"ordinary config padding line for the large body EDITED\n", 1)
	writeFilePath(t, filepath.Join(dir, "config.yaml"), modified)
	patchBudgetCommitAll(t, dir, "edit config")

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
	if strings.Contains(d, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("redaction violated: the secret literal survived " +
			"the redaction policy")
	}
}

// ---------------------------------------------------------------------------
// Case 5: aggregate budget exceeded -> existing policy conserved.
// ---------------------------------------------------------------------------

// TestPatchBudget_AggregateBudgetExceeded_Conserved verifies that the
// aggregate (total) render budget still clamps the digest, exactly as
// before this ACT.
func TestPatchBudget_AggregateBudgetExceeded_Conserved(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// Many small-patch files, each inside a large source file, so
	// that the per-file budget never fires but the aggregate does.
	for i := 0; i < 40; i++ {
		var sb strings.Builder
		for j := 0; j < 20000; j++ {
			sb.WriteString("aggregate pressure padding content line\n")
		}
		writeFilePath(t, filepath.Join(dir, fmt.Sprintf("agg_%02d.ts", i)), sb.String())
	}
	patchBudgetCommitAll(t, dir, "add many large sources")

	for i := 0; i < 40; i++ {
		p := filepath.Join(dir, fmt.Sprintf("agg_%02d.ts", i))
		body, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		edited := strings.Replace(string(body),
			"aggregate pressure padding content line\n",
			"aggregate pressure padding content line EDIT\n", 1)
		writeFilePath(t, p, edited)
	}
	patchBudgetCommitAll(t, dir, "edit many large sources")

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	if len(d) > MaxTotalRenderBytes+4*1024 {
		t.Fatalf("aggregate budget violated: digest=%d bytes "+
			"exceeds the %d byte total budget ceiling",
			len(d), MaxTotalRenderBytes)
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
