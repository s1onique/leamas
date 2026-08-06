// SPDX-License-Identifier: Apache-2.0

// Package evidence - evidence_test.go implements the matrix
// tests required by CORRECTION01.

package evidence

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeRunner is a deterministic CommandRunner.
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
	if len(capture.LaneResults) != 2 {
		t.Errorf("lanes: %d", len(capture.LaneResults))
	}
	if len(capture.PreExistingFindings) != 1 {
		t.Errorf("findings: %d", len(capture.PreExistingFindings))
	}
	if collector.Calls() != 1 {
		t.Errorf("calls: %d (exactly-once)", collector.Calls())
	}
	// Second collector does not see the first collector's calls.
	collector2 := NewGateCollector(&fakeRunner{out: "lane test:OK\n", code: 0})
	evidenceDir2 := filepath.Join(tmp, "evidence2")
	if _, err := collector2.Capture(context.Background(), GateCaptureRequest{
		SubjectRoot: tmp, EvidenceDir: evidenceDir2, RunID: "run-2",
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

// TestClosureGateCaptureRunScoped proves concurrent collectors
// never share data.
func TestClosureGateCaptureRunScoped(t *testing.T) {
	tmp, err := os.MkdirTemp("", "evidence-concurrent-")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(tmp)
	collector := NewGateCollector(&fakeRunner{out: "EXEC_GATE_OBSERVED_STATUS:OK\n", code: 0})
	var wg sync.WaitGroup
	wg.Add(8)
	for i := 0; i < 8; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := collector.Capture(context.Background(), GateCaptureRequest{
				SubjectRoot: tmp,
				EvidenceDir: filepath.Join(tmp, "run-"+string(rune('a'+i))),
				RunID:       "run-concurrent",
			})
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
		{
			name: "observed_ok",
			inputs: ClassificationInputs{
				ObservedStatus:   "OK",
				ObservedFindings: nil,
				BaselineFindings: nil,
				ACTOwnedPaths:    nil,
			},
			want: ACTOwnedPass,
		},
		{
			name:   "observed_skipped",
			inputs: ClassificationInputs{ObservedStatus: "SKIP"},
			want:   ACTOwnedUnavailable,
		},
		{
			name: "timed_out",
			inputs: ClassificationInputs{
				ObservedStatus: "OK",
				LaneTimedOut:   true,
			},
			want: ACTOwnedUnavailable,
		},
		{
			name: "truncated",
			inputs: ClassificationInputs{
				ObservedStatus: "OK",
				LaneTruncated:  true,
			},
			want: ACTOwnedUnavailable,
		},
		{
			name: "failed_with_baseline_only",
			inputs: ClassificationInputs{
				ObservedStatus: "FAILED",
				ObservedFindings: []GateFinding{
					{Path: "cmd/leamas/x.go", Rule: "r1", Severity: "warning"},
				},
				BaselineFindings: []GateFinding{
					{Path: "cmd/leamas/x.go", Rule: "r1", Severity: "warning"},
				},
				ACTOwnedPaths: []string{"internal/factory/closure/**"},
			},
			want: ACTOwnedPass,
		},
		{
			name: "failed_owned_path",
			inputs: ClassificationInputs{
				ObservedStatus: "FAILED",
				ObservedFindings: []GateFinding{
					{Path: "internal/factory/closure/x.go", Rule: "r1"},
				},
				ACTOwnedPaths: []string{"internal/factory/closure/**"},
			},
			want: ACTOwnedFail,
		},
		{
			name: "failed_new_finding",
			inputs: ClassificationInputs{
				ObservedStatus: "FAILED",
				ObservedFindings: []GateFinding{
					{Path: "x.go", Rule: "new-rule"},
				},
				BaselineFindings: nil,
				ACTOwnedPaths:    []string{"internal/factory/closure/**"},
			},
			want: ACTOwnedFail,
		},
		{
			name: "failed_no_findings",
			inputs: ClassificationInputs{
				ObservedStatus:   "FAILED",
				ObservedFindings: nil,
				ACTOwnedPaths:    []string{"internal/factory/closure/**"},
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

// TestClosureEvidenceAtomicPublication exercises the publication
// step.
func TestClosureEvidenceAtomicPublication(t *testing.T) {
	tmp, err := os.MkdirTemp("", "evidence-pub-")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(tmp)
	evidence := ClosureEvidence{
		SchemaVersion: ClosureEvidenceSchemaVersion,
		Runtime: RuntimeContextSubset{
			ACTID:             "ACT-TEST",
			RepositoryRoot:    "/tmp/repo",
			RunID:             "run-1",
			FreezeCommit:      "1111111111111111111111111111111111111111",
			FreezeTree:        "2222222222222222222222222222222222222222",
			SubjectCommit:     "3333333333333333333333333333333333333333",
			SubjectTree:       "4444444444444444444444444444444444444444",
			PlanPath:          "docs/plan.json",
			PlanBlob:          "5555555555555555555555555555555555555555",
			PlanSHA256:        strings.Repeat("a", 64),
			EvidenceDirectory: "/tmp/evidence",
			StartedAt:         "2026-08-06T00:00:00Z",
		},
		Gate: GateCapture{
			RawOutputPath: "/tmp/raw",
			RawSHA256:     strings.Repeat("b", 64),
		},
		Binary: BuiltBinaryEvidence{
			BinaryPath:   "/tmp/bin/leamas",
			BinarySHA256: strings.Repeat("c", 64),
		},
		Completeness: EvidenceComplete,
	}
	outputPath := filepath.Join(tmp, "evidence.json")
	pub, err := PublishClosureEvidence(PublicationRequest{
		OutputPath: outputPath,
		Evidence:   evidence,
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if pub.DocumentPath == "" || pub.SidecarPath == "" {
		t.Fatalf("paths empty: %+v", pub)
	}
	sidecar, err := os.ReadFile(pub.SidecarPath)
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	if !strings.Contains(string(sidecar), pub.DocumentSHA) {
		t.Errorf("sidecar does not match document hash")
	}
	if strings.Contains(string(pub.DocumentBytes), pub.DocumentSHA) {
		t.Errorf("document embeds its own hash")
	}
	var decoded ClosureEvidence
	if err := json.Unmarshal(pub.DocumentBytes, &decoded); err != nil {
		t.Errorf("decode: %v", err)
	}
	if decoded.Runtime.ACTID != evidence.Runtime.ACTID {
		t.Errorf("decoded act id mismatch")
	}
}

// TestClosureEvidenceValidityPredicate asserts the predicate
// rejects documents with contradictory or missing fields.
func TestClosureEvidenceValidityPredicate(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*ClosureEvidence)
		expectErr bool
	}{
		{name: "valid", mutate: nil, expectErr: false},
		{name: "schema_version", mutate: func(e *ClosureEvidence) { e.SchemaVersion = 99 }, expectErr: true},
		{name: "hardcoded_invalid", mutate: func(e *ClosureEvidence) { e.Completeness = "" }, expectErr: true},
		{name: "empty_act", mutate: func(e *ClosureEvidence) { e.Runtime.ACTID = "" }, expectErr: true},
		{name: "bad_freeze", mutate: func(e *ClosureEvidence) { e.Runtime.FreezeCommit = "x" }, expectErr: true},
		{name: "bad_subject", mutate: func(e *ClosureEvidence) { e.Runtime.SubjectCommit = "x" }, expectErr: true},
		{name: "bad_plan_blob", mutate: func(e *ClosureEvidence) { e.Runtime.PlanBlob = "x" }, expectErr: true},
		{name: "bad_plan_sha", mutate: func(e *ClosureEvidence) { e.Runtime.PlanSHA256 = "x" }, expectErr: true},
		{name: "empty_binary", mutate: func(e *ClosureEvidence) { e.Binary.BinaryPath = "" }, expectErr: true},
		{name: "empty_gate", mutate: func(e *ClosureEvidence) { e.Gate.RawOutputPath = "" }, expectErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := goodClosureEvidence()
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

// goodClosureEvidence returns a syntactically valid document.
func goodClosureEvidence() ClosureEvidence {
	return ClosureEvidence{
		SchemaVersion: ClosureEvidenceSchemaVersion,
		Runtime: RuntimeContextSubset{
			ACTID:         "ACT-TEST",
			FreezeCommit:  "1111111111111111111111111111111111111111",
			SubjectCommit: "3333333333333333333333333333333333333333",
			PlanBlob:      "5555555555555555555555555555555555555555",
			PlanSHA256:    strings.Repeat("a", 64),
		},
		Gate: GateCapture{
			RawOutputPath: "/tmp/raw",
			RawSHA256:     strings.Repeat("b", 64),
		},
		Binary: BuiltBinaryEvidence{
			BinaryPath:   "/tmp/bin/leamas",
			BinarySHA256: strings.Repeat("c", 64),
		},
		Completeness: EvidenceComplete,
	}
}

// versionRunner is a fake CommandRunner for the binary tests.
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
// invariants with a fake runner.
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
}
