package doctrine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// --- R3.1: Final-component symlink escape tests ---

func TestCheckECF_PathEscape_AGENTS(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	mustRemove(t, filepath.Join(tmpDir, "AGENTS.md"))
	mustSymlink(t, "/tmp", filepath.Join(tmpDir, "AGENTS.md"))
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfAgentsMDFile) {
		t.Errorf("expected ECF010 for AGENTS.md symlink escape, got %d findings:", len(findings))
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

func TestCheckECF_PathEscape_Copilot(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	copilotPath := filepath.Join(tmpDir, ".github", "copilot-instructions.md")
	mustRemove(t, copilotPath)
	mustSymlink(t, "/tmp", copilotPath)
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfCopilotFile) {
		t.Errorf("expected ECF010 for copilot-instructions.md symlink escape")
	}
}

func TestCheckECF_PathEscape_AgentInstruction(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	agentPath := filepath.Join(tmpDir, "docs/doctrine", "executable-contract-first-agent.md")
	mustRemove(t, agentPath)
	mustSymlink(t, "/tmp", agentPath)
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfAgentInstructionFile) {
		t.Errorf("expected ECF010 for agent-instruction symlink escape")
	}
}

func TestCheckECF_PathEscape_Doctrine(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	doctrinePath := filepath.Join(tmpDir, "docs/doctrine", "executable-contract-first.md")
	mustRemove(t, doctrinePath)
	mustSymlink(t, "/tmp", doctrinePath)
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfDoctrineFile) {
		t.Errorf("expected ECF010 for doctrine symlink escape")
	}
}

func TestCheckECF_PathEscape_ACTTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	actPath := filepath.Join(tmpDir, "docs/templates", "act.md")
	mustRemove(t, actPath)
	mustSymlink(t, "/tmp", actPath)
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfActTemplateFile) {
		t.Errorf("expected ECF010 for ACT template symlink escape")
	}
}

// --- R3.2: Parent-directory symlink escape tests ---
// A regular file reached through a symlinked parent directory still
// escapes the root. The verifier must detect this even though the final
// component is a regular file.

func TestCheckECF_PathEscape_ParentDir_GitHub(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)

	// Replace .github with a symlink to an outside directory.
	githubDir := filepath.Join(tmpDir, ".github")
	mustRemove(t, githubDir)
	outsideDir := t.TempDir()
	mustMkdirAll(t, filepath.Join(outsideDir, ".github"))
	mustWriteFile(t, filepath.Join(outsideDir, ".github", "copilot-instructions.md"),
		canonicalAgentInstruction)
	mustSymlink(t, outsideDir+"/.github", githubDir)

	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfCopilotFile) {
		t.Errorf("expected ECF010 for .github parent-dir symlink escape, got %d findings:", len(findings))
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

func TestCheckECF_PathEscape_ParentDir_Docs(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)

	// Replace docs/doctrine with a symlink to an outside directory.
	doctrineDir := filepath.Join(tmpDir, "docs", "doctrine")
	mustRemove(t, doctrineDir)
	outsideDir := t.TempDir()
	mustMkdirAll(t, filepath.Join(outsideDir, "doctrine"))
	mustWriteFile(t, filepath.Join(outsideDir, "doctrine", "executable-contract-first.md"),
		"# Outside doctrine\n")
	mustSymlink(t, filepath.Join(outsideDir, "doctrine"), doctrineDir)

	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfDoctrineFile) {
		t.Errorf("expected ECF010 for docs/doctrine parent-dir symlink escape, got %d findings:", len(findings))
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

func TestCheckECF_PathEscape_ParentDir_Templates(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)

	// Replace docs/templates with a symlink to an outside directory.
	templatesDir := filepath.Join(tmpDir, "docs", "templates")
	mustRemove(t, templatesDir)
	outsideDir := t.TempDir()
	mustMkdirAll(t, filepath.Join(outsideDir, "templates"))
	mustWriteFile(t, filepath.Join(outsideDir, "templates", "act.md"),
		"# Outside ACT\n")
	mustSymlink(t, filepath.Join(outsideDir, "templates"), templatesDir)

	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfActTemplateFile) {
		t.Errorf("expected ECF010 for docs/templates parent-dir symlink escape, got %d findings:", len(findings))
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

// --- R3.3: In-root symlink variants ---

// A symlink whose target is a regular file inside the root must succeed.
// Note: os.OpenInRoot requires relative symlink targets; absolute
// targets trigger "path escapes from parent" on macOS/Linux.
func TestCheckECF_PathEscape_InRootReadableSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)

	// Create an in-root readable target file.
	targetPath := filepath.Join(tmpDir, "real_agents.md")
	mustWriteFile(t, targetPath, canonicalAgentInstruction)

	// Replace AGENTS.md with a relative symlink to that in-root target.
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	mustRemove(t, agentsPath)
	mustSymlink(t, "real_agents.md", agentsPath)

	findings := CheckExecutableContractFirst(tmpDir)
	if findByKindPath(findings, ecf010, ecfAgentsMDFile) {
		t.Errorf("unexpected ECF010 for in-root readable symlink")
	}
	if findByKindPath(findings, ecf011, ecfAgentsMDFile) {
		t.Errorf("unexpected ECF011 for in-root readable symlink")
	}
}

