// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_final_endpoint_test.go
// holds the G3 fail-closed comparison-authority witnesses for
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 FINAL AUTHORITY CORRECTION.
//
// INDEPENDENTLY REPRODUCED DEFECT: a `git diff` invocation WITHOUT
// comparison endpoints does not fail. It compares the mutable WORKTREE
// against the index and SUCCEEDS, so a patch can be produced against
// state that is not the historical range the evidence claims to
// describe.
//
// The companion files hold the A1/A2, A3/A4 and G2 witnesses.
package digest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// G3: fail-closed comparison authority.
// ---------------------------------------------------------------------------

// TestAmendment_G3_UnresolvedEndpointsAreNotCompared verifies that a
// PatchSpec without resolved comparison endpoints does NOT produce a
// diff argument vector.
//
// INDEPENDENTLY REPRODUCED DEFECT: `git diff --unified=3 -- :(literal)p`
// compares the WORKTREE against the INDEX and exits 0. A canonical
// specification that silently dropped its comparison argument therefore
// produced a SUCCESSFUL comparison against the wrong authority.
//
// POSTCONDITION: the specification reports that it has no comparison,
// and its argument vector carries NO diff argument that could compare
// mutable state.
func TestAmendment_G3_UnresolvedEndpointsAreNotCompared(t *testing.T) {
	spec := newPatchSpec("", "", "", "", false, RangeFile{
		Path: "f.txt", Status: "M"})

	if spec.HasComparison() {
		t.Fatalf("a specification without resolved endpoints must " +
			"report that it has no comparison")
	}
	if cmp := spec.Comparison(); cmp != "" {
		t.Fatalf("a specification without resolved endpoints must not "+
			"expose a comparison expression, got %q", cmp)
	}

	// The argument vector must NOT contain a comparison that Git
	// would resolve against mutable state. The only permitted
	// arguments are the diff subcommand, the diff options, the
	// rename-detection options, the "--" separator and the literal
	// pathspecs.
	args := spec.Args()
	for i, a := range args {
		if a == "diff" {
			continue
		}
		if a == "--" || strings.HasPrefix(a, ":(") {
			continue
		}
		if strings.HasPrefix(a, "--") {
			continue
		}
		t.Fatalf("the argument vector must not contain a comparison "+
			"expression: argument %d is %q (vector %v)", i, a, args)
	}
}

// TestAmendment_G3_DiffOptionsDisableExternalDiff verifies that the
// canonical diff options disable the external-diff driver and the
// textconv filters.
//
// INDEPENDENTLY REPRODUCED DEFECT: `git diff` honours `diff.external`
// and the `diff` textconv attribute by DEFAULT. With an external
// driver configured, Git emits the driver's output INSTEAD of the
// patch, so the published "PatchBytes" would describe bytes that are
// not a unified diff at all.
//
// POSTCONDITION: the canonical argument vector carries the disabling
// options, so the published retrieval command reproduces the same
// bytes as the generated patch.
func TestAmendment_G3_DiffOptionsDisableExternalDiff(t *testing.T) {
	args := newPatchSpec("aaa", "bbb", "", "", false, RangeFile{
		Path: "f.txt", Status: "M"}).Args()

	hasNoExtDiff := false
	hasNoTextconv := false
	for _, a := range args {
		switch a {
		case "--no-ext-diff":
			hasNoExtDiff = true
		case "--no-textconv":
			hasNoTextconv = true
		}
	}
	if !hasNoExtDiff {
		t.Fatalf("the canonical diff options must disable the "+
			"external-diff driver, got %v", args)
	}
	if !hasNoTextconv {
		t.Fatalf("the canonical diff options must disable the "+
			"textconv filters, got %v", args)
	}
}

