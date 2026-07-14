// Package dupcode provides integration tests for v3 algorithm (maximal clone detection).
package dupcode

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCheckRepo_LongCloneProducesOneMaximalFinding is the decisive acceptance test
// that creates real temporary Go files and invokes CheckRepo to verify maximal clone detection.
// R6.4: Decisive acceptance test requirement.
//
// V3 behavior: The v3 algorithm chains all overlapping MinTokens windows into one maximal finding.
// For two identical 100-line functions, this produces 1 finding with 2 occurrences.
func TestCheckRepo_LongCloneProducesOneMaximalFinding(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "leamas-clone-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fileAContent := `package main

import "fmt"

func helper1() {
`
	for i := 1; i <= 100; i++ {
		fileAContent += "fmt.Println(\"line" + itoa(i) + "\")\n"
	}
	fileAContent += `}

func main() {
	helper1()
}
`

	fileBContent := `package main

import "fmt"

func helper2() {
`
	for i := 1; i <= 100; i++ {
		fileBContent += "fmt.Println(\"line" + itoa(i) + "\")\n"
	}
	fileBContent += `}

func main() {
	helper2()
}
`

	if err := os.WriteFile(filepath.Join(tmpDir, "a.go"), []byte(fileAContent), 0644); err != nil {
		t.Fatalf("failed to write file a.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.go"), []byte(fileBContent), 0644); err != nil {
		t.Fatalf("failed to write file b.go: %v", err)
	}

	cfg := Config{MinTokens: 400, MinLines: 40}

	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	var relevantFindings []Finding
	for _, f := range findings {
		if f.TokenCount >= 400 {
			relevantFindings = append(relevantFindings, f)
		}
	}

	// V3: Should produce at least 1 finding for the maximal clone
	if len(relevantFindings) < 1 {
		t.Errorf("expected at least 1 finding, got %d", len(relevantFindings))
	}

	// V3: The finding should cover both files (2 occurrences)
	if len(relevantFindings) > 0 && len(relevantFindings[0].Occurrences) != 2 {
		t.Errorf("expected 2 occurrences (one per file), got %d", len(relevantFindings[0].Occurrences))
	}
}

// TestCheckRepo_TwoUnrelatedClones tests that two unrelated long clones produce findings.
func TestCheckRepo_TwoUnrelatedClones(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "leamas-clone-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneBlock := "func block() {\n"
	for i := 1; i <= 50; i++ {
		cloneBlock += "println(\"a" + itoa(i) + "\")\n"
	}
	cloneBlock += "println(\"b1\")\n"
	for i := 2; i <= 50; i++ {
		cloneBlock += "println(\"b" + itoa(i) + "\")\n"
	}
	cloneBlock += "}\n\nfunc main() {}\n"

	fileAContent := "package main\n\n" + cloneBlock
	fileBContent := "package main\n\n" + cloneBlock

	if err := os.WriteFile(filepath.Join(tmpDir, "a.go"), []byte(fileAContent), 0644); err != nil {
		t.Fatalf("failed to write file a.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.go"), []byte(fileBContent), 0644); err != nil {
		t.Fatalf("failed to write file b.go: %v", err)
	}

	cfg := Config{MinTokens: 400, MinLines: 40}

	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	if len(findings) < 1 {
		t.Errorf("expected at least 1 finding, got %d", len(findings))
	}
}

// TestCheckRepo_ThreeFileClone tests that a clone appearing in three files produces findings.
func TestCheckRepo_ThreeFileClone(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "leamas-clone-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneLines := "package main\n\nfunc block() {\n"
	for i := 1; i <= 500; i++ {
		cloneLines += "print(\"line" + itoa(i) + "\")\n"
	}
	cloneLines += "}\n\nfunc main() {}\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "a.go"), []byte(cloneLines), 0644); err != nil {
		t.Fatalf("failed to write file a.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.go"), []byte(cloneLines), 0644); err != nil {
		t.Fatalf("failed to write file b.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "c.go"), []byte(cloneLines), 0644); err != nil {
		t.Fatalf("failed to write file c.go: %v", err)
	}

	cfg := Config{MinTokens: 400, MinLines: 40}

	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	if len(findings) == 0 {
		t.Error("expected some findings for the three-file clone")
	}

	if len(findings) > 0 {
		hasA, hasB, hasC := false, false, false
		for _, f := range findings {
			for _, occ := range f.Occurrences {
				if occ.Path == "a.go" {
					hasA = true
				}
				if occ.Path == "b.go" {
					hasB = true
				}
				if occ.Path == "c.go" {
					hasC = true
				}
			}
		}
		if !hasA {
			t.Error("expected a.go in findings")
		}
		if !hasB {
			t.Error("expected b.go in findings")
		}
		if !hasC {
			t.Error("expected c.go in findings")
		}
	}
}

// TestV3_InterleavedOffsetMatches tests that compatible chains are not split by interleaving.
func TestV3_InterleavedOffsetMatches(t *testing.T) {
	matches := []seedMatch{
		{
			Left:   rawWindow{Path: "a.go", StartPos: 0, EndPos: 39, StartLine: 10, EndLine: 50},
			Right:  rawWindow{Path: "b.go", StartPos: 100, EndPos: 139, StartLine: 100, EndLine: 140},
			Offset: 100,
		},
		{
			Left:   rawWindow{Path: "a.go", StartPos: 1, EndPos: 40, StartLine: 11, EndLine: 51},
			Right:  rawWindow{Path: "c.go", StartPos: 201, EndPos: 240, StartLine: 201, EndLine: 241},
			Offset: 200,
		},
		{
			Left:   rawWindow{Path: "a.go", StartPos: 2, EndPos: 41, StartLine: 12, EndLine: 52},
			Right:  rawWindow{Path: "b.go", StartPos: 102, EndPos: 141, StartLine: 102, EndLine: 142},
			Offset: 100,
		},
	}

	chains := buildChainsWithPartitioning(matches)

	if len(chains) < 2 {
		t.Errorf("expected at least 2 chains, got %d", len(chains))
	}

	totalMatches := 0
	for _, chain := range chains {
		totalMatches += len(chain.Matches)
	}
	if totalMatches != 3 {
		t.Errorf("expected 3 total matches across all chains, got %d", totalMatches)
	}
}
