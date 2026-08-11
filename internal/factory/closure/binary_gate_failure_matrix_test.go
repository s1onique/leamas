// SPDX-License-Identifier: Apache-2.0

// binary_gate_failure_matrix_test.go owns the canonical
// 12-row R6-B failure matrix that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-BINARY-GATE-
// INTEGRATION01-CORRECTION07 requires. Each row exercises
// one R6-B failure family at the production integration
// boundary and asserts the typed V2DiagnosticCode the
// owned R6-B authority surfaces.
//
// The matrix schema (r6BFailureOwner) is test metadata
// only; it does not introduce a new production diagnostics
// framework. Splitting the schema from the test body
// keeps the file under the LLM-friendly 400-line threshold.

package closure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// r6BFailureOwner is the canonical closed set of R6-B
// failure-owning authorities the integration can produce.
// The enum is test metadata: it does NOT introduce a new
// production diagnostics framework.
type r6BFailureOwner string

const (
	ownerB1Build           r6BFailureOwner = "b1_build"
	ownerBinaryAuthority   r6BFailureOwner = "binary_authority"
	ownerUnsafeOutput      r6BFailureOwner = "unsafe_output"
	ownerGateObservation   r6BFailureOwner = "gate_observation"
	ownerGateUnavailable   r6BFailureOwner = "gate_unavailable"
	ownerGateFailed        r6BFailureOwner = "gate_failed"
	ownerCollectorIdentity r6BFailureOwner = "collector_identity"
	ownerSubjectCleanup    r6BFailureOwner = "subject_cleanup"
)

// r6BMatrixRow is the schema the strict 12-row matrix
// uses. Each row records the test-time setup, the
// expected typed V2 code (or typed sentinel), and the
// mandatory B2 consequence requirement.
//
// Set runner to construct the CommandRunner the
// GateCollector uses. Set setup to inject any additional
// failure-injection seams (buildFn, gitClient). Set
// expectCauseIs to assert the wrapped cause via
// errors.Is (e.g. evidence.ErrCollectorRequestMismatch).
//
// b2Consequence is REQUIRED for every row. There is no
// opt-out from B2 consequence proof; rows must declare
// whether the failure surfaces before the candidate is
// constructible (consequenceCandidateUnreachable) or
// surfaces after, in which case B2 must reject the
// candidate as INCOMPLETE (consequenceBarrierRejects).
type r6BMatrixRow struct {
	name          string
	owner         r6BFailureOwner
	setup         func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps
	expectCode    V2DiagnosticCode
	expectCause   error // optional; checked via errors.Is
	b2Consequence b2Consequence
}

