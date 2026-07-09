// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildReviewMap_Grouping(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{
		"main.go",
		"util.go",
		"main_test.go",
		"README.md",
		"config.yaml",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		os.WriteFile(path, []byte("content"), 0644)
	}

	manifest := make([]ReviewChangedFile, len(files))
	for i, f := range files {
		manifest[i] = ReviewChangedFile{Status: StatusModified, Path: f}
	}

	rm := BuildReviewMap(manifest, tmpDir)

	if len(rm.Production) != 2 {
		t.Errorf("Production has %d files, want 2", len(rm.Production))
	}
	if len(rm.Tests) != 1 {
		t.Errorf("Tests has %d files, want 1", len(rm.Tests))
	}
	if len(rm.Docs) != 1 {
		t.Errorf("Docs has %d files, want 1", len(rm.Docs))
	}
	if len(rm.Config) != 1 {
		t.Errorf("Config has %d files, want 1", len(rm.Config))
	}
}

func TestRenderReviewMap_EmptyGroupsRenderNone(t *testing.T) {
	rm := ReviewMap{
		Production: []string{},
		Tests:      []string{},
		Docs:       []string{},
		Config:     []string{},
		Generated:  []string{},
		Binary:     []string{},
	}

	result := RenderReviewMap(rm)

	expectedGroups := []string{"production", "tests", "docs", "config", "generated", "binary"}
	for _, group := range expectedGroups {
		if !strings.Contains(result, group+":") {
			t.Errorf("expected group %q in output", group)
		}
		if !strings.Contains(result, group+":\n  - none") {
			t.Errorf("expected '- none' for empty group %q", group)
		}
	}
}

func TestRenderReviewMap_FixedGroupOrder(t *testing.T) {
	rm := ReviewMap{
		Production: []string{"main.go"},
		Tests:      []string{"main_test.go"},
		Docs:       []string{"README.md"},
		Config:     []string{"config.yaml"},
		Generated:  []string{},
		Binary:     []string{},
	}

	result := RenderReviewMap(rm)

	groupOrder := []string{"production", "tests", "docs", "config", "generated", "binary"}
	for i, group := range groupOrder {
		idx := strings.Index(result, group+":")
		if i > 0 {
			prevIdx := strings.Index(result, groupOrder[i-1]+":")
			if idx <= prevIdx {
				t.Errorf("group %q should come after %q", group, groupOrder[i-1])
			}
		}
	}
}
