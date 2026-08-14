// SPDX-License-Identifier: Apache-2.0

// Package digest: range_scope_git_test.go provides hermetic Git fixture tests
// for the range scope collector.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "--initial-branch=main")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	return dir
}

func commitFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", filename)
	runGit(t, dir, "commit", "-m", "add "+filename)
}

func commitManyFiles(t *testing.T, dir string, count int) {
	t.Helper()
	generatedDir := filepath.Join(dir, "generated")
	if err := os.MkdirAll(generatedDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for i := 0; i < count; i++ {
		path := filepath.Join(generatedDir, "file_"+padNumber(i)+".txt")
		if err := os.WriteFile(path, []byte("content "+padNumber(i)+"\n"), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	runGit(t, dir, "add", "generated/")
	runGit(t, dir, "commit", "-m", "add "+padNumber(count)+" files")
}

func padNumber(n int) string {
	if n < 10 {
		return "000" + itoa(n)
	}
	if n < 100 {
		return "00" + itoa(n)
	}
	if n < 1000 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func TestRangeScope_FixtureA_SmallLinear(t *testing.T) {
	dir := makeTestRepo(t)
	commitFile(t, dir, "initial.txt", "initial\n")
	commitFile(t, dir, "file_a.txt", "content a\n")
	commitFile(t, dir, "file_b.txt", "content b\n")
	commitFile(t, dir, "file_c.txt", "content c\n")

	commitA := runGit(t, dir, "rev-parse", "HEAD~2")
	commitC := runGit(t, dir, "rev-parse", "HEAD")

	scope := CollectRangeScope(dir, commitA+".."+commitC, 2)

	if scope.CommitCount != 2 {
		t.Errorf("CommitCount = %d, want 2", scope.CommitCount)
	}
	if scope.MergeCommitCount != 0 {
		t.Errorf("MergeCommitCount = %d, want 0", scope.MergeCommitCount)
	}
	if scope.CrossesMerge {
		t.Error("CrossesMerge = true, want false")
	}
	if scope.FilesChanged != 2 {
		t.Errorf("FilesChanged = %d, want 2", scope.FilesChanged)
	}
	if scope.Targetedness != RangeTargetednessNormal {
		t.Errorf("Targetedness = %s, want NORMAL", scope.Targetedness)
	}
}

func TestRangeScope_FixtureB_MergeCrossing(t *testing.T) {
	dir := makeTestRepo(t)
	commitFile(t, dir, "main_0.txt", "main 0\n")
	runGit(t, dir, "checkout", "-b", "feature")
	commitFile(t, dir, "feature_1.txt", "feature 1\n")
	commitFile(t, dir, "feature_2.txt", "feature 2\n")
	runGit(t, dir, "checkout", "main")
	commitFile(t, dir, "main_1.txt", "main 1\n")
	commitFile(t, dir, "main_2.txt", "main 2\n")
	runGit(t, dir, "merge", "-m", "merge feature", "feature")

	commitBeforeMerge := runGit(t, dir, "rev-parse", "HEAD~3")
	commitAfterMerge := runGit(t, dir, "rev-parse", "HEAD")

	scope := CollectRangeScope(dir, commitBeforeMerge+".."+commitAfterMerge, 5)

	if scope.MergeCommitCount < 1 {
		t.Errorf("MergeCommitCount = %d, want >= 1", scope.MergeCommitCount)
	}
	if !scope.CrossesMerge {
		t.Error("CrossesMerge = false, want true")
	}
	if scope.Targetedness != RangeTargetednessNormal {
		t.Errorf("Targetedness = %s, want NORMAL", scope.Targetedness)
	}
}

func TestRangeScope_FixtureD_ExtremeFileSurface(t *testing.T) {
	dir := makeTestRepo(t)
	commitFile(t, dir, "initial.txt", "initial\n")
	commitManyFiles(t, dir, 1001)

	commitStart := runGit(t, dir, "rev-parse", "HEAD~1")
	commitEnd := runGit(t, dir, "rev-parse", "HEAD")

	scope := CollectRangeScope(dir, commitStart+".."+commitEnd, 1001)

	if scope.Targetedness != RangeTargetednessExtreme {
		t.Errorf("Targetedness = %s, want EXTREME", scope.Targetedness)
	}
	if scope.WarningCode != WarningCodeDigestRangeExtreme {
		t.Errorf("WarningCode = %s, want DIGEST_RANGE_EXTREME", scope.WarningCode)
	}
}

func TestRangeScope_EmptyRange(t *testing.T) {
	dir := makeTestRepo(t)
	commitFile(t, dir, "only.txt", "only\n")
	commit := runGit(t, dir, "rev-parse", "HEAD")

	scope := CollectRangeScope(dir, commit+".."+commit, 0)

	if scope.Targetedness != RangeTargetednessNormal {
		t.Errorf("Targetedness = %s, want NORMAL", scope.Targetedness)
	}
}

func TestRangeScope_EndpointResolution(t *testing.T) {
	dir := makeTestRepo(t)
	commitFile(t, dir, "a.txt", "a\n")
	commitFile(t, dir, "b.txt", "b\n")

	head1 := runGit(t, dir, "rev-parse", "HEAD~1")
	head2 := runGit(t, dir, "rev-parse", "HEAD")

	scope := CollectRangeScope(dir, "HEAD~1..HEAD", 1)

	if scope.LeftEndpointOID != head1 {
		t.Errorf("LeftEndpointOID = %s, want %s", scope.LeftEndpointOID, head1)
	}
	if scope.RightEndpointOID != head2 {
		t.Errorf("RightEndpointOID = %s, want %s", scope.RightEndpointOID, head2)
	}
}

func TestRenderRangeScope_Normal(t *testing.T) {
	scope := &RangeScope{
		LeftEndpointOID:  "abc123def456abc123def456abc123def456abcd",
		RightEndpointOID: "789xyz012789xyz012789xyz012789xyz0123456",
		CommitCount:      3,
		MergeCommitCount: 0,
		FilesChanged:     8,
		CrossesMerge:     false,
		Targetedness:     RangeTargetednessNormal,
		WarningCode:      WarningCodeNone,
		DiagnosticStatus: DiagnosticStatusAvailable,
	}

	output := RenderRangeScope(scope)

	if !strings.Contains(output, "## RANGE_SCOPE") {
		t.Error("missing RANGE_SCOPE header")
	}
	if !strings.Contains(output, "authority=explicit_range") {
		t.Error("missing authority line")
	}
	if !strings.Contains(output, "diagnostic_status=available") {
		t.Error("missing diagnostic_status")
	}
}

func TestRenderRangeScope_Unavailable(t *testing.T) {
	scope := &RangeScope{
		FilesChanged:     0,
		DiagnosticStatus: DiagnosticStatusUnavailable,
		RawRangeSpec:     "A...B",
		DiagnosticError:  "symmetric range (A...B) is not supported",
	}

	output := RenderRangeScope(scope)

	if !strings.Contains(output, "## RANGE_SCOPE") {
		t.Error("missing RANGE_SCOPE header for unavailable")
	}
	if !strings.Contains(output, "targetedness=UNKNOWN") {
		t.Error("missing UNKNOWN targetedness for unavailable")
	}
	if !strings.Contains(output, "diagnostic_status=unavailable") {
		t.Error("missing unavailable diagnostic_status")
	}

	output2 := RenderRangeScope(nil)
	if !strings.Contains(output2, "## RANGE_SCOPE") {
		t.Error("nil scope should still render RANGE_SCOPE section")
	}
}