// r6BMatrixRows returns the canonical 12-row matrix. The
// list length is guarded by TestClosureBinaryGateFailureMatrix
// before any row is exercised.
func r6BMatrixRows() []r6BMatrixRow {
	return []r6BMatrixRow{
		{
			name:  "01_b1_build_failure",
			owner: ownerB1Build,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn: func(context.Context, ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
						return ExactSubjectBinaryResult{}, errors.New("simulated b1 build failure")
					},
				}
			},
			expectCode:    "", // B1 family surfaces as a non-V2Error in production
			b2Consequence: consequenceCandidateUnreachable,
		},
		{
			name:  "02_wrong_b1_identity",
			owner: ownerBinaryAuthority,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				binaryPath := filepath.Join(t.TempDir(), "leamas-wrong-identity")
				if err := os.WriteFile(binaryPath, []byte("wrong\n"), 0o755); err != nil {
					t.Fatalf("write binary: %v", err)
				}
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn: makeFakeBinaryBuilderWithCommit(binaryPath, strings.Repeat("0", 40)),
				}
			},
			expectCode:    V2CodeR6BBinaryAuthorityInvalid,
			b2Consequence: consequenceCandidateUnreachable,
		},
		{
			name:  "03_unsafe_output_root",
			owner: ownerUnsafeOutput,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn: makeFakeBinaryBuilderWithUnsafeOutput(),
				}
			},
			expectCode:    "", // permission-denied surfaces as a non-V2Error
			b2Consequence: consequenceCandidateUnreachable,
		},
		{
			name:  "04_gate_spawn_failure",
			owner: ownerGateObservation,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn:        r6BStubBuildFn(t),
					NewCollectorFn: nil, // default constructor
					CommandRunner:  &r6BRecordingRunner{spawnFail: true},
					OutputRoot:     r6BOutputRoot(t),
					OutputName:     "leamas",
					RunID:          "r6b-row-spawn",
					EvidenceDir:    r6BEvidenceDir(t),
				}
			},
			expectCode:    V2CodeR6BGateObservationFailed,
			b2Consequence: consequenceCandidateUnreachable,
		},
		{
			name:  "05_gate_timeout",
			owner: ownerGateUnavailable,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn:       r6BStubBuildFn(t),
					CommandRunner: &r6BRecordingRunner{timeOut: true},
					OutputRoot:    r6BOutputRoot(t),
					OutputName:    "leamas",
					RunID:         "r6b-row-timeout",
					EvidenceDir:   r6BEvidenceDir(t),
				}
			},
			expectCode:    V2CodeR6BGateClassificationUnavailable,
			b2Consequence: consequenceCandidateUnreachable,
		},
		{
			name:  "06_gate_stdout_truncation",
			owner: ownerGateUnavailable,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn:       r6BStubBuildFn(t),
					CommandRunner: &r6BRecordingRunner{stdoutTrunc: true},
					OutputRoot:    r6BOutputRoot(t),
					OutputName:    "leamas",
					RunID:         "r6b-row-stdout-trunc",
					EvidenceDir:   r6BEvidenceDir(t),
				}
			},
			expectCode:    V2CodeR6BGateClassificationUnavailable,
			b2Consequence: consequenceCandidateUnreachable,
		},
		{
			name:  "07_gate_stderr_truncation",
			owner: ownerGateUnavailable,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn:       r6BStubBuildFn(t),
					CommandRunner: &r6BRecordingRunner{stderrTrunc: true},
					OutputRoot:    r6BOutputRoot(t),
					OutputName:    "leamas",
					RunID:         "r6b-row-stderr-trunc",
					EvidenceDir:   r6BEvidenceDir(t),
				}
			},
			expectCode:    V2CodeR6BGateClassificationUnavailable,
			b2Consequence: consequenceCandidateUnreachable,
		},
		{
			name:  "08_gate_nonzero_exit",
			owner: ownerGateFailed,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn:           r6BStubBuildFn(t),
					CommandRunner:     &r6BRecordingRunner{nonZero: true, stdoutField: []byte("EXEC_GATE_OBSERVED_STATUS:FAILED\ncmd/leamas/main.go:42:warning:rule-new:nonzero-lane finding\n")},
					OutputRoot:        r6BOutputRoot(t),
					OutputName:        "leamas",
					RunID:             "r6b-row-nonzero",
					EvidenceDir:       r6BEvidenceDir(t),
					GateACTOwnedPaths: []string{"cmd/leamas/**"},
				}
			},
			expectCode:    V2CodeR6BGateClassificationFailed,
			b2Consequence: consequenceCandidateUnreachable,
		},
		{
			name:  "09_classifier_fail_independent",
			owner: ownerGateFailed,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn:           r6BStubBuildFn(t),
					CommandRunner:     &r6BRecordingRunner{stdoutField: []byte("EXEC_GATE_OBSERVED_STATUS:FAILED\ncmd/leamas/main.go:42:warning:rule-new:extra finding\n")},
					OutputRoot:        r6BOutputRoot(t),
					OutputName:        "leamas",
					RunID:             "r6b-row-cls-fail",
					EvidenceDir:       r6BEvidenceDir(t),
					GateACTOwnedPaths: []string{"cmd/leamas/**"},
				}
			},
			expectCode:    V2CodeR6BGateClassificationFailed,
			b2Consequence: consequenceCandidateUnreachable,
		},
		{
			name:  "10_classifier_unavailable_independent",
			owner: ownerGateUnavailable,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn:       r6BStubBuildFn(t),
					CommandRunner: &r6BRecordingRunner{stdoutField: []byte("lane-lint: OK\n")},
					OutputRoot:    r6BOutputRoot(t),
					OutputName:    "leamas",
					RunID:         "r6b-row-cls-unavail",
					EvidenceDir:   r6BEvidenceDir(t),
				}
			},
			expectCode:    V2CodeR6BGateClassificationUnavailable,
			b2Consequence: consequenceCandidateUnreachable,
		},
		{
			name:  "11_collector_identity_mismatch",
			owner: ownerCollectorIdentity,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				collectorRunner := &r6BCollectorMismatchingRunner{}
				collector := evidence.NewGateCollector(collectorRunner)
				firstSubjectRoot := filepath.Join(t.TempDir(), "subj-pre")
				if err := os.MkdirAll(firstSubjectRoot, 0o700); err != nil {
					t.Fatalf("mkdir first: %v", err)
				}
				if _, err := collector.Capture(context.Background(), evidence.GateCaptureRequest{
					RepositoryRoot: dir,
					EvidenceDir:    filepath.Join(t.TempDir(), "ev"),
					RunID:          "pre",
					SubjectRoot:    firstSubjectRoot,
					MakeExecutable: []string{"/bin/true"},
				}); err != nil {
					t.Fatalf("first capture: %v", err)
				}
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn:        r6BStubBuildFn(t),
					NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
					CommandRunner:  &r6BRecordingRunner{},
					OutputRoot:     r6BOutputRoot(t),
					OutputName:     "leamas",
					RunID:          "r6b-row-collector-mismatch",
					EvidenceDir:    r6BEvidenceDir(t),
				}
			},
			expectCode:    V2CodeR6BGateObservationFailed,
			expectCause:   evidence.ErrCollectorRequestMismatch,
			b2Consequence: consequenceCandidateUnreachable,
		},
		{
			name:  "12_subject_cleanup_failure",
			owner: ownerSubjectCleanup,
			setup: func(t *testing.T, dir, freeze, subject string) RunClosureProtocolV2ExecuteDeps {
				return RunClosureProtocolV2ExecuteDeps{
					BuildFn:       r6BStubBuildFn(t),
					CommandRunner: &r6BRecordingRunner{stdoutField: []byte("EXEC_GATE_OBSERVED_STATUS:OK\n")},
					OutputRoot:    r6BOutputRoot(t),
					OutputName:    "leamas",
					RunID:         "r6b-row-cleanup",
					EvidenceDir:   r6BEvidenceDir(t),
					GitClient:     r6BRealSubjectCleanupFailureGitClient(),
				}
			},
			expectCode:    V2CodeR6BSubjectCleanupFailed,
			b2Consequence: consequenceCandidateUnreachable,
		},
	}
}

