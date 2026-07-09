// Package gate provides the quality gate commands.
package gate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// GateSummarySchemaVersion is the current schema version.
const GateSummarySchemaVersion = 1

// MaxEvidenceLength is the maximum length for evidence field values.
const MaxEvidenceLength = 240

// CheckStatus represents the status of a gate check.
type CheckStatus string

const (
	CheckStatusPass        CheckStatus = "pass"
	CheckStatusFail        CheckStatus = "fail"
	CheckStatusSkip        CheckStatus = "skip"
	CheckStatusUnavailable CheckStatus = "unavailable"
)

// ValidCheckStatuses is the set of valid check status values.
var ValidCheckStatuses = map[CheckStatus]bool{
	CheckStatusPass:        true,
	CheckStatusFail:        true,
	CheckStatusSkip:        true,
	CheckStatusUnavailable: true,
}

// Check represents a single gate check result.
type Check struct {
	Name       string      `json:"name"`
	Status     CheckStatus `json:"status"`
	DurationMs int64       `json:"duration_ms,omitempty"`
	Evidence   string      `json:"evidence,omitempty"`
}

// GateSummary represents the gate summary artifact.
type GateSummary struct {
	SchemaVersion int     `json:"schema_version"`
	GeneratedAt   string  `json:"generated_at"`
	Tool          string  `json:"tool"`
	OverallStatus string  `json:"overall_status"`
	Checks        []Check `json:"checks"`
}

// WriteGateSummary writes the gate summary to a JSON file.
func WriteGateSummary(root, outputPath string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	summary := buildGateSummary(root)
	summary.Tool = "leamas factory gate-summary"

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal gate summary: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write gate summary: %w", err)
	}

	return nil
}

// buildGateSummary runs configured checks and builds the summary.
func buildGateSummary(root string) GateSummary {
	checks := []Check{
		runGoTest(root),
		runGoVet(root),
		runMakeFactorize(root),
		runMakeGate(root),
	}

	// Sort checks by name for deterministic output
	sort.Slice(checks, func(i, j int) bool {
		return checks[i].Name < checks[j].Name
	})

	// Compute overall status
	overallStatus := CheckStatusPass
	for _, c := range checks {
		if c.Status == CheckStatusFail {
			overallStatus = CheckStatusFail
			break
		}
		if c.Status == CheckStatusUnavailable {
			overallStatus = CheckStatusUnavailable
		}
	}

	return GateSummary{
		SchemaVersion: GateSummarySchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		OverallStatus: string(overallStatus),
		Checks:        checks,
	}
}

func runGoTest(root string) Check {
	start := time.Now()
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = root
	err := cmd.Run()
	duration := time.Since(start).Milliseconds()

	status := CheckStatusPass
	if err != nil {
		status = CheckStatusFail
	}

	return Check{
		Name:       "go_test",
		Status:     status,
		DurationMs: duration,
		Evidence:   "go test ./...",
	}
}

func runGoVet(root string) Check {
	start := time.Now()
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = root
	err := cmd.Run()
	duration := time.Since(start).Milliseconds()

	status := CheckStatusPass
	if err != nil {
		status = CheckStatusFail
	}

	return Check{
		Name:       "go_vet",
		Status:     status,
		DurationMs: duration,
		Evidence:   "go vet ./...",
	}
}

func runMakeFactorize(root string) Check {
	start := time.Now()
	cmd := exec.Command("make", "factorize")
	cmd.Dir = root
	err := cmd.Run()
	duration := time.Since(start).Milliseconds()

	status := CheckStatusPass
	if err != nil {
		status = CheckStatusFail
	}

	return Check{
		Name:       "factorize",
		Status:     status,
		DurationMs: duration,
		Evidence:   "make factorize",
	}
}

func runMakeGate(root string) Check {
	start := time.Now()
	cmd := exec.Command("make", "gate")
	cmd.Dir = root
	err := cmd.Run()
	duration := time.Since(start).Milliseconds()

	status := CheckStatusPass
	if err != nil {
		status = CheckStatusFail
	}

	return Check{
		Name:       "gate",
		Status:     status,
		DurationMs: duration,
		Evidence:   "make gate",
	}
}

// ErrGateSummaryMissing indicates the gate summary file does not exist.
var ErrGateSummaryMissing = errors.New("gate summary file does not exist")

// ErrGateSummaryInvalid indicates the gate summary file exists but is invalid.
var ErrGateSummaryInvalid = errors.New("gate summary file is invalid")

// ReadGateSummary reads a gate summary from a JSON file.
func ReadGateSummary(path string) (*GateSummary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrGateSummaryMissing
		}
		return nil, fmt.Errorf("%w: %v", ErrGateSummaryInvalid, err)
	}

	var summary GateSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGateSummaryInvalid, err)
	}

	// Validate schema version
	if summary.SchemaVersion != GateSummarySchemaVersion {
		return nil, fmt.Errorf("%w: expected schema version %d, got %d", ErrGateSummaryInvalid, GateSummarySchemaVersion, summary.SchemaVersion)
	}

	// Sanitize and validate checks
	for i := range summary.Checks {
		sanitizeCheck(&summary.Checks[i])
	}

	// Sort checks by name for deterministic output
	sort.Slice(summary.Checks, func(i, j int) bool {
		return summary.Checks[i].Name < summary.Checks[j].Name
	})

	return &summary, nil
}

// sanitizeCheck sanitizes a check's fields for safe rendering.
func sanitizeCheck(c *Check) {
	c.Name = sanitizeString(c.Name)
	c.Evidence = sanitizeString(c.Evidence)

	// Validate and normalize status
	if !ValidCheckStatuses[c.Status] {
		c.Status = CheckStatusUnavailable
	}
}

