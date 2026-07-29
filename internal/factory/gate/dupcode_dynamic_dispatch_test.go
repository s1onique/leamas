// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"fmt"
	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// countingDupcodeRunner counts every protected-operation call. The
// production runner is replaced with this fake via the invocation-local
// dupcodeBinderDeps injection point.
type countingDupcodeRunner struct {
	newRunnerCalls    atomic.Int64
	loadBaselineCalls atomic.Int64
	scanCalls         atomic.Int64
	compareCalls      atomic.Int64
	verifyCalls       atomic.Int64
	writeCalls        atomic.Int64

	// Baseline fingerprint returned by LoadBaseline.
	baselineToReturn dupcode.Baseline
	// Report returned by RunCheckReport.
	reportToReturn dupcode.Report
	// Findings returned by VerifyBaseline.
	verifyFindingsToReturn []checks.Finding

	// Reject everything with this error if set.
	loadErr   error
	scanErr   error
	verifyErr error
	writeErr  error
}

func (r *countingDupcodeRunner) LoadBaseline(path string) (dupcode.Baseline, error) {
	r.loadBaselineCalls.Add(1)
	return r.baselineToReturn, r.loadErr
}

func (r *countingDupcodeRunner) RunCheckRepo(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
	r.scanCalls.Add(1)
	return nil, nil
}

func (r *countingDupcodeRunner) RunCheckReport(root string, cfg dupcode.Config) (dupcode.Report, error) {
	r.scanCalls.Add(1)
	if r.scanErr != nil {
		return dupcode.Report{}, r.scanErr
	}
	return r.reportToReturn, nil
}

func (r *countingDupcodeRunner) VerifyBaseline(root string, policy dupcode.BaselinePolicy) ([]checks.Finding, error) {
	r.verifyCalls.Add(1)
	if r.verifyErr != nil {
		return nil, r.verifyErr
	}
	return r.verifyFindingsToReturn, nil
}

func (r *countingDupcodeRunner) WriteBaseline(path string, report dupcode.Report) error {
	r.writeCalls.Add(1)
	return r.writeErr
}

func (r *countingDupcodeRunner) CompareToBaseline(report dupcode.Report, baseline dupcode.Baseline) dupcode.CompareResult {
	r.compareCalls.Add(1)
	return dupcode.CompareResult{
		NewFindings: []dupcode.NewFinding{{
			Fingerprint: "fake-fingerprint",
			TokenCount:  101,
			LineCount:   42,
			Occurrences: []dupcode.BaselineOccurrence{{Path: "fake/path.go", StartLine: 1, EndLine: 42}},
		}},
		WorsenedFindings: []dupcode.WorsenedFinding{{
			Fingerprint:         "worsened-fingerprint",
			BaselineOccurrences: []dupcode.BaselineOccurrence{{Path: "baseline/path.go", StartLine: 5, EndLine: 50}},
			NewOccurrences:      []dupcode.BaselineOccurrence{{Path: "new/path.go", StartLine: 1, EndLine: 60}},
			TotalNow:            250,
		}},
		HasChanges: true,
	}
}

// denyingObserver returns an ExecutionContext that always fails authority
// validation. Used by denial-zero tests.
type denyingObserver struct{}

func (d *denyingObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		AuthorityMarker: "deny",
		// All other fields empty -> authority will reject.
	}
}

// admittingObserver returns an ExecutionContext that admits the request
// for a CI-exact-checkout authority. Used by admission-exactly-once tests.
type admittingObserver struct{}

func (a *admittingObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: verifierauthority.AuthorityMarker,
		GitHubSHA:       "abc123def456abc123def456abc123def456abcd",
		GitHubWorkspace: root,
		HeadCommit:      "abc123def456abc123def456abc123def456abcd",
		WorktreeStatus:  "",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}

// localSafeObserver returns an explicitly classified local execution
// context. It is the trusted observer pattern: the only way to obtain
// EnvironmentLocal is via the authority-package observation provenance,
// never via implicit "all environment strings empty == local". This
// observer delegates to NewLocalOnlyContext, which records the local
// classification through the unexported observation field.
type localSafeObserver struct{}

func (l *localSafeObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return *verifierauthority.NewLocalOnlyContext()
}