// A symlink whose target is an in-root but unreadable file: ECF011.
// Uses a relative symlink target because os.OpenInRoot only follows
// relative symlinks.
func TestCheckECF_PathEscape_InRootUnreadableSymlink(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; chmod does not restrict file access")
	}
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)

	targetPath := filepath.Join(tmpDir, "real_agents.md")
	mustWriteFile(t, targetPath, canonicalAgentInstruction)
	mustChmod(t, targetPath, 0000)
	t.Cleanup(func() { _ = os.Chmod(targetPath, 0644) })

	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	mustRemove(t, agentsPath)
	mustSymlink(t, "real_agents.md", agentsPath)

	findings := CheckExecutableContractFirst(tmpDir)
	if findByKindPath(findings, ecf010, ecfAgentsMDFile) {
		t.Errorf("in-root unreadable symlink must not be classified as ECF010")
	}
	if !findByKindPath(findings, ecf011, ecfAgentsMDFile) {
		t.Errorf("expected ECF011 for in-root unreadable symlink, got %d findings:", len(findings))
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

// A dangling in-root symlink (relative target): missing/ECF011, not
// ECF010. Absolute targets are always confinement violations per the
// os.Root contract, so this test uses a relative target.
func TestCheckECF_PathEscape_DanglingInRootSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)

	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	mustRemove(t, agentsPath)
	mustSymlink(t, "missing_target.md", agentsPath)

	findings := CheckExecutableContractFirst(tmpDir)
	if findByKindPath(findings, ecf010, ecfAgentsMDFile) {
		t.Errorf("dangling in-root relative symlink must not be classified as ECF010")
	}
	// Either ECF003 (file missing) or ECF011 (read failure) is acceptable.
	if !(findByKindPath(findings, ecf011, ecfAgentsMDFile) ||
		findByKindPath(findings, ecf003, ecfAgentsMDFile)) {
		t.Errorf("expected ECF011 or ECF003 for dangling in-root symlink, got %d findings:", len(findings))
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

// --- R4 dangling-escape regression tests ---

// AGENTS.md -> /outside/missing.md (absolute dangling outside symlink).
// ECF010 must be emitted even though the target does not exist.
func TestCheckECF_PathEscape_DanglingOutside_FinalSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	mustRemove(t, filepath.Join(tmpDir, "AGENTS.md"))
	mustSymlink(t, "/outside/does-not-exist.md",
		filepath.Join(tmpDir, "AGENTS.md"))

	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfAgentsMDFile) {
		t.Errorf("expected ECF010 for dangling outside final symlink, got %d findings:", len(findings))
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
	// Must be exactly one ECF010, no cascading ECF011.
	if got := countByKindPath(findings, ecf010, ecfAgentsMDFile); got != 1 {
		t.Errorf("expected exactly one ECF010 on AGENTS.md, got %d", got)
	}
}

// .github -> /outside/missing-directory (parent dangling outside).
func TestCheckECF_PathEscape_DanglingOutside_ParentSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	githubDir := filepath.Join(tmpDir, ".github")
	mustRemove(t, githubDir)
	mustSymlink(t, "/outside/does-not-exist-dir", githubDir)

	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfCopilotFile) {
		t.Errorf("expected ECF010 for dangling outside parent symlink, got %d findings:", len(findings))
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

// docs/doctrine -> /outside/missing (parent dangling outside, docs path).
func TestCheckECF_PathEscape_DanglingOutside_ParentSymlink_Docs(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	doctrineDir := filepath.Join(tmpDir, "docs", "doctrine")
	mustRemove(t, doctrineDir)
	mustSymlink(t, "/outside/does-not-exist", doctrineDir)

	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfDoctrineFile) {
		t.Errorf("expected ECF010 for dangling outside docs parent symlink")
	}
}

// relative symlink -> ../../outside/missing.md (relative dangling outside).
func TestCheckECF_PathEscape_DanglingOutside_RelativeSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	mustRemove(t, filepath.Join(tmpDir, "AGENTS.md"))
	mustSymlink(t, "../../outside/missing.md",
		filepath.Join(tmpDir, "AGENTS.md"))

	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf010, ecfAgentsMDFile) {
		t.Errorf("expected ECF010 for relative dangling outside symlink, got %d findings:", len(findings))
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

// Unreadable parent directory: produces ECF011, not ECF010.
func TestCheckECF_PathEscape_UnreadableParentDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; chmod does not restrict file access")
	}
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	githubDir := filepath.Join(tmpDir, ".github")
	mustChmod(t, githubDir, 0000)
	t.Cleanup(func() { _ = os.Chmod(githubDir, 0755) })

	findings := CheckExecutableContractFirst(tmpDir)
	if findByKindPath(findings, ecf010, ecfCopilotFile) {
		t.Errorf("unreadable in-root parent must not be ECF010")
	}
	if !findByKindPath(findings, ecf011, ecfCopilotFile) {
		t.Errorf("expected ECF011 for unreadable in-root parent, got %d findings:", len(findings))
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

// countByKindPath returns the number of findings matching both kind and path.
func countByKindPath(findings []checks.Finding, kind, path string) int {
	n := 0
	for _, f := range findings {
		if f.Kind == kind && f.Path == path {
			n++
		}
	}
	return n
}
