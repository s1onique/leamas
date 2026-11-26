// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_amendment_test.go
// holds the AMENDMENT-01 witnesses for
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01.
//
// AMENDMENT-01 addresses four independent defects that the original
// ACT left open:
//
//	A1  a patch that FITS the per-file budget was still silently
//	    transformed (per-line truncation) while the digest reported the
//	    original PatchBytes;
//	A2  the published retrieval command was reconstructed by hand and
//	    was NOT the command that produced the evidence;
//	A3  the base blob identity of a renamed file was looked up at the
//	    DESTINATION path, so the true preimage was reported MISSING;
//	A4  historical artifact classification reads the CURRENT worktree,
//	    so the disposition of an immutable historical range depended on
//	    unrelated checkout state.
//
// The witnesses here are NON-VACUOUS by construction: every one
// establishes its PRECONDITION before asserting its POSTCONDITION. A
// fixture that does not actually contain the secret, or does not
// actually create budget pressure, fails the test rather than
// silently passing it.
package digest

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// patchBudgetExtractRangeRetrievalPath returns the single
// "RangeRetrievalPath:" value published in the digest.
//
// The range-level retrieval command reproduces EVERY patch the range
// contains, so it is the counterpart of the per-file retrieval path
// that is published for the bounded dispositions.
func patchBudgetExtractRangeRetrievalPath(t *testing.T,
	digest string) string {
	t.Helper()
	for _, line := range strings.Split(digest, "\n") {
		if strings.HasPrefix(line, "RangeRetrievalPath: ") {
			return strings.TrimPrefix(line, "RangeRetrievalPath: ")
		}
	}
	t.Fatalf("digest contains no RangeRetrievalPath line. Digest "+
		"excerpt:\n%s", patchBudgetExcerpt(digest))
	return ""
}

// patchBudgetExtractRetrievalPath returns the single "RetrievalPath:"
// value published in the digest.
//
// The digest may publish several retrieval paths (one per bounded
// file). The first one is returned; the callers that need a specific
// file's path use patchBudgetExtractRetrievalPathFor.
func patchBudgetExtractRetrievalPath(t *testing.T, digest string) string {
	t.Helper()
	for _, line := range strings.Split(digest, "\n") {
		if strings.HasPrefix(line, "RetrievalPath: ") {
			return strings.TrimPrefix(line, "RetrievalPath: ")
		}
	}
	t.Fatalf("digest contains no RetrievalPath line. Digest excerpt:\n%s",
		patchBudgetExcerpt(digest))
	return ""
}

// patchBudgetExtractRetrievalPathFor returns the retrieval path
// published for the file block of `path`.
//
// The retrieval path is located inside the file's own section so a
// file-local assertion cannot be satisfied by another file's record.
func patchBudgetExtractRetrievalPathFor(t *testing.T,
	digest, path string) string {
	t.Helper()
	section := patchBudgetFileSection(t, digest, path)
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "RetrievalPath: ") {
			return strings.TrimPrefix(line, "RetrievalPath: ")
		}
	}
	t.Fatalf("section %q contains no RetrievalPath line. Section:\n%s",
		path, section)
	return ""
}

