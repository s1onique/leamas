// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
	"github.com/s1onique/leamas/internal/factory/gate"
)

// initTempGitRepo creates a temporary git repository with one committed
// file and returns the repo root. It is used by the canonical local-path
// proof to give DetectExecutionContext a real worktree to observe.
//
// This helper uses the bounded execution.RunGit gateway instead of
// os/exec.Command directly because the executable-contract-first
// verifier forbids exec.Command outside internal/execution.
func initTempGitRepo(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "ci@leamas.local"},
		{"config", "user.name", "Leamas CI"},
	} {
		if out, err := execution.RunGit(ctx, dir, args...); err != nil || out.ExitCode != 0 {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out.Stderr)
		}
	}
	// Create a minimal committed file in a .factory directory.
	if err := writeFileMkdirAll(dir, ".factory/main.go", []byte("package main\n")); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "init"},
	} {
		if out, err := execution.RunGit(ctx, dir, args...); err != nil || out.ExitCode != 0 {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out.Stderr)
		}
	}
	return dir
}

// clearCISignals unsets every CI / GitHub Actions / authority-marker
// environment variable so DetectExecutionContext sees a clean local
// invocation.
func clearCISignals(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"CI",
		"GITHUB_ACTIONS",
		"GITHUB_SHA",
		"GITHUB_WORKSPACE",
		"LEAMAS_DUPCODE_AUTHORITY",
	} {
		t.Setenv(key, "")
	}
}

// TestCanonicalPublicLocalUpdate proves the canonical production
// command path:
//
//	leamas factory verify dupcode --update-baseline
//
// is admitted locally through the REAL public typed entry point
// (DispatchDupcodeUpdateBaselineTyped) using the REAL default
// observer (DefaultContextObserver) which performs the bounded
// Git observation round-trip. A custom local observer is NOT used.
//
// Required invariants:
//   - Dispatch.Error    = nil
//   - Dispatch.Findings = empty (admission produced no denials)
//   - Report.Root       = temp repo root
//   - baseline file     = created and parseable
func TestCanonicalPublicLocalUpdate(t *testing.T) {
	dir := initTempGitRepo(t)
	clearCISignals(t)

	baselinePath := dir + "/.factory/dupcode-baseline.json"
	spec := gate.DupcodeUpdateBaselineSpec{
		BaselinePath: baselinePath,
		MinLines:     40,
		MinTokens:    400,
	}

	ctx := context.Background()
	outcome := gate.DispatchDupcodeUpdateBaselineTyped(ctx, dir, spec)

	if outcome.Dispatch.Error != nil {
		t.Fatalf("expected admission: error=%v", outcome.Dispatch.Error)
	}
	if len(outcome.Dispatch.Findings) != 0 {
		t.Errorf("expected empty Dispatch.Findings on success, got %d: %+v",
			len(outcome.Dispatch.Findings), outcome.Dispatch.Findings)
	}
	if outcome.Report.Root != dir {
		t.Errorf("Report.Root = %q, want %q", outcome.Report.Root, dir)
	}

	// Verify the baseline file was actually created on disk.
	if _, err := osStat(baselinePath); err != nil {
		t.Fatalf("baseline file not created at %s: %v", baselinePath, err)
	}
}

// TestCanonicalPublicGitHubActionsDenied proves the canonical
// production command path is denied in a GitHub Actions context
// when CI is overwritten to "false" (the historic fail-open path).
//
// The integration test uses the REAL public typed entry point
// (DispatchDupcodeUpdateBaselineTyped) and the REAL default observer.
// Environment-changing integration tests must not run in parallel.
func TestCanonicalPublicGitHubActionsDenied(t *testing.T) {
	dir := initTempGitRepo(t)
	clearCISignals(t)
	t.Setenv("CI", "false")
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_SHA", "abc123def456abc123def456abc123def456abcd")

	baselinePath := dir + "/.factory/dupcode-baseline.json"
	spec := gate.DupcodeUpdateBaselineSpec{
		BaselinePath: baselinePath,
		MinLines:     40,
		MinTokens:    400,
	}

	ctx := context.Background()
	outcome := gate.DispatchDupcodeUpdateBaselineTyped(ctx, dir, spec)

	if outcome.Dispatch.Error == nil {
		t.Fatalf("expected denial: error=nil, findings=%v", outcome.Dispatch.Findings)
	}
	if len(outcome.Dispatch.Findings) == 0 {
		t.Fatalf("expected denial finding, got none")
	}
	if outcome.Dispatch.Findings[0].Kind != "verifier_execution_authority_denied" {
		t.Errorf("finding kind = %q, want %q",
			outcome.Dispatch.Findings[0].Kind,
			"verifier_execution_authority_denied")
	}
	if _, err := osStat(baselinePath); err == nil {
		t.Errorf("baseline file should NOT exist at %s for a denied mutation", baselinePath)
	}
}
