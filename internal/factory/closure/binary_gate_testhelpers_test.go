// SPDX-License-Identifier: Apache-2.0

// binary_gate_testhelpers_test.go owns the small test seams
// the R6-B umbrellas use to drive the production integration
// without re-implementing the B1 build or the gate runner.
// The helpers live in a _test.go file so they are not part of
// the production binary.
//
// CORRECTION08: r6BRecordingRunner follows the production
// OsRunner lifecycle contract exactly:
//
//   - spawnFail     -> StartErr != nil, Err == nil
//   - timeOut       -> StartErr == nil, TimedOut = true
//   - nonZero       -> StartErr == nil, ExitCode != 0
//   - stdoutTrunc   -> StartErr == nil
//   - stderrTrunc   -> StartErr == nil
//
// TestR6BFakeRunnerMatchesProcessLifecycleContract (in
// binary_gate_runner_parity_test.go) asserts the parity
// invariant from Phase 27 of CORRECTION08.
package closure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// r6BInitRepo initialises a fresh hermetic Git repo for the
// R6-B umbrellas. The helper is a thin wrapper around the
// existing initRepo helper so the umbrellas do not have to
// import the production-only symbols directly.
func r6BInitRepo(t *testing.T) string {
	t.Helper()
	return initRepo(t)
}

// r6BMakeCommit creates a commit on the supplied repo with
// the supplied file map. The helper is a thin wrapper around
// the existing makeCommit helper.
func r6BMakeCommit(t *testing.T, dir, message string, files map[string]string) string {
	t.Helper()
	return makeCommit(t, dir, message, files)
}

// r6BValidPlanBytes returns the canonical Plan Contract v1
// plan bytes the R6-B umbrellas use. The bytes satisfy the
// FULL Plan Contract v1 semantic pass.
func r6BValidPlanBytes() []byte {
	b, err := BuildV2ValidPlanFixture("ACT-R6B-CORRECTION01", strings.Repeat("a", 40), strings.Repeat("b", 40))
	if err != nil {
		panic(err)
	}
	return b
}

// r6BRequestFor builds the V2Request the R6-B umbrellas use.
// The helper exists so every umbrella uses the same request
// construction.
func r6BRequestFor(t *testing.T, dir, freeze, subject string) V2Request {
	t.Helper()
	return V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/X.json",
		EvidenceDirectory:      r6BEvidenceDir(t),
		ManifestOutput:         "",
	}
}

// r6BOutputRoot returns a fresh per-run external binary
// output root the R6-B umbrellas use.
func r6BOutputRoot(t *testing.T) string {
	t.Helper()
	// CORRECTION06: on macOS, t.TempDir() returns /var/folders/...
	// which is symlinked to /private/var/folders/.... Resolve before
	// use so the path matches what the inventory records.
	tempDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve temp dir symlinks: %v", err)
	}
	return filepath.Join(tempDir, "leamas-binary-gate")
}

// r6BEvidenceDir returns a fresh per-run external gate
// evidence directory the R6-B umbrellas use.
func r6BEvidenceDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "leamas-gate-evidence")
}

// newR6BTestBinaryIdentity returns a valid V2BinaryIdentity
// the R6-B umbrellas use. The identity names a real file on
// disk so the manifest identity validation accepts.
func newR6BTestBinaryIdentity(t testing.TB) V2BinaryIdentity {
	t.Helper()
	// CORRECTION06: on macOS, t.TempDir() returns /var/folders/...
	// which is symlinked to /private/var/folders/.... The
	// binary identity validation requires the path to be
	// symlink-resolved before comparison. Pre-resolve the
	// temp directory to ensure the identity path passes.
	tempDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve temp dir symlinks: %v", err)
	}
	path := filepath.Join(tempDir, "leamas-test-binary")
	data := []byte("deterministic fake leamas binary identity\n")
	if err := os.WriteFile(path, data, 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return V2BinaryIdentity{
		Path:          path,
		SHA256:        hex.EncodeToString(sum[:]),
		VCSRevision:   strings.Repeat("7", 40),
		VCSModified:   false,
		LeamasVersion: "0.1.0+test",
	}
}

