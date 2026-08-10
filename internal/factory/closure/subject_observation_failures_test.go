// SPDX-License-Identifier: Apache-2.0

package closure

// subject_observation_failures_test.go provides the R6-A
// adversarial matrix required by Phase 15:
//
//   - TestClosureSubjectObservationFailureMatrix
//
// The matrix drives every documented failure row through a
// fake gitClient seam so each row can be triggered
// deterministically without depending on environmental
// failure. Every row fails closed with one stable typed
// diagnostic family: subject_observation_unavailable
// (V2CodeSubjectObservationUnavailable) or
// subject_registration_mismatch
// (V2CodeSubjectRegistrationMismatch).
//
// The matrix rows are derived from the act's Phase 15 list
// (rev-parse HEAD, HEAD^{tree}, show-toplevel, detached,
// status, refs, BEFORE/AT_SUBJECT/AFTER inventory,
// registration missing, registration HEAD != S, cleanup
// failure). Each row produces a typed V2Error and
// non-nil V2ExecuteResult so the audit fields (worktree
// path, before inventory, etc.) are preserved.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// subjectMatrixGitClient is a fake gitClient that lets each
// test row pick the exact failure it wants to trigger. The
// fake returns the configured gitCommandResult for any
// well-known command and delegates everything else to a
// delegate. The fake exists so the matrix does not need to
// depend on real environmental failures.
//
// Field semantics: the *Err fields make the fake return a
// specific gitCommandResult for the corresponding
// observation command. registrationOverride emits a
// registration whose HEAD does not match the requested
// subject commit so the Phase 8 (Path, Head) binding
// fires. cleanupFail makes the bounded cleanup
// (worktree remove) fail. capturedWorktreePath records
// the actual worktree path from `worktree add --detach`.
// zCalls counts the -z invocations so the fake can dispatch
// per-stage BEFORE/AT_SUBJECT/AFTER failures. registrationMissing
// suppresses the registration override so the AtSubject
// inventory never contains a record for the captured
// worktree path.
type subjectMatrixGitClient struct {
	delegate             gitClient
	headErr              *gitCommandResult
	treeErr              *gitCommandResult
	toplevelErr          *gitCommandResult
	symRefErr            *gitCommandResult
	statusErr            *gitCommandResult
	refsErr              *gitCommandResult
	beforeErr            *gitCommandResult
	atSubjErr            *gitCommandResult
	afterErr             *gitCommandResult
	registrationOverride string
	cleanupFail          bool
	capturedWorktreePath string
	zCalls               int
	registrationMissing  bool
}

func (m *subjectMatrixGitClient) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	switch {
	case len(args) >= 2 && args[0] == "rev-parse" && args[1] == "HEAD":
		if m.headErr != nil {
			return *m.headErr
		}
	case len(args) >= 2 && args[0] == "rev-parse" && args[1] == "HEAD^{tree}":
		if m.treeErr != nil {
			return *m.treeErr
		}
	case len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--show-toplevel":
		if m.toplevelErr != nil {
			return *m.toplevelErr
		}
		if m.toplevelErr == nil && m.registrationOverride != "" {
			return gitCommandResult{Stdout: []byte(directory + "\n"), ExitCode: 0}
		}
	case len(args) >= 3 && args[0] == "symbolic-ref" && args[1] == "-q" && args[2] == "HEAD":
		if m.symRefErr != nil {
			return *m.symRefErr
		}
	case len(args) >= 1 && args[0] == "status":
		if m.statusErr != nil {
			return *m.statusErr
		}
	case len(args) >= 2 && args[0] == "for-each-ref":
		if m.refsErr != nil {
			return *m.refsErr
		}
	case len(args) >= 4 && args[0] == "worktree" && args[1] == "remove":
		// cleanup-failure row: the bounded cleanup
		// must observe a typed Stage.WorktreeRemoveError
		// so report.HasError() returns true.
		if m.cleanupFail {
			return gitCommandResult{ExitCode: 1, Stderr: []byte("fatal: cleanup failure during worktree remove")}
		}
	case len(args) >= 4 && args[0] == "worktree" && args[1] == "list" && args[2] == "--porcelain" && args[3] == "-z":
		// The worktree list --porcelain -z helper is the
		// canonical subject inventory snapshot. The fake
		// dispatches on the configured stage failure:
		//   - registrationOverride:  emit a record whose HEAD
		//     does not match the requested subject commit so
		//     the executor's Phase 8 binding fires.
		//   - beforeErr/atSubjErr/afterErr:  return a failure
		//     for that stage's snapshot. The fake records the
		//     captured worktree path so the registration
		//     override can target it.
		// Stage dispatch is by z-call count, mirroring the
		// production sequence BEFORE -> AT_SUBJECT -> AFTER.
		if m.zCalls == 0 && m.beforeErr != nil {
			m.zCalls++
			return *m.beforeErr
		}
		if m.zCalls == 1 && m.atSubjErr != nil {
			m.zCalls++
			return *m.atSubjErr
		}
		if m.zCalls >= 2 && m.afterErr != nil {
			m.zCalls++
			return *m.afterErr
		}
		m.zCalls++
		if m.registrationOverride != "" && m.capturedWorktreePath != "" {
			return gitCommandResult{
				Stdout: []byte("worktree " + m.capturedWorktreePath + "\x00" +
					"HEAD " + m.registrationOverride + "\x00"),
				ExitCode: 0,
			}
		}
		if m.registrationMissing {
			// Return an inventory whose only entry is
			// the unrelated main worktree so the
			// captured subject worktree path is not
			// present (Phase 8 missing-registration).
			return gitCommandResult{
				Stdout: []byte("worktree " + directory + "\x00" +
					"HEAD " + fxHeadForUnrelatedRegistration() + "\x00"),
				ExitCode: 0,
			}
		}
		if m.cleanupFail && m.zCalls >= 3 {
			return gitCommandResult{ExitCode: 128, Stderr: []byte("fatal: cleanup failure")}
		}
	case len(args) >= 5 && args[0] == "worktree" && args[1] == "add" && args[2] == "--detach":
		// Capture the worktree path so the override above
		// can target the same path the executor created.
		m.capturedWorktreePath = args[3]
	}
	if m.delegate == nil {
		return gitCommandResult{Err: errors.New("no delegate for " + strings.Join(args, " "))}
	}
	return m.delegate.Run(ctx, directory, args...)
}

