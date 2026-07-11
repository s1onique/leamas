package doctrine

import (
	"os"
	"path/filepath"
	"testing"
)

// mustMkdirAll fails the test on mkdir errors.
func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

// mustWriteFile fails the test on write errors.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// mustRemove fails the test on remove errors (other than not-exist).
// Works for both files and directories.
func mustRemove(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("remove %s: %v", path, err)
	}
}

// mustSymlink fails the test on symlink errors.
func mustSymlink(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		t.Fatalf("symlink %s -> %s: %v", newname, oldname, err)
	}
}

// mustChmod fails the test on chmod errors.
func mustChmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

// mustReadFile fails the test on read errors.
func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// canonicalAgentInstruction is the projected ECF instruction text used by
// the test fixture for both the canonical file and the marked blocks in
// AGENTS.md / copilot-instructions.md.
const canonicalAgentInstruction = "## Executable Contract First\n\n" +
	"For every behavior-changing task:\n\n" +
	"1. Inspect the existing behavioral contract and relevant tests.\n" +
	"2. Before editing production code, identify the narrowest stable " +
	"boundary and design an orthogonal, declarative test matrix.\n" +
	"3. Implement the relevant tests and run them to establish RED for " +
	"the intended behavioral reason.\n" +
	"4. Only then implement the smallest coherent production change.\n" +
	"5. Establish focused GREEN, run affected subsystem tests, and run " +
	"the repository gate.\n" +
	"6. Refactor only while the executable contract remains green.\n\n" +
	"Test observable behavior rather than private implementation details.\n" +
	"Prefer table-driven tests where cases share execution logic.\n"

// createCanonicalECFStructure builds a minimal valid ECF repository.
func createCanonicalECFStructure(t *testing.T, tmpDir string) {
	t.Helper()
	docsDir := filepath.Join(tmpDir, "docs/doctrine")
	dotGithub := filepath.Join(tmpDir, ".github")
	templatesDir := filepath.Join(tmpDir, "docs/templates")
	mustMkdirAll(t, docsDir)
	mustMkdirAll(t, dotGithub)
	mustMkdirAll(t, templatesDir)

	canonDoctrine := "# Executable Contract First\n\n" +
		"## Purpose\n\nDefine behavior.\n\n" +
		"## Applicability\n\nApplies to all behavior-changing work.\n"
	mustWriteFile(t, filepath.Join(docsDir, "executable-contract-first.md"), canonDoctrine)

	mustWriteFile(t, filepath.Join(docsDir, "executable-contract-first-agent.md"), canonicalAgentInstruction)

	agentsContent := "# AGENTS.md\n\n## Project\n\nLeamas.\n\n" +
		ecfBeginMarker + "\n" + canonicalAgentInstruction + ecfEndMarker +
		"\n\n## Non-Negotiable Rules\n\n- No Python anywhere.\n"
	mustWriteFile(t, filepath.Join(tmpDir, "AGENTS.md"), agentsContent)

	copilotContent := "# GitHub Copilot Instructions\n\n" +
		ecfBeginMarker + "\n" + canonicalAgentInstruction + ecfEndMarker + "\n"
	mustWriteFile(t, filepath.Join(dotGithub, "copilot-instructions.md"), copilotContent)

	actContent := "# ACT Template\n\n## Title\n\n## Executable contract\n\n" +
		"### Stable boundary\n\n### Test matrix\n\n| A | B |\n\n" +
		"### RED evidence\n\n### GREEN evidence\n\n### Exceptions\n\nNone.\n"
	mustWriteFile(t, filepath.Join(templatesDir, "act.md"), actContent)
}