// sanitizeString normalizes a string for digest rendering.
// It replaces newlines with spaces, trims whitespace, and truncates to max length.
func sanitizeString(s string) string {
	// Replace newlines and carriage returns with spaces
	re := regexp.MustCompile(`[\r\n]+`)
	s = re.ReplaceAllString(s, " ")
	// Replace multiple spaces with single space
	spaceRe := regexp.MustCompile(`\s+`)
	s = spaceRe.ReplaceAllString(s, " ")
	// Trim leading/trailing whitespace
	s = strings.TrimSpace(s)
	// Truncate if too long
	if len(s) > MaxEvidenceLength {
		s = s[:MaxEvidenceLength]
	}
	return s
}

// GateSummaryExists checks if the gate summary file exists.
func GateSummaryExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// RenderGateSummary renders a GateSummary as a string suitable for digest.
// If summary is nil and err is nil, renders as missing.
// If summary is nil and err is non-nil, renders as invalid.
// If summary is non-nil, renders the actual summary.
func RenderGateSummary(summary *GateSummary, err error) string {
	var sb strings.Builder
	sb.WriteString("## GATE_SUMMARY\n")

	sourcePath := ".factory/gate-summary.json"

	if summary == nil {
		if err != nil {
			// Invalid artifact
			sb.WriteString(fmt.Sprintf("source=%s\n", sourcePath))
			sb.WriteString("source_status=invalid\n")
			sb.WriteString("schema_version=0\n")
			sb.WriteString("generated_at=\n")
			sb.WriteString("overall_status=unavailable\n")
			sb.WriteString("checks_total=0\n")
			sb.WriteString("checks_passed=0\n")
			sb.WriteString("checks_failed=0\n")
			sb.WriteString("checks_skipped=0\n")
			sb.WriteString("checks_unavailable=0\n")
			// Bound the error message to avoid injection
			errMsg := sanitizeString(err.Error())
			sb.WriteString(fmt.Sprintf("diagnostics:\n  - failed to parse .factory/gate-summary.json: %s\n", errMsg))
			return sb.String()
		}
		// Missing artifact
		sb.WriteString(fmt.Sprintf("source=%s\n", sourcePath))
		sb.WriteString("source_status=missing\n")
		sb.WriteString("schema_version=0\n")
		sb.WriteString("generated_at=\n")
		sb.WriteString("overall_status=unavailable\n")
		sb.WriteString("checks_total=0\n")
		sb.WriteString("checks_passed=0\n")
		sb.WriteString("checks_failed=0\n")
		sb.WriteString("checks_skipped=0\n")
		sb.WriteString("checks_unavailable=0\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("source=%s\n", sourcePath))
	sb.WriteString("source_status=present\n")
	sb.WriteString(fmt.Sprintf("schema_version=%d\n", summary.SchemaVersion))
	sb.WriteString(fmt.Sprintf("generated_at=%s\n", sanitizeString(summary.GeneratedAt)))
	sb.WriteString(fmt.Sprintf("overall_status=%s\n", summary.OverallStatus))

	// Count checks
	total := len(summary.Checks)
	passed := 0
	failed := 0
	skipped := 0
	unavailable := 0

	for _, c := range summary.Checks {
		switch c.Status {
		case CheckStatusPass:
			passed++
		case CheckStatusFail:
			failed++
		case CheckStatusSkip:
			skipped++
		case CheckStatusUnavailable:
			unavailable++
		}
	}

	sb.WriteString(fmt.Sprintf("checks_total=%d\n", total))
	sb.WriteString(fmt.Sprintf("checks_passed=%d\n", passed))
	sb.WriteString(fmt.Sprintf("checks_failed=%d\n", failed))
	sb.WriteString(fmt.Sprintf("checks_skipped=%d\n", skipped))
	sb.WriteString(fmt.Sprintf("checks_unavailable=%d\n", unavailable))

	// Render check rows
	// Sort checks by name for deterministic output in rendering
	sortedChecks := make([]Check, len(summary.Checks))
	copy(sortedChecks, summary.Checks)
	sort.Slice(sortedChecks, func(i, j int) bool {
		return sortedChecks[i].Name < sortedChecks[j].Name
	})

	sb.WriteString("checks:\n")
	for _, c := range sortedChecks {
		evidence := c.Evidence
		if evidence == "" {
			evidence = c.Name
		}
		durationStr := ""
		if c.DurationMs > 0 {
			durationStr = fmt.Sprintf(" duration_ms=%d", c.DurationMs)
		}
		sb.WriteString(fmt.Sprintf("  - name=%s status=%s%s evidence=%s\n",
			sanitizeString(c.Name), c.Status, durationStr, sanitizeString(evidence)))
	}

	return sb.String()
}

// ParseGateSummaryStatus parses overall_status from a rendered summary.
func ParseGateSummaryStatus(rendered string) string {
	for _, line := range strings.Split(rendered, "\n") {
		if strings.HasPrefix(line, "overall_status=") {
			return strings.TrimPrefix(line, "overall_status=")
		}
	}
	return "unavailable"
}

// ParseGateSummarySourceStatus parses source_status from a rendered summary.
func ParseGateSummarySourceStatus(rendered string) string {
	for _, line := range strings.Split(rendered, "\n") {
		if strings.HasPrefix(line, "source_status=") {
			return strings.TrimPrefix(line, "source_status=")
		}
	}
	return "missing"
}
