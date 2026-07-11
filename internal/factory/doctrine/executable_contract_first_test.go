package doctrine

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// findByKind returns true if a finding with the given kind is present.
func findByKind(findings []checks.Finding, kind string) bool {
	for _, f := range findings {
		if f.Kind == kind {
			return true
		}
	}
	return false
}

// findByKindPath returns true if a finding with the given kind AND path is present.
func findByKindPath(findings []checks.Finding, kind, path string) bool {
	for _, f := range findings {
		if f.Kind == kind && f.Path == path {
			return true
		}
	}
	return false
}

// --- Diagnostic coverage tests ---

func TestCheckECF_CanonicalDoctrinMissing(t *testing.T) {
	tmpDir := t.TempDir()
	// The doctrine file is required to live at
	// docs/doctrine/executable-contract-first.md under root. Without any
	// files written, the root looks empty and we should report ECF001.
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKind(findings, ecf001) {
		t.Error("expected ECF001 for missing canonical doctrine file")
	}
}

func TestCheckECF_CanonicalAgentInstructionMissing(t *testing.T) {
	tmpDir := t.TempDir()
	mustMkdirAll(t, filepath.Join(tmpDir, "docs/doctrine"))
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKind(findings, ecf002) {
		t.Error("expected ECF002 for missing canonical agent instruction")
	}
}

func TestCheckECF_AGENTSMissing(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	mustRemove(t, filepath.Join(tmpDir, "AGENTS.md"))
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf003, ecfAgentsMDFile) {
		t.Error("expected ECF003 for missing AGENTS.md")
	}
}

func TestCheckECF_CopilotInstructionsMissing(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	mustRemove(t, filepath.Join(tmpDir, ".github", "copilot-instructions.md"))
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf003, ecfCopilotFile) {
		t.Error("expected ECF003 for missing copilot-instructions.md")
	}
}

func TestCheckECF_MissingBeginMarker(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	content := mustReadFile(t, agentsPath)
	newContent := strings.ReplaceAll(content, ecfBeginMarker+"\n", "")
	mustWriteFile(t, agentsPath, newContent)
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf004, ecfAgentsMDFile) {
		t.Error("expected ECF004 for missing begin marker")
	}
}

func TestCheckECF_MissingEndMarker(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	content := mustReadFile(t, agentsPath)
	newContent := strings.ReplaceAll(content, "\n"+ecfEndMarker, "")
	mustWriteFile(t, agentsPath, newContent)
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf004, ecfAgentsMDFile) {
		t.Error("expected ECF004 for missing end marker")
	}
}

func TestCheckECF_DuplicateBeginMarker(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	content := mustReadFile(t, agentsPath)
	newContent := content + "\n" + ecfBeginMarker + "\n"
	mustWriteFile(t, agentsPath, newContent)
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf005, ecfAgentsMDFile) {
		t.Error("expected ECF005 for duplicate begin marker")
	}
}

func TestCheckECF_DuplicateEndMarker(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	content := mustReadFile(t, agentsPath)
	newContent := content + "\n" + ecfEndMarker + "\n"
	mustWriteFile(t, agentsPath, newContent)
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf005, ecfAgentsMDFile) {
		t.Error("expected ECF005 for duplicate end marker")
	}
}

func TestCheckECF_EndBeforeBegin(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	mustWriteFile(t, agentsPath, "# AGENTS.md\n\n"+ecfEndMarker+"\nSome content\n"+ecfBeginMarker+"\nMore content\n")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf006, ecfAgentsMDFile) {
		t.Error("expected ECF006 for end marker before begin")
	}
}

func TestCheckECF_InstructionDrift(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	content := mustReadFile(t, agentsPath)
	newContent := strings.ReplaceAll(content, "Before editing", "After editing")
	mustWriteFile(t, agentsPath, newContent)
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf007, ecfAgentsMDFile) {
		t.Error("expected ECF007 for instruction drift")
	}
}

func TestCheckECF_MissingACTTemplateHeading(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	actTemplate := filepath.Join(tmpDir, "docs/templates/act.md")
	mustWriteFile(t, actTemplate, "# ACT Template\n\n## Title\n\nJust title\n")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKind(findings, ecf008) {
		t.Error("expected ECF008 for missing ACT template heading")
	}
}

func TestCheckECF_OversizedFile(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	canonPath := filepath.Join(tmpDir, "docs/doctrine/executable-contract-first-agent.md")
	largeContent := make([]byte, maxECFInstructionFileSize+1)
	for i := range largeContent {
		largeContent[i] = 'x'
	}
	mustWriteFile(t, canonPath, string(largeContent))
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf009, ecfAgentInstructionFile) {
		t.Error("expected ECF009 for oversized agent instruction file")
	}
}