// r6BStubBuildFn returns a deterministic BuildFn that the
// fallback happy-path tests use. The builder writes the
// stub binary to the request's OutputRoot path so the
// B2 BinaryAuthority matches the on-disk content.
func r6BStubBuildFn(t *testing.T) func(context.Context, ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
	t.Helper()
	return func(_ context.Context, req ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
		if err := os.MkdirAll(req.OutputRoot, 0o755); err != nil {
			return ExactSubjectBinaryResult{}, err
		}
		binaryPath := filepath.Join(req.OutputRoot, req.OutputName)
		if err := os.WriteFile(binaryPath, []byte("stub binary\n"), 0o755); err != nil {
			return ExactSubjectBinaryResult{}, err
		}
		data, err := os.ReadFile(binaryPath)
		if err != nil {
			return ExactSubjectBinaryResult{}, err
		}
		sum := sha256.Sum256(data)
		return ExactSubjectBinaryResult{
			BinaryPath:                binaryPath,
			BinarySHA256:              hex.EncodeToString(sum[:]),
			BinaryCommit:              req.SubjectCommit,
			BinaryModified:            false,
			SourceCommit:              req.SubjectCommit,
			SourceTree:                req.SubjectTree,
			SourceClean:               true,
			SourceDetached:            true,
			OutputOutsideAllWorktrees: true,
			Executable:                true,
		}, nil
	}
}

// makeFakeBinaryBuilderWithCommit returns a BuildFn that
// writes a stub binary to the request's OutputRoot so the
// B2 BinaryAuthority matches the on-disk content. The
// function is used by run-scoped tests that need independent
// binary paths per run. For "wrong B1 identity" tests,
// use makeFakeBinaryBuilderWithWrongCommit instead.
func makeFakeBinaryBuilderWithCommit(binaryPath, _ string) func(context.Context, ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
	return func(_ context.Context, req ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
		if err := os.MkdirAll(req.OutputRoot, 0o755); err != nil {
			return ExactSubjectBinaryResult{}, err
		}
		actualPath := binaryPath
		if actualPath == "" {
			actualPath = filepath.Join(req.OutputRoot, req.OutputName)
			if err := os.WriteFile(actualPath, []byte("stub binary\n"), 0o755); err != nil {
				return ExactSubjectBinaryResult{}, err
			}
		}
		data, err := os.ReadFile(actualPath)
		if err != nil {
			return ExactSubjectBinaryResult{}, err
		}
		sum := sha256.Sum256(data)
		return ExactSubjectBinaryResult{
			BinaryPath:                actualPath,
			BinarySHA256:              hex.EncodeToString(sum[:]),
			BinaryCommit:              req.SubjectCommit,
			BinaryModified:            false,
			SourceCommit:              req.SubjectCommit,
			SourceTree:                req.SubjectTree,
			SourceClean:               true,
			SourceDetached:            true,
			OutputOutsideAllWorktrees: true,
			Executable:                true,
		}, nil
	}
}

// makeFakeBinaryBuilderWithWrongCommit returns a BuildFn that
// reports a BinaryCommit explicitly different from the subject
// commit so the "wrong B1 identity" failure row can assert the
// run fails closed. The mismatch is the only purpose of this
// builder; the SourceCommit and SourceTree are pinned to the
// request.
func makeFakeBinaryBuilderWithWrongCommit(wrongCommit string) func(context.Context, ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
	return func(_ context.Context, req ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
		if err := os.MkdirAll(req.OutputRoot, 0o755); err != nil {
			return ExactSubjectBinaryResult{}, err
		}
		binaryPath := filepath.Join(req.OutputRoot, req.OutputName)
		if err := os.WriteFile(binaryPath, []byte("wrong binary\n"), 0o755); err != nil {
			return ExactSubjectBinaryResult{}, err
		}
		data, err := os.ReadFile(binaryPath)
		if err != nil {
			return ExactSubjectBinaryResult{}, err
		}
		sum := sha256.Sum256(data)
		return ExactSubjectBinaryResult{
			BinaryPath:                binaryPath,
			BinarySHA256:              hex.EncodeToString(sum[:]),
			BinaryCommit:              wrongCommit,
			BinaryModified:            false,
			SourceCommit:              req.SubjectCommit,
			SourceTree:                req.SubjectTree,
			SourceClean:               true,
			SourceDetached:            true,
			OutputOutsideAllWorktrees: true,
			Executable:                true,
		}, nil
	}
}

// makeFakeBinaryBuilderWithUnsafeOutput returns a BuildFn
// that fails with the "output root" error so the
// "unsafe OutputRoot" failure row can assert the run fails
// closed.
func makeFakeBinaryBuilderWithUnsafeOutput() func(context.Context, ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
	return func(_ context.Context, _ ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
		return ExactSubjectBinaryResult{}, os.ErrPermission
	}
}

