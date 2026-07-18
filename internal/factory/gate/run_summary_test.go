package gate

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteGateRunSummary(t *testing.T) {
	startedAt := time.Date(2026, 7, 18, 1, 39, 28, 0, time.UTC)
	finishedAt := startedAt.Add(1500 * time.Millisecond)
	tests := []struct {
		name            string
		exitCode        int
		wantOverall     string
		wantCheckStatus CheckStatus
		wantFailed      string
	}{
		{
			name:            "passing gate",
			exitCode:        0,
			wantOverall:     "pass",
			wantCheckStatus: CheckStatusPass,
			wantFailed:      "checks_failed=0",
		},
		{
			name:            "failing gate",
			exitCode:        1,
			wantOverall:     "fail",
			wantCheckStatus: CheckStatusFail,
			wantFailed:      "checks_failed=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".factory", "gate-summary.json")
			if err := WriteGateRunSummary(path, startedAt, finishedAt, tt.exitCode); err != nil {
				t.Fatalf("WriteGateRunSummary failed: %v", err)
			}

			summary, err := ReadGateSummary(path)
			if err != nil {
				t.Fatalf("ReadGateSummary failed: %v", err)
			}
			if summary.GeneratedAt != finishedAt.Format(time.RFC3339) {
				t.Errorf("generated_at=%q, want %q", summary.GeneratedAt, finishedAt.Format(time.RFC3339))
			}
			if summary.Tool != "leamas factory gate" {
				t.Errorf("tool=%q, want %q", summary.Tool, "leamas factory gate")
			}
			if summary.OverallStatus != tt.wantOverall {
				t.Errorf("overall_status=%q, want %q", summary.OverallStatus, tt.wantOverall)
			}
			if len(summary.Checks) != 1 {
				t.Fatalf("checks=%d, want 1", len(summary.Checks))
			}
			check := summary.Checks[0]
			if check.Name != "gate" || check.Status != tt.wantCheckStatus {
				t.Errorf("gate check=(%q,%q), want (%q,%q)",
					check.Name, check.Status, "gate", tt.wantCheckStatus)
			}
			if check.DurationMs != 1500 {
				t.Errorf("duration_ms=%d, want 1500", check.DurationMs)
			}

			rendered := RenderGateSummary(summary, nil)
			for _, want := range []string{
				"source_status=present",
				"overall_status=" + tt.wantOverall,
				tt.wantFailed,
				"checks_unavailable=0",
			} {
				if !strings.Contains(rendered, want) {
					t.Errorf("rendered summary missing %q:\n%s", want, rendered)
				}
			}
		})
	}
}
