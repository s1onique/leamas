package doctrine

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckECF_AdditionalSubsectionsPass(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	actPath := filepath.Join(tmpDir, "docs/templates/act.md")
	content := mustReadFile(t, actPath)
	extra := "\n### Notes\n\nExtra content that is not a required heading.\n"
	mustWriteFile(t, actPath, content+extra)
	findings := CheckExecutableContractFirst(tmpDir)
	if len(findings) != 0 {
		t.Errorf("additional subsections should not cause failure, got %d findings", len(findings))
	}
}

func TestCheckECF_FileAtMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	canonPath := filepath.Join(tmpDir, "docs/doctrine/executable-contract-first-agent.md")
	content := strings.Repeat("a", maxECFInstructionFileSize)
	mustWriteFile(t, canonPath, content)
	findings := CheckExecutableContractFirst(tmpDir)
	if findByKind(findings, ecf009) {
		t.Error("file at exactly max size should be accepted")
	}
}

func TestCheckECF_FileOneByteOver(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	canonPath := filepath.Join(tmpDir, "docs/doctrine/executable-contract-first-agent.md")
	content := strings.Repeat("a", maxECFInstructionFileSize+1)
	mustWriteFile(t, canonPath, content)
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf009, ecfAgentInstructionFile) {
		t.Error("expected ECF009 for file one byte over max size")
	}
}

func TestCheckECF_UnrelatedTextPass(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	content := mustReadFile(t, agentsPath)
	// Insert additional non-marker text outside the marked block.
	newContent := content + "\n## Extra unrelated heading\n\nThis is fine.\n"
	mustWriteFile(t, agentsPath, newContent)
	findings := CheckExecutableContractFirst(tmpDir)
	if len(findings) != 0 {
		t.Errorf("unrelated text should not cause failure, got %d findings", len(findings))
	}
}

func TestCheckECF_EmptyMarkedBlock(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	content := mustReadFile(t, agentsPath)
	// Make the marked block empty.
	newContent := strings.Replace(content,
		ecfBeginMarker+"\n",
		ecfBeginMarker+"\n"+ecfEndMarker,
		1)
	mustWriteFile(t, agentsPath, newContent)
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf007, ecfAgentsMDFile) {
		t.Error("expected ECF007 for empty marked block")
	}
}

func TestCheckECF_ProseNotHeading(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	actPath := filepath.Join(tmpDir, "docs/templates/act.md")
	// Replace heading with prose that has the same text but extra words.
	mustWriteFile(t, actPath, "# ACT\n\n## Title\n\nThis is about the stable boundary heading.\n")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKind(findings, ecf008) {
		t.Error("expected ECF008 for prose that is not an exact heading")
	}
}

func TestCheckECF_SemanticallySimilarInstruction(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	canonPath := filepath.Join(tmpDir, "docs/doctrine/executable-contract-first-agent.md")
	mustWriteFile(t, canonPath, "## Executable Contract First\n\nA semantically different paragraph without the required six-step sequence.\n")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKind(findings, ecf007) {
		t.Error("expected ECF007 for semantically similar but textually different instruction")
	}
}

// --- R2.8: orthogonal heading tests ---

// removeHeading removes the literal heading line and the blank line that
// follows it from the file.
func removeHeading(t *testing.T, path, heading string) {
	t.Helper()
	content := mustReadFile(t, path)
	lines := strings.Split(content, "\n")
	var out []string
	skipNext := false
	for _, line := range lines {
		if skipNext {
			skipNext = false
			continue
		}
		if line == heading {
			// Drop this heading line and the blank line that follows.
			skipNext = true
			continue
		}
		out = append(out, line)
	}
	mustWriteFile(t, path, strings.Join(out, "\n"))
}

func TestCheckECF_StableBoundaryHeading(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	actPath := filepath.Join(tmpDir, "docs/templates/act.md")
	removeHeading(t, actPath, "### Stable boundary")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf008, ecfActTemplateFile) {
		t.Errorf("expected ECF008 for missing Stable boundary heading")
	}
	// Orthogonal check: confirm the message names the specific heading.
	found := false
	for _, f := range findings {
		if f.Kind == ecf008 && strings.Contains(f.Message, "Stable boundary") {
			found = true
			break
		}
	}
	if !found {
		t.Error("ECF008 message did not identify 'Stable boundary' specifically")
	}
}

func TestCheckECF_TestMatrixHeading(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	actPath := filepath.Join(tmpDir, "docs/templates/act.md")
	removeHeading(t, actPath, "### Test matrix")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf008, ecfActTemplateFile) {
		t.Error("expected ECF008 for missing Test matrix heading")
	}
	found := false
	for _, f := range findings {
		if f.Kind == ecf008 && strings.Contains(f.Message, "Test matrix") {
			found = true
			break
		}
	}
	if !found {
		t.Error("ECF008 message did not identify 'Test matrix' specifically")
	}
}

func TestCheckECF_REDEvidenceHeading(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	actPath := filepath.Join(tmpDir, "docs/templates/act.md")
	removeHeading(t, actPath, "### RED evidence")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf008, ecfActTemplateFile) {
		t.Error("expected ECF008 for missing RED evidence heading")
	}
	found := false
	for _, f := range findings {
		if f.Kind == ecf008 && strings.Contains(f.Message, "RED evidence") {
			found = true
			break
		}
	}
	if !found {
		t.Error("ECF008 message did not identify 'RED evidence' specifically")
	}
}

func TestCheckECF_GREENEvidenceHeading(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	actPath := filepath.Join(tmpDir, "docs/templates/act.md")
	removeHeading(t, actPath, "### GREEN evidence")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf008, ecfActTemplateFile) {
		t.Error("expected ECF008 for missing GREEN evidence heading")
	}
	found := false
	for _, f := range findings {
		if f.Kind == ecf008 && strings.Contains(f.Message, "GREEN evidence") {
			found = true
			break
		}
	}
	if !found {
		t.Error("ECF008 message did not identify 'GREEN evidence' specifically")
	}
}

func TestCheckECF_ExceptionsHeading(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)
	actPath := filepath.Join(tmpDir, "docs/templates/act.md")
	removeHeading(t, actPath, "### Exceptions")
	findings := CheckExecutableContractFirst(tmpDir)
	if !findByKindPath(findings, ecf008, ecfActTemplateFile) {
		t.Error("expected ECF008 for missing Exceptions heading")
	}
	found := false
	for _, f := range findings {
		if f.Kind == ecf008 && strings.Contains(f.Message, "Exceptions") {
			found = true
			break
		}
	}
	if !found {
		t.Error("ECF008 message did not identify 'Exceptions' specifically")
	}
}
