package closure

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

const maxProjectedRows = 64

func Render(manifest Manifest, plan Plan) ([]byte, error) {
	if err := VerifyManifestAgainstPlan(manifest, plan); err != nil {
		return nil, fmt.Errorf("verify manifest before rendering: %w", err)
	}
	var report strings.Builder
	fmt.Fprintf(&report, "# %s Close Report\n\n", manifest.ActID)
	renderVerdictSection(&report, manifest)
	renderSubjectSection(&report, manifest)
	renderPlanSection(&report, manifest)
	renderChecksSection(&report, manifest.Checks)
	renderArtifactsSection(&report, manifest.Artifacts)
	renderExcludedSection(&report, manifest.ExcludedChecks)
	renderPolicySection(&report, manifest)
	renderRunnerSection(&report, manifest.Runner)
	renderLifecycleSection(&report, manifest)

	data := []byte(strings.TrimRight(report.String(), "\n") + "\n")
	if len(data) > MaxReportBytes {
		return nil, fmt.Errorf("rendered report exceeds %d-byte limit", MaxReportBytes)
	}
	if bytes.Count(data, []byte{'\n'}) > MaxReportLines {
		return nil, fmt.Errorf("rendered report exceeds %d-line limit", MaxReportLines)
	}
	if bytes.Contains(data, []byte("tag_object_oid")) || bytes.Contains(data, fullDigestMarker) {
		return nil, fmt.Errorf("rendered report contains prohibited evidence content")
	}
	return data, nil
}

func RenderFile(repositoryRoot, manifestPath, outputPath string) ([]byte, error) {
	manifest, _, err := VerifyManifestFile(repositoryRoot, manifestPath)
	if err != nil {
		return nil, err
	}
	plan, _, err := LoadPlan(joinRepositoryPath(repositoryRoot, manifest.Plan.Path))
	if err != nil {
		return nil, err
	}
	report, err := Render(manifest, plan)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outputPath, report, 0o644); err != nil {
		return nil, fmt.Errorf("write close report: %w", err)
	}
	return report, nil
}

func renderVerdictSection(report *strings.Builder, manifest Manifest) {
	report.WriteString("## Verdict\n\n")
	report.WriteString(strings.ToUpper(manifest.Verdict))
	report.WriteString("\n\n")
}

func renderSubjectSection(report *strings.Builder, manifest Manifest) {
	report.WriteString("## Subject\n\n")
	fmt.Fprintf(report, "- Commit: `%s`\n", manifest.Subject.CommitOID)
	fmt.Fprintf(report, "- Tree: `%s`\n\n", manifest.Subject.TreeOID)
}

func renderPlanSection(report *strings.Builder, manifest Manifest) {
	report.WriteString("## Plan\n\n")
	fmt.Fprintf(report, "- Path: `%s`\n", manifest.Plan.Path)
	fmt.Fprintf(report, "- SHA-256: `%s`\n\n", manifest.Plan.SHA256)
}

func renderChecksSection(report *strings.Builder, checks []CheckResult) {
	report.WriteString("## Checks\n\n")
	fmt.Fprintf(report, "Ordered results: %d.\n\n", len(checks))
	report.WriteString("| Check | Result | Duration | Exit |\n")
	report.WriteString("|---|---|---:|---:|\n")
	for _, check := range projectedChecks(checks) {
		exit := "—"
		if check.ExitCode != nil {
			exit = fmt.Sprint(*check.ExitCode)
		}
		fmt.Fprintf(report, "| %s | %s | %dms | %s |\n", check.CheckID, strings.ToUpper(check.Status), check.DurationMS, exit)
	}
	if len(checks) > maxProjectedRows {
		fmt.Fprintf(report, "\n%d additional ordered results remain authoritative in the manifest.\n", len(checks)-maxProjectedRows)
	}
	report.WriteString("\n")
}

func projectedChecks(checks []CheckResult) []CheckResult {
	if len(checks) <= maxProjectedRows {
		return checks
	}
	return checks[:maxProjectedRows]
}

func renderArtifactsSection(report *strings.Builder, artifacts []ArtifactResult) {
	report.WriteString("## Artifacts\n\n")
	if len(artifacts) == 0 {
		report.WriteString("None.\n\n")
		return
	}
	report.WriteString("| Artifact | Status | SHA-256 | Bytes |\n")
	report.WriteString("|---|---|---|---:|\n")
	limit := len(artifacts)
	if limit > maxProjectedRows {
		limit = maxProjectedRows
	}
	for _, artifact := range artifacts[:limit] {
		hash := artifact.SHA256
		if hash == "" {
			hash = "—"
		}
		fmt.Fprintf(report, "| %s | %s | %s | %d |\n", artifact.ArtifactID, strings.ToUpper(artifact.Status), hash, artifact.ByteCount)
	}
	if len(artifacts) > limit {
		fmt.Fprintf(report, "\n%d additional artifacts remain authoritative in the manifest.\n", len(artifacts)-limit)
	}
	report.WriteString("\n")
}

func renderExcludedSection(report *strings.Builder, excluded []ExcludedCheck) {
	report.WriteString("## Excluded checks\n\n")
	if len(excluded) == 0 {
		report.WriteString("None.\n\n")
		return
	}
	limit := len(excluded)
	if limit > maxProjectedRows {
		limit = maxProjectedRows
	}
	for _, check := range excluded[:limit] {
		fmt.Fprintf(report, "- `%s` — %s\n", check.CheckID, markdownText(check.Reason))
	}
	if len(excluded) > limit {
		fmt.Fprintf(report, "- %d additional exclusions remain authoritative in the manifest.\n", len(excluded)-limit)
	}
	report.WriteString("\n")
}

func markdownText(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}
