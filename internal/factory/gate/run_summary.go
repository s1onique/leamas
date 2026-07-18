package gate

import "time"

// WriteGateRunSummary records the observed result of one literal gate run.
func WriteGateRunSummary(
	outputPath string,
	startedAt time.Time,
	finishedAt time.Time,
	exitCode int,
) error {
	status := CheckStatusPass
	if exitCode != 0 {
		status = CheckStatusFail
	}

	duration := finishedAt.Sub(startedAt)
	if duration < 0 {
		duration = 0
	}
	summary := GateSummary{
		SchemaVersion: GateSummarySchemaVersion,
		GeneratedAt:   finishedAt.UTC().Format(time.RFC3339),
		Tool:          "leamas factory gate",
		OverallStatus: string(status),
		Checks: []Check{
			{
				Name:       "gate",
				Status:     status,
				DurationMs: duration.Milliseconds(),
				Evidence:   "leamas factory gate",
			},
		},
	}
	return writeGateSummaryArtifact(outputPath, summary)
}
