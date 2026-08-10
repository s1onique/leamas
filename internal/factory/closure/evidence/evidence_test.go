// SPDX-License-Identifier: Apache-2.0

// Package evidence - evidence_test.go provides the umbrella
// tests for the B2 canonical evidence authority.
//
// The file retains the GateCollector classification tests
// from earlier ACTs (those are not regressed by B2) and
// replaces the obsolete PublishClosureEvidence /
// DeriveClosureEvidenceCompleteness stubs with barrier-aware
// tests that exercise the new canonical authority.
package evidence

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeRunner is a deterministic CommandRunner used by the
// GateCollector tests. It is the same shim the previous ACT
// used; the B2 work does not change the collector surface.
type fakeRunner struct {
	calls int
	out   string
	code  int
}

func (f *fakeRunner) Run(ctx context.Context, name string, args []string, dir string, env []string) CommandResult {
	f.calls++
	return CommandResult{
		ExitCode: f.code,
		Stdout:   []byte(f.out),
		Stderr:   []byte(""),
	}
}

// TestClosureSingleGateCapture exercises the per-run
// GateCollector and asserts two independent collectors never
// share state.
func TestClosureSingleGateCapture(t *testing.T) {
	tmp, err := os.MkdirTemp("", "evidence-test-")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(tmp)
	collector := NewGateCollector(&fakeRunner{
		out:  "lane lint:OK\nlane test:OK\nEXEC_GATE_OBSERVED_STATUS:OK\ncmd/leamas/main.go:42:warning:unused:rule-1\n",
		code: 0,
	})
	capture, err := collector.Capture(context.Background(), GateCaptureRequest{
		SubjectRoot: tmp,
		EvidenceDir: filepath.Join(tmp, "evidence"),
		RunID:       "run-test",
	})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if capture.ExitCode != 0 {
		t.Errorf("exit: %d", capture.ExitCode)
	}
	if capture.ExecGateObservedStatus != "OK" {
		t.Errorf("status: %s", capture.ExecGateObservedStatus)
	}
	if collector.Calls() != 1 {
		t.Errorf("calls: %d", collector.Calls())
	}
	// Second collector does not see the first collector's calls.
	collector2 := NewGateCollector(&fakeRunner{out: "lane test:OK\n", code: 0})
	if _, err := collector2.Capture(context.Background(), GateCaptureRequest{
		SubjectRoot: tmp, EvidenceDir: filepath.Join(tmp, "evidence2"), RunID: "run-2",
	}); err != nil {
		t.Fatalf("second capture: %v", err)
	}
	if collector.Calls() != 1 {
		t.Errorf("first collector calls drift: %d", collector.Calls())
	}
	if collector2.Calls() != 1 {
		t.Errorf("second collector calls: %d", collector2.Calls())
	}
}

