package doctrine

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// Diagnostic codes for Executable Contract First verification.
// Unexported; gate integration uses only CheckExecutableContractFirst.
const (
	ecf001 = "ECF001" // canonical doctrine file missing or empty
	ecf002 = "ECF002" // canonical agent instruction missing or empty
	ecf003 = "ECF003" // projection target missing
	ecf004 = "ECF004" // marker missing (begin/end)
	ecf005 = "ECF005" // duplicate marker
	ecf006 = "ECF006" // marker order/nesting error
	ecf007 = "ECF007" // projected instruction differs from canonical
	ecf008 = "ECF008" // required ACT template section missing or template unreadable
	ecf009 = "ECF009" // input file exceeds size bound
	ecf010 = "ECF010" // path confinement violation (symlink escape)
	ecf011 = "ECF011" // file read failure
)

const maxECFInstructionFileSize = 256 * 1024

// Path constants - unexported.
var (
	ecfDoctrineFile         = "docs/doctrine/executable-contract-first.md"
	ecfAgentInstructionFile = "docs/doctrine/executable-contract-first-agent.md"
	ecfAgentsMDFile         = "AGENTS.md"
	ecfCopilotFile          = ".github/copilot-instructions.md"
	ecfActTemplateFile      = "docs/templates/act.md"
)

const (
	ecfBeginMarker = "<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:BEGIN -->"
	ecfEndMarker   = "<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:END -->"
)

var ecfRequiredACTSections = []string{
	"## Executable contract",
	"### Stable boundary",
	"### Test matrix",
	"### RED evidence",
	"### GREEN evidence",
	"### Exceptions",
}

// readStatus describes the outcome of opening and reading a configured file.
type readStatus int

const (
	readOK       readStatus = iota // content read successfully (may still be "")
	readMissing                    // file does not exist (or root inaccessible)
	readEmpty                      // file exists but has no usable content
	readEscaped                    // symlink component escapes root (ECF010)
	readTooLarge                   // file exceeds size bound
	readIO                         // I/O or permission error (ECF011)
)

// readResult couples the read status with the (possibly empty) content.
type readResult struct {
	status  readStatus
	content string
}

// CheckExecutableContractFirst verifies ECF doctrine compliance.
// Returns findings sorted by path, then code, then message.
//
// Each configured file is opened via a persistent os.Root so that any
// path component (final or intermediate) referencing a location outside
// the supplied root is rejected before the file is read.
func CheckExecutableContractFirst(root string) []checks.Finding {
	var findings []checks.Finding

	// Acquire a persistent root handle. If this fails (invalid root,
	// permission denied, I/O error) emit a single root-level ECF011 and
	// skip per-file checks; per-file "missing" findings would be
	// misleading when the failure is upstream of any file.
	r, rootErr := os.OpenRoot(root)
	if r == nil || rootErr != nil {
		findings = append(findings, checks.Finding{
			Path: root, Kind: ecf011,
			Message:  fmt.Sprintf("root not accessible: %v", rootErr),
			Severity: checks.SeverityError,
		})
		checks.SortFindings(findings)
		return findings
	}

	// 1. Canonical doctrine file - exists, readable, bounded, confined.
	doctrine := readECFConfined(r, root, ecfDoctrineFile, ecf001, &findings)
	if (doctrine.status == readOK || doctrine.status == readEmpty) &&
		strings.TrimSpace(doctrine.content) == "" {
		findings = append(findings, checks.Finding{
			Path: ecfDoctrineFile, Kind: ecf001,
			Message: "canonical doctrine file is empty", Severity: checks.SeverityError,
		})
	}

	// 2. Canonical agent instruction - exists, readable, bounded, confined,
	//    non-empty.
	canon := readECFConfined(r, root, ecfAgentInstructionFile, ecf002, &findings)
	if (canon.status == readOK || canon.status == readEmpty) &&
		strings.TrimSpace(canon.content) == "" {
		findings = append(findings, checks.Finding{
			Path: ecfAgentInstructionFile, Kind: ecf002,
			Message: "canonical agent instruction is empty", Severity: checks.SeverityError,
		})
		canon.content = ""
		canon.status = readEmpty
	}

	// 3. AGENTS.md projection target.
	agents := readECFConfined(r, root, ecfAgentsMDFile, ecf003, &findings)

	// 4. Copilot instructions projection target.
	copilot := readECFConfined(r, root, ecfCopilotFile, ecf003, &findings)

	// 5. ACT template.
	act := readECFConfined(r, root, ecfActTemplateFile, ecf008, &findings)

	// Marker checks run only when the projection target was read OK or was
	// empty. Missing / escape / bounds / I/O failures are represented by
	// their primary diagnostic only; cascading diagnostics are suppressed.
	if agents.status == readOK || agents.status == readEmpty {
		findings = checkECFMarkers(ecfAgentsMDFile, agents.content, canon.content, findings)
	}
	if copilot.status == readOK || copilot.status == readEmpty {
		findings = checkECFMarkers(ecfCopilotFile, copilot.content, canon.content, findings)
	}
	if act.status == readOK || act.status == readEmpty {
		findings = checkACTemplate(act.content, findings)
	}

	if r != nil {
		_ = r.Close()
	}

	checks.SortFindings(findings)
	return findings
}