// TestAmendment_G3_ExternalDiffCannotReplaceThePatch verifies that an
// external diff driver CANNOT replace the published patch.
//
// The witness installs a real external diff driver through the
// repository's Git configuration and then asserts that the canonical
// argument vector still yields the genuine unified patch.
//
// This is the OBSERVABLE consequence of the option guarantee: the
// evidence bytes stay a unified diff even when the environment
// configures an external driver.
func TestAmendment_G3_ExternalDiffCannotReplaceThePatch(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	writeFilePath(t, filepath.Join(dir, "f.txt"), "alpha\n")
	patchBudgetCommitAll(t, dir, "base")
	writeFilePath(t, filepath.Join(dir, "f.txt"), "beta\n")
	patchBudgetCommitAll(t, dir, "change")

	// A real external diff driver that would replace the patch.
	driver := filepath.Join(dir, "fake-ext-diff.sh")
	if err := os.WriteFile(driver, []byte(
		"#!/bin/sh\necho 'FAKE EXTERNAL DIFF OUTPUT'\n"), 0o755); err != nil {
		t.Fatalf("write the external diff driver: %v", err)
	}
	setGitConfig(t, dir, "diff.external", driver)

	spec := patchBudgetResolvedSpec(t, dir, "f.txt")

	// PRECONDITION: without the disabling option, the external driver
	// DOES replace the patch. This proves the witness is non-vacuous:
	// it is the canonical options, not the fixture, that make the
	// difference.
	unprotected, unprotectedErr := RunGit(dir, []string{
		"diff", "--unified=3", spec.Comparison(), "--",
		literalPathspec("f.txt")})
	if unprotectedErr != nil {
		t.Fatalf("fixture setup: the unprotected diff must succeed: %v",
			unprotectedErr)
	}
	if !strings.Contains(unprotected, "FAKE EXTERNAL DIFF OUTPUT") {
		t.Fatalf("fixture setup: without the disabling option the "+
			"external driver must replace the patch, got %q",
			unprotected)
	}

	// The genuine patch, produced with the canonical options.
	genOut, genErr := RunGit(dir, spec.Args())
	if genErr != nil {
		t.Fatalf("the canonical diff must succeed: %v", genErr)
	}
	if !strings.Contains(genOut, "diff --git") {
		t.Fatalf("the canonical diff must produce a unified patch, "+
			"got %q", genOut)
	}
	if strings.Contains(genOut, "FAKE EXTERNAL DIFF OUTPUT") {
		t.Fatalf("the external diff driver replaced the patch: the "+
			"canonical options must disable it. Got %q", genOut)
	}

	// The published retrieval command must reproduce the SAME bytes.
	published := spec.RetrievalArgv()
	reproOut := patchBudgetRunShell(t, dir, published)
	if reproOut != genOut {
		t.Fatalf("the published retrieval command must reproduce the "+
			"generated patch byte for byte.\npublished:\n%s\ngenerated:\n%s",
			published, reproOut)
	}
}

// TestAmendment_G3_PublishedRetrievalIsLossless verifies that the
// published retrieval specification remains LOSSLESS for a filename
// that contains shell metacharacters.
//
// The retrieval command is the reviewer's only way to reconstruct the
// evidence. A filename containing a space, a quote or a glob character
// must survive the serialization unchanged, so that re-executing the
// published command reproduces the patch.
func TestAmendment_G3_PublishedRetrievalIsLossless(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// A filename with a space, a quote and a glob character.
	name := "odd name's [x]*.txt"
	writeFilePath(t, filepath.Join(dir, name), "alpha\n")
	patchBudgetCommitAll(t, dir, "base")
	writeFilePath(t, filepath.Join(dir, name), "beta\n")
	patchBudgetCommitAll(t, dir, "change")

	spec := patchBudgetResolvedSpec(t, dir, name)

	genOut, genErr := RunGit(dir, spec.Args())
	if genErr != nil {
		t.Fatalf("the canonical diff must succeed: %v", genErr)
	}
	if !strings.Contains(genOut, "diff --git") {
		t.Fatalf("the canonical diff must produce a unified patch, "+
			"got %q", genOut)
	}

	// The published command must reproduce the SAME bytes.
	published := spec.RetrievalArgv()
	reproOut := patchBudgetRunShell(t, dir, published)
	if reproOut != genOut {
		t.Fatalf("the published retrieval command must reproduce the "+
			"generated patch byte for byte.\npublished:\n%s\ngenerated:\n%s",
			published, reproOut)
	}
}

// patchBudgetResolvedSpec builds a canonical specification whose
// comparison endpoints are RESOLVED to immutable object IDs.
//
// The helper exists because the G3 witnesses need a real comparison
// authority: a specification without endpoints cannot produce a patch
// at all (which is the point of the fail-closed contract).
func patchBudgetResolvedSpec(t *testing.T, dir, path string) PatchSpec {
	t.Helper()
	base, head, baseFallback, headFallback, dotDot :=
		resolvePatchEndpoints(realGitRunner{}, dir, "HEAD~1..HEAD",
			"", "")
	if base == "" || head == "" {
		t.Fatalf("fixture setup: the comparison endpoints must "+
			"resolve, got base=%q head=%q", base, head)
	}
	return newPatchSpec(base, head, baseFallback, headFallback,
		dotDot, RangeFile{Path: path, Status: "M"})
}

// setGitConfig sets a Git configuration value in the repository.
func setGitConfig(t *testing.T, dir, key, value string) {
	t.Helper()
	cmd := exec.Command("git", "config", key, value)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config %s: %v\n%s", key, err, string(out))
	}
}