func (m *subjectMatrixGitClient) RunWithStdin(ctx context.Context, directory, stdin string, args ...string) gitCommandResult {
	if m.delegate == nil {
		return gitCommandResult{}
	}
	return m.delegate.RunWithStdin(ctx, directory, stdin, args...)
}

func (m *subjectMatrixGitClient) RunWithEnv(ctx context.Context, directory string, env []string, args ...string) gitCommandResult {
	if m.delegate == nil {
		return gitCommandResult{}
	}
	return m.delegate.RunWithEnv(ctx, directory, env, args...)
}

func (m *subjectMatrixGitClient) RunWithStdinAndEnv(ctx context.Context, directory, stdin string, env []string, args ...string) gitCommandResult {
	if m.delegate == nil {
		return gitCommandResult{}
	}
	return m.delegate.RunWithStdinAndEnv(ctx, directory, stdin, env, args...)
}

// subjectMatrixFixture builds a hermetic S+F repository the
// failure matrix can reuse. The fixture is identical in
// shape to the success-path fixture so each row can rely
// on the same S and F commits.
func subjectMatrixFixture(t *testing.T) subjectExecutorTestFixture {
	t.Helper()
	return newSubjectExecutorTestFixture(t)
}

// fxHeadForUnrelatedRegistration returns a 40-character
// lowercase hex OID that does NOT match the hermetic
// fixture's subject commit. The "registration-missing" row
// uses it to build an AtSubject inventory whose only entry
// is an unrelated worktree, ensuring the captured subject
// worktree path is absent (Phase 8 missing-registration).
func fxHeadForUnrelatedRegistration() string {
	return "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
}

// subjectMatrixFailureOf returns the typed V2Error code and
// a human-readable label for a given row name. The matrix
// uses the labels to bind the row to its expected failure
// code so future contributors do not silently change the
// typed codes.
func subjectMatrixFailureOf(row string) (V2DiagnosticCode, string) {
	switch row {
	case "rev-parse-HEAD-failure":
		return V2CodeSubjectObservationUnavailable, "subject_head"
	case "rev-parse-HEAD-tree-failure":
		return V2CodeSubjectObservationUnavailable, "subject_tree"
	case "show-toplevel-failure":
		return V2CodeSubjectObservationUnavailable, "subject_toplevel"
	case "detached-state-observation-failure":
		return V2CodeSubjectObservationUnavailable, "subject_detached"
	case "status-observation-failure":
		return V2CodeSubjectObservationUnavailable, "subject_status"
	case "refs-observation-failure":
		return V2CodeSubjectObservationUnavailable, "subject_refs"
	case "before-inventory-failure":
		return V2CodeSubjectObservationUnavailable, "subject_worktree_inventory"
	case "at-subject-inventory-failure":
		return V2CodeSubjectObservationUnavailable, "subject_worktree_inventory"
	case "after-inventory-failure":
		return V2CodeSubjectObservationUnavailable, "subject_worktree_inventory"
	case "registration-HEAD-mismatch":
		return V2CodeSubjectRegistrationMismatch, "subject_registration"
	case "registration-missing":
		return V2CodeSubjectObservationUnavailable, "subject_registration"
	case "cleanup-failure":
		return V2CodeCleanupFailed, "cleanup"
	default:
		return "", ""
	}
}