// readECFConfined reads the configured relative file under a persistent
// os.Root. Symlink classification uses Lstat + Readlink so dangling
// escapes are detected without requiring the target to exist.
//
// The caller guarantees r != nil.
func readECFConfined(
	r *os.Root,
	root, rel, missingKind string,
	findings *[]checks.Finding,
) readResult {
	label := rel

	// Pre-classify: walk each path component under root. If any component
	// is a symlink whose target (absolute or relative-resolved) would land
	// outside the root, emit ECF010 immediately. This includes dangling
	// outside escapes because Lstat + Readlink do not require the target
	// to exist.
	if ecfConfinedByWalk(r, root, rel) {
		*findings = append(*findings, checks.Finding{
			Path: label, Kind: ecf010,
			Message:  "path confinement violation: symlink component escapes root",
			Severity: checks.SeverityError,
		})
		return readResult{status: readEscaped}
	}

	// Component walk passed: open via the persistent root.
	f, err := r.Open(rel)
	if err != nil {
		// If the path is a missing component, emit the configured missing
		// kind.
		if isNotExist(err) {
			*findings = append(*findings, checks.Finding{
				Path: label, Kind: missingKind,
				Message: "file missing", Severity: checks.SeverityError,
			})
			return readResult{status: readMissing}
		}
		// Otherwise: ordinary I/O / permission failure.
		*findings = append(*findings, checks.Finding{
			Path: label, Kind: ecf011,
			Message:  fmt.Sprintf("file read failure: %v", err),
			Severity: checks.SeverityError,
		})
		return readResult{status: readIO}
	}
	defer f.Close()

	// Bounded read: stop as soon as the bound is exceeded.
	lr := io.LimitReader(f, int64(maxECFInstructionFileSize)+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		*findings = append(*findings, checks.Finding{
			Path: label, Kind: ecf011,
			Message:  fmt.Sprintf("file read failure: %v", err),
			Severity: checks.SeverityError,
		})
		return readResult{status: readIO}
	}

	if len(data) > maxECFInstructionFileSize {
		*findings = append(*findings, checks.Finding{
			Path: label, Kind: ecf009,
			Message:  fmt.Sprintf("input file exceeds configured bound (%d bytes)", maxECFInstructionFileSize),
			Severity: checks.SeverityError,
		})
		return readResult{status: readTooLarge}
	}

	content := normalizeECFContent(string(data))
	if content == "" {
		return readResult{status: readEmpty, content: ""}
	}
	return readResult{status: readOK, content: content}
}

// isNotExist returns true for any error that wraps os.ErrNotExist or
// fs.ErrNotExist (including *os.PathError and *os.LinkError).
func isNotExist(err error) bool {
	return err != nil && (errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err))
}

func checkECFMarkers(rel, content, canonicalInstruction string, findings []checks.Finding) []checks.Finding {
	label := rel

	if content == "" {
		findings = append(findings, checks.Finding{
			Path: label, Kind: ecf004,
			Message: "projection target has no content", Severity: checks.SeverityError,
		})
		return findings
	}

	hasBegin := strings.Contains(content, ecfBeginMarker)
	hasEnd := strings.Contains(content, ecfEndMarker)
	beginIdx := strings.Index(content, ecfBeginMarker)
	endIdx := strings.Index(content, ecfEndMarker)

	if hasBegin && hasEnd && beginIdx > endIdx {
		findings = append(findings, checks.Finding{
			Path: label, Kind: ecf006,
			Message: "end marker before begin marker", Severity: checks.SeverityError,
		})
		return findings
	}

	if !hasBegin && !hasEnd {
		findings = append(findings, checks.Finding{
			Path: label, Kind: ecf004,
			Message: "begin and end markers missing", Severity: checks.SeverityError,
		})
		return findings
	}
	if !hasBegin {
		findings = append(findings, checks.Finding{
			Path: label, Kind: ecf004,
			Message: "begin marker missing", Severity: checks.SeverityError,
		})
		return findings
	}
	if !hasEnd {
		findings = append(findings, checks.Finding{
			Path: label, Kind: ecf004,
			Message: "end marker missing", Severity: checks.SeverityError,
		})
		return findings
	}

	markedContent := extractECFMarkedBlock(content)
	if markedContent == "" {
		findings = append(findings, checks.Finding{
			Path: label, Kind: ecf007,
			Message:  "projected instruction differs from canonical source",
			Severity: checks.SeverityError,
		})
		return findings
	}

	if countOccurrences(content, ecfBeginMarker) > 1 {
		findings = append(findings, checks.Finding{
			Path: label, Kind: ecf005,
			Message: "duplicate begin marker", Severity: checks.SeverityError,
		})
	}
	if countOccurrences(content, ecfEndMarker) > 1 {
		findings = append(findings, checks.Finding{
			Path: label, Kind: ecf005,
			Message: "duplicate end marker", Severity: checks.SeverityError,
		})
	}

	if hasNestedECFMarks(content) {
		findings = append(findings, checks.Finding{
			Path: label, Kind: ecf006,
			Message: "nested marked block detected", Severity: checks.SeverityError,
		})
	}

	if canonicalInstruction != "" && markedContent != canonicalInstruction {
		findings = append(findings, checks.Finding{
			Path: label, Kind: ecf007,
			Message:  "projected instruction differs from canonical source",
			Severity: checks.SeverityError,
		})
	}

	return findings
}

func checkACTemplate(content string, findings []checks.Finding) []checks.Finding {
	if content == "" {
		findings = append(findings, checks.Finding{
			Path: ecfActTemplateFile, Kind: ecf008,
			Message:  "ACT template file missing or unreadable",
			Severity: checks.SeverityError,
		})
		return findings
	}

	for _, heading := range ecfRequiredACTSections {
		found := false
		for _, line := range strings.Split(content, "\n") {
			if line == heading {
				found = true
				break
			}
		}
		if !found {
			headingText := strings.TrimPrefix(heading, "## ")
			headingText = strings.TrimPrefix(headingText, "### ")
			findings = append(findings, checks.Finding{
				Path: ecfActTemplateFile, Kind: ecf008,
				Message:  fmt.Sprintf("required ACT template section missing: %s", headingText),
				Severity: checks.SeverityError,
			})
		}
	}
	return findings
}
