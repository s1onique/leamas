// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"strings"
	"testing"
)

func TestBuildManifest_DeterministicOrder(t *testing.T) {
	files := []ChangedFile{
		{Path: "zebra.go", Tracked: true, UnstagedPresent: true},
		{Path: "alpha.go", Tracked: true, UnstagedPresent: true},
		{Path: "beta.go", Tracked: true, UnstagedPresent: true},
	}

	manifest := BuildManifest(files)

	if len(manifest) != 3 {
		t.Fatalf("expected 3 files, got %d", len(manifest))
	}

	expected := []string{"alpha.go", "beta.go", "zebra.go"}
	for i, want := range expected {
		if manifest[i].Path != want {
			t.Errorf("manifest[%d] = %q, want %q", i, manifest[i].Path, want)
		}
	}
}

func TestBuildManifest_StatusDetection(t *testing.T) {
	tests := []struct {
		name       string
		file       ChangedFile
		wantStatus string
	}{
		{"untracked", ChangedFile{Path: "untracked.txt", Untracked: true}, StatusUntracked},
		{"staged only", ChangedFile{Path: "staged.txt", Tracked: true, StagedPresent: true, UnstagedPresent: false}, StatusAdded},
		{"unstaged only", ChangedFile{Path: "modified.txt", Tracked: true, StagedPresent: false, UnstagedPresent: true}, StatusModified},
		{"both staged and unstaged", ChangedFile{Path: "both.txt", Tracked: true, StagedPresent: true, UnstagedPresent: true}, StatusModified},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := BuildManifest([]ChangedFile{tt.file})
			if len(manifest) != 1 {
				t.Fatalf("expected 1 file, got %d", len(manifest))
			}
			if manifest[0].Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", manifest[0].Status, tt.wantStatus)
			}
		})
	}
}

func TestBuildRangeManifest_StatusNormalization(t *testing.T) {
	files := []RangeFile{
		{Path: "added.go", Status: "added"},
		{Path: "modified.go", Status: "modified"},
		{Path: "deleted.go", Status: "deleted"},
		{Path: "renamed.go", Status: "renamed", From: "old_name.go"},
	}

	manifest := BuildRangeManifest(files)

	expected := []struct {
		path    string
		status  string
		oldPath string
	}{
		{"added.go", StatusAdded, ""},
		{"deleted.go", StatusDeleted, ""},
		{"modified.go", StatusModified, ""},
		{"renamed.go", StatusRenamed, "old_name.go"},
	}

	for i, want := range expected {
		if manifest[i].Path != want.path {
			t.Errorf("manifest[%d].Path = %q, want %q", i, manifest[i].Path, want.path)
		}
		if manifest[i].Status != want.status {
			t.Errorf("manifest[%d].Status = %q, want %q", i, manifest[i].Status, want.status)
		}
		if manifest[i].OldPath != want.oldPath {
			t.Errorf("manifest[%d].OldPath = %q, want %q", i, manifest[i].OldPath, want.oldPath)
		}
	}
}

func TestRenderManifest_EmptyList(t *testing.T) {
	manifest := []ReviewChangedFile{}
	result := RenderManifest(manifest)

	if !strings.Contains(result, "(no changed files)") {
		t.Error("expected '(no changed files)' for empty manifest")
	}
}

func TestRenderManifest_WithFiles(t *testing.T) {
	manifest := []ReviewChangedFile{
		{Status: StatusAdded, Path: "new.go"},
		{Status: StatusModified, Path: "changed.go"},
		{Status: StatusRenamed, Path: "renamed.go", OldPath: "old_name.go"},
	}

	result := RenderManifest(manifest)

	if !strings.Contains(result, "## CHANGESET_MANIFEST") {
		t.Error("expected CHANGESET_MANIFEST header")
	}
	if !strings.Contains(result, "A  new.go") {
		t.Error("expected 'A  new.go' in output")
	}
	if !strings.Contains(result, "M  changed.go") {
		t.Error("expected 'M  changed.go' in output")
	}
	if !strings.Contains(result, "R  old_name.go -> renamed.go") {
		t.Error("expected 'R  old_name.go -> renamed.go' in output")
	}
}
