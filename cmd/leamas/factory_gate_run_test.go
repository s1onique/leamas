package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/factory/gate"
)

func TestRunFactoryGateWritesObservedSummary(t *testing.T) {
	startedAt := time.Date(2026, 7, 18, 1, 39, 28, 0, time.UTC)
	finishedAt := startedAt.Add(2 * time.Second)
	tests := []struct {
		name        string
		gateCode    int
		wantOverall string
	}{
		{name: "pass", gateCode: 0, wantOverall: "pass"},
		{name: "fail", gateCode: 1, wantOverall: "fail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			times := []time.Time{startedAt, finishedAt}
			now := func() time.Time {
				got := times[0]
				times = times[1:]
				return got
			}
			var gotRoot string
			run := func(root string) int {
				gotRoot = root
				return tt.gateCode
			}
			path := filepath.Join(t.TempDir(), ".factory", "gate-summary.json")

			gotCode := runFactoryGate("repo-root", path, &bytes.Buffer{}, run, now)
			if gotCode != tt.gateCode {
				t.Errorf("exit code=%d, want %d", gotCode, tt.gateCode)
			}
			if gotRoot != "repo-root" {
				t.Errorf("gate root=%q, want repo-root", gotRoot)
			}
			summary, err := gate.ReadGateSummary(path)
			if err != nil {
				t.Fatalf("ReadGateSummary failed: %v", err)
			}
			if summary.OverallStatus != tt.wantOverall {
				t.Errorf("overall_status=%q, want %q", summary.OverallStatus, tt.wantOverall)
			}
			if summary.GeneratedAt != finishedAt.Format(time.RFC3339) {
				t.Errorf("generated_at=%q, want %q", summary.GeneratedAt, finishedAt.Format(time.RFC3339))
			}
		})
	}
}

func TestRunFactoryGateFailsWhenSummaryCannotBeWritten(t *testing.T) {
	blockedPath := t.TempDir()
	var stderr bytes.Buffer
	now := func() time.Time { return time.Unix(0, 0).UTC() }

	code := runFactoryGate("repo-root", blockedPath, &stderr, func(string) int { return 0 }, now)
	if code == 0 {
		t.Fatal("runFactoryGate returned success after gate-summary write failure")
	}
	if !strings.Contains(stderr.String(), "write gate summary") {
		t.Errorf("stderr missing gate-summary failure: %q", stderr.String())
	}
}