func TestCheckECF_UnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; chmod does not restrict file access")
	}
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	unreadablePath := filepath.Join(tmpDir, "docs/doctrine", "executable-contract-first-agent.md")
	mustChmod(t, unreadablePath, 0000)
	findings := CheckExecutableContractFirst(tmpDir)
	mustChmod(t, unreadablePath, 0644)
	if !findByKindPath(findings, ecf011, ecfAgentInstructionFile) {
		t.Errorf("expected ECF011 for unreadable file, got: %+v", findings)
	}
}

func TestCheckECF_ValidRepository(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	findings := CheckExecutableContractFirst(tmpDir)
	if len(findings) != 0 {
		for _, f := range findings {
			t.Logf("unexpected finding: %s: %s: %s", f.Path, f.Kind, f.Message)
		}
		t.Errorf("expected no findings for valid repository, got %d", len(findings))
	}
}

func TestCheckECF_CRLFNormalization(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	content := mustReadFile(t, agentsPath)
	newContent := insertCRLFInMarkedBlock(content)
	mustWriteFile(t, agentsPath, newContent)
	findings := CheckExecutableContractFirst(tmpDir)
	if len(findings) != 0 {
		for _, f := range findings {
			t.Logf("CRLF finding: %s: %s: %s", f.Path, f.Kind, f.Message)
		}
		t.Errorf("CRLF should be normalized to LF, got %d findings", len(findings))
	}
}

// --- R2.3: empty files must be rejected distinctly ---

func TestCheckECF_EmptyCanonicalInstruction(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	mustWriteFile(t, filepath.Join(tmpDir, "docs/doctrine/executable-contract-first-agent.md"), "")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf002, ecfAgentInstructionFile) {
		t.Errorf("expected ECF002 for empty canonical agent instruction")
	}
}

func TestCheckECF_EmptyAGENTS(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	mustWriteFile(t, filepath.Join(tmpDir, "AGENTS.md"), "")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf004, ecfAgentsMDFile) {
		t.Errorf("expected ECF004 for empty AGENTS.md, got %d findings", len(findings))
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

func TestCheckECF_EmptyCopilot(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	mustWriteFile(t, filepath.Join(tmpDir, ".github", "copilot-instructions.md"), "")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf004, ecfCopilotFile) {
		t.Errorf("expected ECF004 for empty copilot-instructions.md")
	}
}

func TestCheckECF_EmptyACT(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	mustWriteFile(t, filepath.Join(tmpDir, "docs/templates", "act.md"), "")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKind(findings, ecf008) {
		t.Errorf("expected ECF008 for empty ACT template")
	}
}

// --- R2.6: deterministic ordering across path / code / message ---

func TestCheckECF_DeterministicOrdering_MultiViolations(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	mustRemove(t, filepath.Join(tmpDir, "AGENTS.md"))
	mustRemove(t, filepath.Join(tmpDir, ".github", "copilot-instructions.md"))
	findings := CheckExecutableContractFirst(tmpDir)
	if len(findings) < 2 {
		t.Fatalf("need multiple findings to test ordering, got %d", len(findings))
	}
	sortedCopy := append([]checks.Finding(nil), findings...)
	sort.SliceStable(sortedCopy, func(i, j int) bool {
		if sortedCopy[i].Path != sortedCopy[j].Path {
			return sortedCopy[i].Path < sortedCopy[j].Path
		}
		if sortedCopy[i].Kind != sortedCopy[j].Kind {
			return sortedCopy[i].Kind < sortedCopy[j].Kind
		}
		return sortedCopy[i].Message < sortedCopy[j].Message
	})
	for i := range findings {
		if findings[i] != sortedCopy[i] {
			t.Errorf("finding %d not in deterministic order:\n got: %+v\nwant: %+v", i, findings[i], sortedCopy[i])
		}
	}
}

func TestCheckECF_DeterministicOrdering_DoubleInvocation(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	mustRemove(t, filepath.Join(tmpDir, "AGENTS.md"))
	findings1 := CheckExecutableContractFirst(tmpDir)
	findings2 := CheckExecutableContractFirst(tmpDir)
	if len(findings1) != len(findings2) {
		t.Fatalf("lengths differ: %d vs %d", len(findings1), len(findings2))
	}
	for i := range findings1 {
		if findings1[i] != findings2[i] {
			t.Errorf("invocations differ at index %d:\n a=%+v\n b=%+v", i, findings1[i], findings2[i])
		}
	}
}