// r6BRecordingRunner is the deterministic test double for
// the GateCollector.CommandRunner. The runner records every
// invocation (argv[0] + cwd) and lets the test inject
// failure modes (spawn, timeout, nonzero, stdout/stderr
// truncation). stdoutField / stderrField let focused
// production tests override the precise bytes returned so
// the writer flag and the bytes both reach the gate
// capture for the AUTHENTIC truncation propagation.
//
// The runner maintains an actual mutex-guarded counter
// (not a constant return) so Phase 21 of the ACT can
// prove the underlying runner was invoked exactly once.
//
// CORRECTION08 lifecycle contract (must match OsRunner):
//
//   - spawnFail    -> StartErr != nil, Err == nil
//   - timeOut      -> StartErr == nil, TimedOut = true,
//     Err == nil (NOT context.DeadlineExceeded
//     — that would be a fake command-start
//     error and is explicitly forbidden)
//   - nonZero      -> StartErr == nil, Err != nil (wait error),
//     ExitCode != 0
//   - stdoutTrunc  -> StartErr == nil
//   - stderrTrunc  -> StartErr == nil
//
// TestR6BFakeRunnerMatchesProcessLifecycleContract asserts
// the parity invariant.
type r6BRecordingRunner struct {
	mu          sync.Mutex
	argv        []string
	cwd         string
	calls       int
	spawnFail   bool
	timeOut     bool
	nonZero     bool
	stdoutTrunc bool
	stderrTrunc bool
	stdoutField []byte
	stderrField []byte
}

// Calls returns the number of underlying runner
// invocations recorded so far. The counter is incremented
// on every Run call so tests can assert the production
// integration invoked the runner exactly once.
func (r *r6BRecordingRunner) Calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// runStartError is the deterministic fake-start-error the
// r6BRecordingRunner returns when spawnFail is set. It is a
// plain os.PathError so test assertions can inspect it
// without depending on context.DeadlineExceeded (which is
// explicitly forbidden as a fake command-start error per
// Phase 10).
var runStartError = &fsPathErrorStub{msg: "exec: \"r6b-fake-binary\": file does not exist"}

// fsPathErrorStub is the minimal os.PathError-shaped stub
// the r6BRecordingRunner uses for simulated start failures.
// The stub satisfies the error interface and produces the
// same Error() shape an os/exec spawn failure would.
type fsPathErrorStub struct {
	msg string
}

func (e *fsPathErrorStub) Error() string { return e.msg }

func (r *r6BRecordingRunner) Run(ctx context.Context, name string, args []string, dir string, env []string) evidence.CommandResult {
	r.mu.Lock()
	r.argv = append([]string{name}, args...)
	r.cwd = dir
	r.calls++
	r.mu.Unlock()

	// CORRECTION08: spawn failure is signalled via StartErr
	// only. Err stays nil so the GateCollector cannot
	// confuse this with a post-start wait outcome.
	if r.spawnFail {
		return evidence.CommandResult{
			StartErr: runStartError,
			Err:      nil,
			ExitCode: 127,
			Stderr:   []byte(runStartError.Error()),
		}
	}
	// CORRECTION08: timeout is a post-start wait outcome;
	// StartErr stays nil. The fake MUST NOT use
	// context.DeadlineExceeded as a fake command-start
	// error (Phase 10 of CORRECTION08).
	if r.timeOut {
		return evidence.CommandResult{
			StartErr: nil,
			Err:      nil,
			ExitCode: 124,
			TimedOut: true,
			Stderr:   []byte("timeout"),
		}
	}
	stdout := []byte("EXEC_GATE_OBSERVED_STATUS:OK\n")
	stderr := []byte{}
	if r.stdoutField != nil {
		stdout = r.stdoutField
	}
	if r.stderrField != nil {
		stderr = r.stderrField
	}
	stdoutTrunc := r.stdoutTrunc
	stderrTrunc := r.stderrTrunc
	if r.nonZero {
		// CORRECTION08: nonzero exit is a post-start wait
		// outcome. StartErr stays nil; Err carries the
		// canonical *exec.ExitError-shaped wait error.
		return evidence.CommandResult{
			StartErr:    nil,
			Err:         &execExitErrorStub{code: 1},
			ExitCode:    1,
			Stdout:      stdout,
			Stderr:      stderr,
			StdoutTrunc: stdoutTrunc,
			StderrTrunc: stderrTrunc,
		}
	}
	return evidence.CommandResult{
		StartErr:    nil,
		Err:         nil,
		ExitCode:    0,
		Stdout:      stdout,
		Stderr:      stderr,
		StdoutTrunc: stdoutTrunc,
		StderrTrunc: stderrTrunc,
	}
}

// execExitErrorStub is the minimal exec.ExitError-shaped
// stub the r6BRecordingRunner uses for the nonzero-exit
// wait outcome. The stub satisfies the error interface and
// reports the supplied exit code so tests can assert the
// canonical wait-error shape.
type execExitErrorStub struct {
	code int
}

func (e *execExitErrorStub) Error() string {
	return "exit status " + intToString(e.code)
}

// intToString is a tiny no-import helper for stable int
// formatting. Kept package-private to avoid pulling strconv
// into a test file that does not need it elsewhere.
func intToString(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
