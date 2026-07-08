package tooling

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckPythonFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Python file
	pyFile := filepath.Join(tmpDir, "script.py")
	if err := os.WriteFile(pyFile, []byte("#!/usr/bin/env python3\nprint('hello')"), 0644); err != nil {
		t.Fatal(err)
	}

	findings := checkPythonFiles(tmpDir)
	if len(findings) == 0 {
		t.Error("expected findings for Python file")
	}
}

func TestCheckBashLOC(t *testing.T) {
	tmpDir := t.TempDir()

	// Create scripts directory
	scriptsDir := filepath.Join(tmpDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a long Bash script (>50 LOC)
	longScript := filepath.Join(scriptsDir, "long.sh")
	longContent := `#!/bin/bash
# This script is too long
set -euo pipefail
echo line1
echo line2
echo line3
echo line4
echo line5
echo line6
echo line7
echo line8
echo line9
echo line10
echo line11
echo line12
echo line13
echo line14
echo line15
echo line16
echo line17
echo line18
echo line19
echo line20
echo line21
echo line22
echo line23
echo line24
echo line25
echo line26
echo line27
echo line28
echo line29
echo line30
echo line31
echo line32
echo line33
echo line34
echo line35
echo line36
echo line37
echo line38
echo line39
echo line40
echo line41
echo line42
echo line43
echo line44
echo line45
echo line46
echo line47
echo line48
echo line49
echo line50
echo line51
echo line52
`
	if err := os.WriteFile(longScript, []byte(longContent), 0755); err != nil {
		t.Fatal(err)
	}

	findings := checkBashLOC(tmpDir)
	if len(findings) == 0 {
		t.Error("expected findings for long Bash script")
	}

	// Create a short Bash script (≤50 LOC)
	shortScript := filepath.Join(scriptsDir, "short.sh")
	shortContent := `#!/bin/bash
set -euo pipefail
echo hello
echo world
`
	if err := os.WriteFile(shortScript, []byte(shortContent), 0755); err != nil {
		t.Fatal(err)
	}

	findings = checkBashLOC(tmpDir)
	if len(findings) != 1 {
		t.Errorf("expected 1 finding for only long script, got %d", len(findings))
	}
}

func TestCheckNoBashVerifierLogic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create scripts directory
	scriptsDir := filepath.Join(tmpDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a wrapper script (should pass)
	wrapperScript := filepath.Join(scriptsDir, "verify_wrapper.sh")
	wrapperContent := `#!/bin/bash
set -euo pipefail
exec go run ./cmd/leamas factory verify test "$@"
`
	if err := os.WriteFile(wrapperScript, []byte(wrapperContent), 0755); err != nil {
		t.Fatal(err)
	}

	findings := checkNoBashVerifierLogic(tmpDir)
	if len(findings) != 0 {
		t.Errorf("expected no findings for wrapper script, got %d", len(findings))
	}

	// Create a verifier script with implementation (should fail)
	verifierScript := filepath.Join(scriptsDir, "verify_impl.sh")
	verifierContent := `#!/bin/bash
set -euo pipefail
echo "Verifying files..."
# grep-based verification logic
grep -r "forbidden" . || true
`
	if err := os.WriteFile(verifierScript, []byte(verifierContent), 0755); err != nil {
		t.Fatal(err)
	}

	findings = checkNoBashVerifierLogic(tmpDir)
	if len(findings) == 0 {
		t.Error("expected findings for Bash verifier with implementation")
	}
}

func TestCheckRepo(t *testing.T) {
	// Test with current repo
	findings := CheckRepo(".")
	// CheckRepo can return empty slice or nil - just verify no panic
	_ = findings
}