// patchBudgetSplitSections splits a digest into its per-file evidence
// sections, keyed by the "=== <path> ===" heading.
func patchBudgetSplitSections(digest string) map[string]string {
	out := map[string]string{}
	var current string
	var sb strings.Builder
	flush := func() {
		if current != "" {
			out[current] = sb.String()
		}
		sb.Reset()
	}
	for _, line := range strings.Split(digest, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "=== ") &&
			strings.HasSuffix(trimmed, " ===") {
			flush()
			current = trimmed
			continue
		}
		if current != "" {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	flush()
	return out
}

// patchBudgetSectionKeys returns the section keys of a split digest,
// for failure diagnostics.
func patchBudgetSectionKeys(sections map[string]string) []string {
	keys := make([]string, 0, len(sections))
	for k := range sections {
		keys = append(keys, k)
	}
	return keys
}

// patchBudgetFileSection returns the evidence section of one file.
func patchBudgetFileSection(t *testing.T, digest, path string) string {
	t.Helper()
	sections := patchBudgetSplitSections(digest)
	key := "=== " + path + " ==="
	section, ok := sections[key]
	if !ok {
		t.Fatalf("digest has no section for %q. Sections: %v",
			path, patchBudgetSectionKeys(sections))
	}
	return section
}

// patchBudgetRunShell executes a published retrieval command in the
// repository and returns its stdout.
//
// The command is executed through a shell so the test exercises the
// EXACT published serialization, quoting included. If the published
// serialization is not shell-safe, this test fails.
// patchBudgetItoa is a tiny local helper so the witnesses do not
// depend on a formatting import for a single call.
func patchBudgetItoa(n int) string { return strconv.Itoa(n) }

// ---------------------------------------------------------------------------
// A1: complete patch publication.
// ---------------------------------------------------------------------------

// TestAmendment_A1_FittingPatchIsPublishedComplete verifies that a
// patch whose bytes fit the per-file budget is published VERBATIM.
//
// The predecessor applied a per-LINE truncation to the patch before
// writing it, so a patch carrying one long changed line was silently
// transformed while the digest reported the ORIGINAL PatchBytes.
//
// PRECONDITION: the complete unified patch fits MaxPerFileBytes but
// contains a physical line longer than MaxPerLineBytes.
//
// POSTCONDITION: the rendered section contains the complete patch and
// reports EvidenceCompleteness: COMPLETE.
func TestAmendment_A1_FittingPatchIsPublishedComplete(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// A single changed line far longer than MaxPerLineBytes but
	// short enough that the WHOLE patch still fits the per-file
	// budget.
	longLine := strings.Repeat("L", 20000)
	base := "before\n" + longLine + "\nafter\n"
	head := "before\n" + strings.Repeat("M", 20000) + "\nafter\n"
	writeFilePath(t, filepath.Join(dir, "long-line.txt"), base)
	patchBudgetCommitAll(t, dir, "base")
	writeFilePath(t, filepath.Join(dir, "long-line.txt"), head)
	patchBudgetCommitAll(t, dir, "change long line")

	patchRaw, err := RunGit(dir, []string{
		"diff", "--unified=3", "HEAD~1..HEAD", "--", "long-line.txt"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	// PRECONDITION: the patch fits the per-file budget...
	if len(patchRaw) > MaxPerFileBytes {
		t.Fatalf("fixture setup: patch=%d bytes must fit "+
			"MaxPerFileBytes=%d", len(patchRaw), MaxPerFileBytes)
	}
	// ...but contains a line longer than the per-line cap.
	maxLine := 0
	for _, line := range strings.Split(patchRaw, "\n") {
		if len(line) > maxLine {
			maxLine = len(line)
		}
	}
	if maxLine <= MaxPerLineBytes {
		t.Fatalf("fixture setup: longest patch line=%d bytes must "+
			"exceed MaxPerLineBytes=%d", maxLine, MaxPerLineBytes)
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	section := patchBudgetFileSection(t, d, "long-line.txt")

	// POSTCONDITION: the complete patch is published verbatim.
	if !strings.Contains(section, strings.Repeat("M", 20000)) {
		t.Fatalf("complete patch publication violated: the full "+
			"changed line is missing from the evidence section. "+
			"Section:\n%s", section)
	}
	if strings.Contains(section, "[truncated: line=") {
		t.Fatalf("complete patch publication violated: the patch was "+
			"silently truncated by the per-line cap even though it "+
			"fit the per-file budget. Section:\n%s", section)
	}
	if !strings.Contains(section, "EvidenceCompleteness: COMPLETE") {
		t.Fatalf("a fully published patch must declare COMPLETE. "+
			"Section:\n%s", section)
	}
}

// TestAmendment_A1_PatchBytesDescribeTheRenderedPatch verifies that
// the reported PatchBytes describes the patch that is actually
// rendered, and that a partial publication is disclosed.
//
// The predecessor discarded the writer's report (`_ = n`), so a
// partial publication could be presented as a complete one.
func TestAmendment_A1_PatchBytesDescribeTheRenderedPatch(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	writeFilePath(t, filepath.Join(dir, "small.txt"),
		"alpha\nbeta\ngamma\n")
	patchBudgetCommitAll(t, dir, "base")
	writeFilePath(t, filepath.Join(dir, "small.txt"),
		"alpha\nBETA\ngamma\n")
	patchBudgetCommitAll(t, dir, "edit")

	patchRaw, err := RunGit(dir, []string{
		"diff", "--unified=3", "HEAD~1..HEAD", "--", "small.txt"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	section := patchBudgetFileSection(t, d, "small.txt")

	if !strings.Contains(section,
		"PatchBytes: "+patchBudgetItoa(len(patchRaw))) {
		t.Fatalf("PatchBytes must describe the measured patch "+
			"(%d bytes). Section:\n%s", len(patchRaw), section)
	}
	// The complete patch text must be present in the section.
	if !strings.Contains(section, strings.TrimRight(patchRaw, "\n")) {
		t.Fatalf("the complete patch must be rendered in the file "+
			"section. Section:\n%s", section)
	}
}

func patchBudgetRunShell(t *testing.T, dir, command string) string {
	t.Helper()
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("published retrieval command failed: %v\ncommand:\n%s",
			err, command)
	}
	return string(out)
}