// makeVerifyDeps wires the runner through a counting factory.
func makeVerifyDeps(r *countingDupcodeRunner) dupcodeBinderDeps {
	return dupcodeBinderDeps{
		NewRunner: func() dupcodeRunner {
			r.newRunnerCalls.Add(1)
			return r
		},
	}
}

func makeBaselineDeps(r *countingDupcodeRunner) dupcodeBinderDeps {
	return dupcodeBinderDeps{
		NewRunner: func() dupcodeRunner {
			r.newRunnerCalls.Add(1)
			return r
		},
	}
}

func makeUpdateDeps(r *countingDupcodeRunner) dupcodeBinderDeps {
	return dupcodeBinderDeps{
		NewRunner: func() dupcodeRunner {
			r.newRunnerCalls.Add(1)
			return r
		},
	}
}

// TestDupcodeVerifyDenied proves that authority denial performs zero
// protected work for the verify lane.
func TestDupcodeVerifyDenied(t *testing.T) {
	r := &countingDupcodeRunner{}
	outcome := dispatchDupcodeVerifyTypedWith(
		context.Background(), ".", DupcodeVerifySpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &denyingObserver{}, makeVerifyDeps(r),
	)
	if outcome.Dispatch.Error == nil && len(outcome.Dispatch.Findings) == 0 {
		t.Fatalf("expected denial: error=%v findings=%v", outcome.Dispatch.Error, outcome.Dispatch.Findings)
	}
	if got := r.newRunnerCalls.Load(); got != 0 {
		t.Errorf("newRunnerCalls = %d, want 0 (denial before factory)", got)
	}
	if got := r.loadBaselineCalls.Load(); got != 0 {
		t.Errorf("loadBaselineCalls = %d, want 0", got)
	}
	if got := r.scanCalls.Load(); got != 0 {
		t.Errorf("scanCalls = %d, want 0", got)
	}
	if got := r.compareCalls.Load(); got != 0 {
		t.Errorf("compareCalls = %d, want 0", got)
	}
	if len(outcome.Report.Findings) != 0 || outcome.Report.FindingCount != 0 || outcome.Report.Root != "" {
		t.Errorf("Report = %+v, want zero", outcome.Report)
	}
	if outcome.Comparison.HasChanges || outcome.Comparison.NewCount != 0 || outcome.Comparison.WorsenedCount != 0 || len(outcome.Comparison.NewFindings) != 0 || len(outcome.Comparison.WorsenedFindings) != 0 {
		t.Errorf("Comparison = %+v, want zero", outcome.Comparison)
	}
}

// TestDupcodeBaselineDenied proves that authority denial performs zero
// protected work for the dupcode-baseline lane.
func TestDupcodeBaselineDenied(t *testing.T) {
	r := &countingDupcodeRunner{}
	outcome := dispatchDupcodeBaselineVerifyTypedWith(
		context.Background(), ".", DupcodeBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &denyingObserver{}, makeBaselineDeps(r),
	)
	if outcome.Dispatch.Error == nil && len(outcome.Dispatch.Findings) == 0 {
		t.Fatalf("expected denial: error=%v findings=%v", outcome.Dispatch.Error, outcome.Dispatch.Findings)
	}
	if got := r.newRunnerCalls.Load(); got != 0 {
		t.Errorf("newRunnerCalls = %d, want 0", got)
	}
	if got := r.verifyCalls.Load(); got != 0 {
		t.Errorf("verifyCalls = %d, want 0", got)
	}
	if outcome.Findings != nil {
		t.Errorf("Findings cell = %+v, want nil (denial leaves cell empty)", outcome.Findings)
	}
}

// writeFakeBaseline writes a minimal valid baseline file to dir/name and
// returns its path. It exists so admission tests can drive the verify
// lane past the missing_baseline early-return into LoadBaseline/Compare.
func writeFakeBaseline(t *testing.T, dir, name string, minLines, minTokens int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	content := []byte(fmt.Sprintf(`{"schema_version":1,"generated_at":"test","tool":"test","thresholds":{"min_lines":%d,"min_tokens":%d},"findings":[]}`, minLines, minTokens))
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	return path
}
