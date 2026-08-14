// SPDX-License-Identifier: Apache-2.0

// Package digest: range_scope_integration_test.go provides integration tests
// for the range-scope diagnostic integrated into RenderRangeDigestWithResolved.
package digest

import (
	"strings"
	"testing"
)

func TestRenderRangeDigestWithResolved_RangeScopeIntegration(t *testing.T) {
	dir := makeTestRepo(t)
	commitFile(t, dir, "file_a.txt", "content a\n")
	commitFile(t, dir, "file_b.txt", "content b\n")

	commitA := runGit(t, dir, "rev-parse", "HEAD~1")
	commitB := runGit(t, dir, "rev-parse", "HEAD")
	rangeSpec := commitA + ".." + commitB

	files := []RangeFile{
		{Path: "file_a.txt"},
		{Path: "file_b.txt"},
	}

	t.Run("explicit_cli_with_valid_range", func(t *testing.T) {
		resolved := &ResolvedMode{
			Mode:             ModeRange,
			Range:            rangeSpec,
			Reason:           "explicit CLI",
			ResolutionSource: "explicit_cli",
		}

		digest, err := RenderRangeDigestWithResolved(dir, files, resolved)
		if err != nil {
			t.Fatalf("RenderRangeDigestWithResolved failed: %v", err)
		}

		if !strings.Contains(digest, "## RANGE_SCOPE") {
			t.Error("explicit_cli should produce RANGE_SCOPE section")
		}
		if !strings.Contains(digest, "authority=explicit_range") {
			t.Error("RANGE_SCOPE should have explicit_range authority")
		}
		if !strings.Contains(digest, "diagnostic_status=available") {
			t.Error("valid range should have diagnostic_status=available")
		}
		if !strings.Contains(digest, "targetedness=NORMAL") {
			t.Error("2-file range should be NORMAL")
		}
		count := strings.Count(digest, "## RANGE_SCOPE")
		if count != 1 {
			t.Errorf("exactly one RANGE_SCOPE section expected, got %d", count)
		}
	})

	t.Run("explicit_cli_with_unavailable_scope", func(t *testing.T) {
		resolved := &ResolvedMode{
			Mode:             ModeRange,
			Range:            "A...B",
			Reason:           "explicit CLI with unsupported range",
			ResolutionSource: "explicit_cli",
		}

		digest, err := RenderRangeDigestWithResolved(dir, files, resolved)
		if err != nil {
			t.Fatalf("should not error on unavailable: %v", err)
		}

		if !strings.Contains(digest, "## RANGE_SCOPE") {
			t.Error("explicit_cli should produce RANGE_SCOPE section")
		}
		if !strings.Contains(digest, "targetedness=UNKNOWN") {
			t.Error("unavailable scope should have targetedness=UNKNOWN")
		}
		if !strings.Contains(digest, "diagnostic_status=unavailable") {
			t.Error("unavailable scope should have diagnostic_status=unavailable")
		}
	})

	t.Run("non_explicit_resolution_no_range_scope", func(t *testing.T) {
		resolved := &ResolvedMode{
			Mode:             ModeRange,
			Range:            rangeSpec,
			Reason:           "auto-detected",
			ResolutionSource: "auto",
		}

		digest, err := RenderRangeDigestWithResolved(dir, files, resolved)
		if err != nil {
			t.Fatalf("RenderRangeDigestWithResolved failed: %v", err)
		}

		if strings.Contains(digest, "## RANGE_SCOPE") {
			t.Error("non-explicit resolution should not produce RANGE_SCOPE section")
		}
	})

	t.Run("non_explicit_with_empty_range", func(t *testing.T) {
		resolved := &ResolvedMode{
			Mode:             ModeRange,
			Range:            "",
			Reason:           "auto-detected",
			ResolutionSource: "auto",
		}

		digest, err := RenderRangeDigestWithResolved(dir, files, resolved)
		if err != nil {
			t.Fatalf("RenderRangeDigestWithResolved failed: %v", err)
		}

		if strings.Contains(digest, "## RANGE_SCOPE") {
			t.Error("empty range should not produce RANGE_SCOPE section")
		}
	})
}

func TestRenderRangeDigestWithResolved_BroadRangeWarning(t *testing.T) {
	dir := makeTestRepo(t)
	commitFile(t, dir, "initial.txt", "initial\n")
	commitManyFiles(t, dir, 600)

	commitA := runGit(t, dir, "rev-parse", "HEAD~1")
	commitB := runGit(t, dir, "rev-parse", "HEAD")

	files := []RangeFile{}
	for i := 0; i < 600; i++ {
		files = append(files, RangeFile{Path: "generated/file_" + padNumber(i) + ".txt"})
	}

	resolved := &ResolvedMode{
		Mode:             ModeRange,
		Range:            commitA + ".." + commitB,
		Reason:           "explicit CLI",
		ResolutionSource: "explicit_cli",
	}

	digest, err := RenderRangeDigestWithResolved(dir, files, resolved)
	if err != nil {
		t.Fatalf("RenderRangeDigestWithResolved failed: %v", err)
	}

	if !strings.Contains(digest, "targetedness=BROAD") {
		t.Error("600-file range should be BROAD")
	}
	if !strings.Contains(digest, "files_changed=600") {
		t.Error("files_changed should match input")
	}
	if !strings.Contains(digest, "warning_code=DIGEST_RANGE_BROAD") {
		t.Error("BROAD range should have DIGEST_RANGE_BROAD warning code")
	}
	if !strings.Contains(digest, "mechanically valid but unusually broad") {
		t.Error("BROAD range should include warning prose")
	}
	if !strings.Contains(digest, "# Targeted digest") {
		t.Error("BROAD range digest should be a valid targeted digest")
	}
}

func TestRenderRangeDigestWithResolved_ExtremeRangeWarning(t *testing.T) {
	dir := makeTestRepo(t)
	commitFile(t, dir, "initial.txt", "initial\n")
	commitManyFiles(t, dir, 1001)

	commitA := runGit(t, dir, "rev-parse", "HEAD~1")
	commitB := runGit(t, dir, "rev-parse", "HEAD")

	files := []RangeFile{}
	for i := 0; i < 1001; i++ {
		files = append(files, RangeFile{Path: "generated/file_" + padNumber(i) + ".txt"})
	}

	resolved := &ResolvedMode{
		Mode:             ModeRange,
		Range:            commitA + ".." + commitB,
		Reason:           "explicit CLI",
		ResolutionSource: "explicit_cli",
	}

	digest, err := RenderRangeDigestWithResolved(dir, files, resolved)
	if err != nil {
		t.Fatalf("RenderRangeDigestWithResolved failed: %v", err)
	}

	if !strings.Contains(digest, "targetedness=EXTREME") {
		t.Error("1001-file range should be EXTREME")
	}
	if !strings.Contains(digest, "files_changed=1001") {
		t.Error("files_changed should match input")
	}
	if !strings.Contains(digest, "warning_code=DIGEST_RANGE_EXTREME") {
		t.Error("EXTREME range should have DIGEST_RANGE_EXTREME warning code")
	}
	if !strings.Contains(digest, "exceptionally large change surface") {
		t.Error("EXTREME range should include warning prose")
	}
}
