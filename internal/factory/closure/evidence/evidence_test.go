// SPDX-License-Identifier: Apache-2.0

// Package evidence - evidence_test.go implements the matrix
// tests required by Phase 11.

package evidence

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner is a deterministic CommandRunner used by the gate
// capture and classification matrix tests.
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

// TestClosureSingleGateCapture exercises CaptureGate against the
// fake runner and asserts the lane ran exactly once.
func TestClosureSingleGateCapture(t *testing.T) {
	before := CollectorGateInvocationCount()
	tmp, err := os.MkdirTemp("", "evidence-test-")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(tmp)
	runner := &fakeRunner{
		out:  "lane lint:OK\nlane test:OK\nEXEC_GATE_OBSERVED_STATUS:OK\ncmd/leamas/main.go:42:warning:unused:rule-1\n",
		code: 0,
	}
	capture, err := CaptureGate(context.Background(), GateCaptureRequest{
		SubjectRoot: tmp,
		EvidenceDir: filepath.Join(tmp, "evidence"),
		RunID:       "run-test",
	}, runner)
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
	after := CollectorGateInvocationCount()
	if after-before != 1 {
		t.Errorf("invocation count: want 1 got %d", after-before)
	}
	// Second invocation must not re-run the gate.
	if _, err := CaptureGate(context.Background(), GateCaptureRequest{
		SubjectRoot: tmp, EvidenceDir: filepath.Join(tmp, "evidence"), RunID: "run-test",
	}, runner); err != nil {
		t.Fatalf("second capture: %v", err)
	}
	after2 := CollectorGateInvocationCount()
	if after2-before != 1 {
		t.Errorf("invocation count after second call: want 1 got %d", after2-before)
	}
}

// TestClosureACTOwnedGateClassification exercises every branch of
// the Phase 5 rule set.
func TestClosureACTOwnedGateClassification(t *testing.T) {
	cases := []struct {
		name   string
		inputs ClassificationInputs
		want   ACTOwnedClassification
	}{
		{
			name:   "observed_ok",
			inputs: ClassificationInputs{ObservedStatus: "OK"},
			want:   ACTOwnedPass,
		},
		{
			name:   "observed_skipped",
			inputs: ClassificationInputs{ObservedStatus: "SKIP"},
			want:   ACTOwnedUnavailable,
		},
		{
			name:   "timed_out",
			inputs: ClassificationInputs{ObservedStatus: "OK", LaneTimedOut: true},
			want:   ACTOwnedUnavailable,
		},
		{
			name:   "truncated",
			inputs: ClassificationInputs{ObservedStatus: "OK", LaneTruncated: true},
			want:   ACTOwnedUnavailable,
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
// step and asserts the sidecar hash matches the document hash.
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
		Valid: true,
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
	// Confirm document does NOT embed its own hash.
	if strings.Contains(string(pub.DocumentBytes), pub.DocumentSHA) {
		t.Errorf("document embeds its own hash")
	}
	// Round-trip.
	doc, err := os.ReadFile(pub.DocumentPath)
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	if string(doc) != string(pub.DocumentBytes) {
		t.Errorf("document mismatch")
	}
	// Confirm ValidateClosureEvidence roundtrips.
	if err := ValidateClosureEvidence(evidence); err != nil {
		t.Errorf("validate: %v", err)
	}
	var decoded ClosureEvidence
	if err := json.Unmarshal(pub.DocumentBytes, &decoded); err != nil {
		t.Errorf("decode: %v", err)
	}
	if decoded.Runtime.ACTID != evidence.Runtime.ACTID {
		t.Errorf("decoded act id mismatch")
	}
}

// TestClosureExactSubjectBinary exercises the binary build
// invariants with a fake runner that returns a synthetic
// version output.
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
	// Create a fake binary at the output path so the post-build
	// stat and hash succeed.
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
		SourceClean:     true,
		SourceDetached:  true,
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

// versionRunner is a fake CommandRunner that returns the
// supplied stdout.
type versionRunner struct {
	stdout string
}

func (v *versionRunner) Run(ctx context.Context, name string, args []string, dir string, env []string) CommandResult {
	return CommandResult{
		ExitCode: 0,
		Stdout:   []byte(v.stdout),
	}
}
