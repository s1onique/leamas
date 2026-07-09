// Package gate provides the quality gate commands.
package gate

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// GateSummarySchemaVersion is the current schema version.
const GateSummarySchemaVersion = 1

// CheckStatus represents the status of a gate check.
type CheckStatus string

const (
	CheckStatusPass        CheckStatus = "pass"
	CheckStatusFail        CheckStatus = "fail"
	CheckStatusSkip        CheckStatus = "skip"
	CheckStatusUnavailable CheckStatus = "unavailable"
)

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

// ReadGateSummary reads a gate summary from a JSON file.
func ReadGateSummary(path string) (*GateSummary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var summary GateSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, err
	}

	return &summary, nil
}

// GateSummaryExists checks if the gate summary file exists.
func GateSummaryExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// RenderGateSummary renders a GateSummary as a string suitable for digest.
func RenderGateSummary(summary *GateSummary) string {
	var sb strings.Builder
	sb.WriteString("## GATE_SUMMARY\n")

	sourcePath := ".factory/gate-summary.json"
	if summary == nil {
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
	sb.WriteString(fmt.Sprintf("generated_at=%s\n", summary.GeneratedAt))
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
	sb.WriteString("checks:\n")
	for _, c := range summary.Checks {
		evidence := c.Evidence
		if evidence == "" {
			evidence = c.Name
		}
		durationStr := ""
		if c.DurationMs > 0 {
			durationStr = fmt.Sprintf(" duration_ms=%d", c.DurationMs)
		}
		sb.WriteString(fmt.Sprintf("  - name=%s status=%s%s evidence=%s\n",
			c.Name, c.Status, durationStr, evidence))
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