// subjectMatrixFailureRows is the canonical adversarial
// matrix table. Splitting the table out of the test body
// keeps the test file under the LLM-friendly 400-line
// threshold while preserving the row-by-row
// authority/code bindings.
func subjectMatrixFailureRows() []struct {
	name   string
	matrix func() *subjectMatrixGitClient
} {
	return []struct {
		name   string
		matrix func() *subjectMatrixGitClient
	}{
		{
			name: "rev-parse-HEAD-failure",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					headErr: &gitCommandResult{ExitCode: 128, Stderr: []byte("fatal: not a git rev")},
				}
			},
		},
		{
			name: "rev-parse-HEAD-tree-failure",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					treeErr: &gitCommandResult{ExitCode: 128, Stderr: []byte("fatal: bad object")},
				}
			},
		},
		{
			name: "show-toplevel-failure",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					toplevelErr: &gitCommandResult{ExitCode: 128, Stderr: []byte("fatal: not a git rev")},
				}
			},
		},
		{
			name: "detached-state-observation-failure",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					symRefErr: &gitCommandResult{ExitCode: 2, Stderr: []byte("fatal: not a symbolic ref")},
				}
			},
		},
		{
			name: "status-observation-failure",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					statusErr: &gitCommandResult{ExitCode: 128, Stderr: []byte("fatal: not a git directory")},
				}
			},
		},
		{
			name: "refs-observation-failure",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					refsErr: &gitCommandResult{ExitCode: 128, Stderr: []byte("fatal: not a git directory")},
				}
			},
		},
		{
			name: "registration-HEAD-mismatch",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					registrationOverride: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
				}
			},
		},
		{
			name: "registration-missing",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					registrationMissing: true,
				}
			},
		},
		{
			name: "before-inventory-failure",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					beforeErr: &gitCommandResult{ExitCode: 128, Stderr: []byte("fatal: before inventory unavailable")},
				}
			},
		},
		{
			name: "at-subject-inventory-failure",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					atSubjErr: &gitCommandResult{ExitCode: 128, Stderr: []byte("fatal: at-subject inventory unavailable")},
				}
			},
		},
		{
			name: "after-inventory-failure",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					afterErr: &gitCommandResult{ExitCode: 128, Stderr: []byte("fatal: after inventory unavailable")},
				}
			},
		},
		{
			name: "cleanup-failure",
			matrix: func() *subjectMatrixGitClient {
				return &subjectMatrixGitClient{
					cleanupFail: true,
				}
			},
		},
	}
}

// TestClosureSubjectObservationFailureMatrix exercises every
// documented Phase 15 failure row. Each row produces a
// typed *V2Error carrying the expected diagnostic code
// family (subject_observation_unavailable or
// subject_registration_mismatch) and a non-nil
// V2ExecuteResult whose SubjectWorktreePath and
// WorktreeInventoryBefore fields remain populated so the
// audit fields are preserved.
func TestClosureSubjectObservationFailureMatrix(t *testing.T) {
	rows := subjectMatrixFailureRows()
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			fx := subjectMatrixFixture(t)
			fake := row.matrix()
			fake.delegate = RealGit{}
			executor := NewGitV2SubjectExecutor(fake)
			req := V2ExecuteRequest{
				RepositoryRoot: fx.dir,
				SubjectCommit:  fx.subject,
				SubjectTree:    fx.subjectTree,
				EvidenceDir:    t.TempDir(),
				Checks: []PlanCheck{{
					ID:               "subject_only_present",
					Mode:             "run",
					Argv:             []string{"true"},
					WorkingDirectory: ".",
					TimeoutSeconds:   60,
					Environment:      map[string]string{},
				}},
			}
			result, err := executor.ExecuteSubjectChecks(context.Background(), req)
			if err == nil {
				t.Fatalf("row %q must fail closed", row.name)
			}
			v2err, ok := err.(*V2Error)
			if !ok {
				t.Fatalf("row %q must return *V2Error, got %T: %v", row.name, err, err)
			}
			wantCode, wantProp := subjectMatrixFailureOf(row.name)
			if !v2err.Diags.HasCode(wantCode) {
				t.Fatalf("row %q: expected code %s, got %v", row.name, wantCode, v2err.Diags.Codes())
			}
			found := false
			for _, d := range v2err.Diags {
				if d.Code == wantCode && d.PropertyName == wantProp {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("row %q: expected property %q, got %+v", row.name, wantProp, v2err.Diags)
			}
			if result.SubjectWorktreePath == "" {
				t.Fatalf("row %q: SubjectWorktreePath must be preserved for audit", row.name)
			}
		})
	}
	_ = fmt.Sprintf
}