// TestClosureBinaryGateFailureMatrix is the strict 12-row
// R6-B failure matrix that proves every owned R6-B failure
// family surfaces at the production integration boundary.
// The matrix is guard-locked at exactly 12 rows; any
// addition or removal MUST update the documented count
// and the corresponding acceptance line.
func TestClosureBinaryGateFailureMatrix(t *testing.T) {
	t.Parallel()
	rows := r6BMatrixRows()
	if len(rows) != 12 {
		t.Fatalf("matrix has %d rows, want exactly 12", len(rows))
	}
	// Surface the count so the act's acceptance line is
	// visible in the test output.
	t.Logf("FAILURE_MATRIX_DECLARED_ROWS=%d", len(rows))
	semanticRows := 0
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			dir := r6BInitRepo(t)
			freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
				"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
			})
			subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
			deps := row.setup(t, dir, freeze, subject)
			// Provide the NewCollectorFn default only when
			// the row didn't already provide one.
			if deps.NewCollectorFn == nil && deps.CommandRunner != nil {
				cr := deps.CommandRunner
				deps.NewCollectorFn = func(_ evidence.CommandRunner) *evidence.GateCollector {
					return evidence.NewGateCollector(cr)
				}
			}
			_, obs, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
				r6BRequestFor(t, dir, freeze, subject),
				newR6BTestBinaryIdentity(t),
				deps)
			if err == nil {
				t.Fatalf("row %q (owner=%s) must fail closed, got nil", row.name, row.owner)
			}
			// Owned-failure assertion: typed code (or
			// acknowledged non-V2Error family for B1).
			if row.expectCode != "" {
				requireV2Code(t, err, row.expectCode)
			}
			if row.expectCause != nil {
				if !errors.Is(err, row.expectCause) {
					t.Fatalf("row %q: err = %v, want errors.Is(%v)", row.name, err, row.expectCause)
				}
			}
			// B2 consequence (mandatory for every row):
			// the candidate cannot cross the barrier, either
			// because the observation was rejected before it
			// could be assembled, or because the B2 barrier
			// rejects it. The row declares which consequence
			// applies; the helper asserts it.
			buildCandidate := func() error {
				_, prepErr := evidence.PrepareClosureEvidenceForPublication(
					evidence.BuildClosureEvidenceCandidate(evidence.CandidateInputs{
						Runtime:      obs.Runtime,
						Results:      obs.Results,
						Gate:         obs.Gate,
						Binary:       obs.Binary,
						CallerBefore: obs.CallerBefore,
						CallerAfter:  obs.CallerAfter,
						Cleanup:      obs.Cleanup,
					}))
				return prepErr
			}
			requireB2Consequence(t, row.b2Consequence, obs, buildCandidate)
			semanticRows++
		})
	}
	t.Logf("FAILURE_MATRIX_SEMANTIC_ROWS=%d", semanticRows)
	if semanticRows != 12 {
		t.Fatalf("matrix semantic rows = %d, want 12", semanticRows)
	}
}