// TestClosureGateCaptureRunScoped proves concurrent calls for one run share
// the same request identity and cached result.
func TestClosureGateCaptureRunScoped(t *testing.T) {
	tmp, err := os.MkdirTemp("", "evidence-concurrent-")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(tmp)
	collector := NewGateCollector(&fakeRunner{out: "EXEC_GATE_OBSERVED_STATUS:OK\n", code: 0})
	req := GateCaptureRequest{
		SubjectRoot: tmp,
		EvidenceDir: filepath.Join(tmp, "run"),
		RunID:       "run-concurrent",
	}
	var wg sync.WaitGroup
	wg.Add(8)
	for i := 0; i < 8; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := collector.Capture(context.Background(), req)
			if err != nil {
				t.Errorf("capture %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
	if collector.Calls() != 1 {
		t.Errorf("calls: %d (exactly-once)", collector.Calls())
	}
}

// TestClosureACTOwnedGateClassification exercises every branch
// of the Phase 9 rule set with full inputs.
func TestClosureACTOwnedGateClassification(t *testing.T) {
	cases := []struct {
		name   string
		inputs ClassificationInputs
		want   ACTOwnedClassification
	}{
		{name: "observed_ok", inputs: ClassificationInputs{ObservedStatus: "OK"}, want: ACTOwnedPass},
		{name: "observed_skipped", inputs: ClassificationInputs{ObservedStatus: "SKIP"}, want: ACTOwnedUnavailable},
		{name: "timed_out", inputs: ClassificationInputs{ObservedStatus: "OK", LaneTimedOut: true}, want: ACTOwnedUnavailable},
		{name: "truncated", inputs: ClassificationInputs{ObservedStatus: "OK", LaneTruncated: true}, want: ACTOwnedUnavailable},
		{
			name: "failed_with_baseline_only",
			inputs: ClassificationInputs{
				ObservedStatus:   "FAILED",
				ObservedFindings: []GateFinding{{Path: "cmd/leamas/x.go", Rule: "r1", Severity: "warning"}},
				BaselineFindings: []GateFinding{{Path: "cmd/leamas/x.go", Rule: "r1", Severity: "warning"}},
				ACTOwnedPaths:    []string{"internal/factory/closure/**"},
			},
			want: ACTOwnedPass,
		},
		{
			name: "failed_owned_path",
			inputs: ClassificationInputs{
				ObservedStatus:   "FAILED",
				ObservedFindings: []GateFinding{{Path: "internal/factory/closure/x.go", Rule: "r1"}},
				ACTOwnedPaths:    []string{"internal/factory/closure/**"},
			},
			want: ACTOwnedFail,
		},
		{
			name: "failed_new_finding",
			inputs: ClassificationInputs{
				ObservedStatus:   "FAILED",
				ObservedFindings: []GateFinding{{Path: "x.go", Rule: "new-rule"}},
				BaselineFindings: nil,
				ACTOwnedPaths:    []string{"internal/factory/closure/**"},
			},
			want: ACTOwnedFail,
		},
		{
			name: "failed_no_findings",
			inputs: ClassificationInputs{
				ObservedStatus: "FAILED",
				ACTOwnedPaths:  []string{"internal/factory/closure/**"},
			},
			want: ACTOwnedUnavailable,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyACTOwnedGate(tc.inputs)
			if got != tc.want {
				t.Errorf("want %s got %s", tc.want, got)
			}
		})
	}
}

// TestClosureEvidenceValidationStructural exercises the shape
// validator against the canonical evidence fields. The previous
// ACT used ClosureEvidence fields that B2 has refactored; the
// test is rebuilt to use the new canonical types.
func TestClosureEvidenceValidationStructural(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*ClosureEvidence)
		expectErr bool
	}{
		{name: "valid", mutate: nil, expectErr: false},
		{name: "schema_version", mutate: func(e *ClosureEvidence) { e.SchemaVersion = 99 }, expectErr: true},
		{name: "protocol", mutate: func(e *ClosureEvidence) { e.Protocol = "wrong" }, expectErr: true},
		{name: "empty_repository", mutate: func(e *ClosureEvidence) { e.Runtime.RepositoryRoot = "" }, expectErr: true},
		{name: "bad_freeze", mutate: func(e *ClosureEvidence) { e.Runtime.FreezeCommit = "x" }, expectErr: true},
		{name: "bad_subject", mutate: func(e *ClosureEvidence) { e.Runtime.SubjectCommit = "x" }, expectErr: true},
		{name: "bad_plan_blob", mutate: func(e *ClosureEvidence) { e.Runtime.PlanBlob = "x" }, expectErr: true},
		{name: "bad_plan_sha", mutate: func(e *ClosureEvidence) { e.Runtime.PlanSHA256 = "x" }, expectErr: true},
		{name: "empty_binary", mutate: func(e *ClosureEvidence) { e.Binary.BinaryPath = "" }, expectErr: true},
		{name: "empty_plan", mutate: func(e *ClosureEvidence) { e.Plan.ExpectedChecks = nil }, expectErr: true},
		{name: "empty_gate_subject_root", mutate: func(e *ClosureEvidence) { e.Gate.SubjectRoot = "" }, expectErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := validCandidate()
			if tc.mutate != nil {
				tc.mutate(&doc)
			}
			err := ValidateClosureEvidence(doc)
			if tc.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestClosureEvidenceAtomicBarrierRejectsIncomplete proves the
// publication barrier refuses to emit publication bytes for an
// INCOMPLETE candidate. This replaces the previous
// TestClosureEvidenceAtomicPublication that exercised
// PublishClosureEvidence; B2 removed the filesystem writer.
func TestClosureEvidenceAtomicBarrierRejectsIncomplete(t *testing.T) {
	// Build a candidate that is structurally valid but
	// completeness-INCOMPLETE by omitting the F-ancestor-of-S
	// verification.
	candidate := validCandidate()
	candidate.Runtime.FAncestorOfSVerified = false
	got, err := PrepareClosureEvidenceForPublication(candidate)
	if err == nil {
		t.Fatalf("barrier must reject incomplete candidate, got %+v", got)
	}
	if got.Bytes() != nil {
		t.Fatalf("barrier must not return bytes for incomplete candidate")
	}
	if got.SHA256() != "" {
		t.Fatalf("barrier must not return SHA256 for incomplete candidate")
	}
	if !strings.Contains(err.Error(), "incomplete") {
		t.Errorf("expected incomplete rejection, got: %v", err)
	}
}

// versionRunner is a fake CommandRunner used by the binary
// builder tests.
type versionRunner struct {
	stdout string
}

func (v *versionRunner) Run(ctx context.Context, name string, args []string, dir string, env []string) CommandResult {
	return CommandResult{
		ExitCode: 0,
		Stdout:   []byte(v.stdout),
	}
}

// TestClosureExactSubjectBinary exercises the binary build
// invariants.
func TestClosureExactSubjectBinary(t *testing.T) {
	dir, err := os.MkdirTemp("", "binary-build-")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	binPath := filepath.Join(out, "leamas")
	if err := os.WriteFile(binPath, []byte("fake-binary"), 0o755); err != nil {
		t.Fatalf("write fake: %v", err)
	}
	runner := &versionRunner{stdout: `{"vcs_revision":"abc","vcs_modified":false}`}
	be, err := BuildBinary(context.Background(), BuildBinaryRequest{
		SubjectRoot:     dir,
		SubjectCommit:   "abc",
		SubjectTree:     "def",
		OutputDirectory: out,
		OutputName:      "leamas",
		BuildArgv:       []string{"go", "build", "-o", binPath, "./cmd/leamas"},
		Runner:          runner,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if be.VCSRevision != "abc" {
		t.Errorf("vcs revision: %s", be.VCSRevision)
	}
	if be.VCSModified {
		t.Errorf("vcs modified: true")
	}
	if !be.Executable {
		t.Errorf("not executable")
	}
	if be.BinarySHA256 == "" {
		t.Errorf("missing sha256")
	}

	// BinaryAuthorityFromBuild maps the build observability
	// into the canonical authority. The mapping is pure; the
	// test exercises it as a regression guard.
	authority := BinaryAuthorityFromBuild(be, be.VCSRevision, be.SourceTree, true)
	if authority.BinaryCommit != be.VCSRevision {
		t.Errorf("BinaryCommit must equal VCSRevision, got %q", authority.BinaryCommit)
	}
	if authority.BinaryModified != be.VCSModified {
		t.Errorf("BinaryModified must equal VCSModified, got %v", authority.BinaryModified)
	}
	if authority.BinaryPath != be.BinaryPath {
		t.Errorf("BinaryPath mismatch")
	}
	if authority.BinarySHA256 != be.BinarySHA256 {
		t.Errorf("BinarySHA256 mismatch")
	}
	if !authority.OutputOutsideAllWorktrees {
		t.Errorf("OutputOutsideAllWorktrees must be true")
	}
}
