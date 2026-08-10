// SPDX-License-Identifier: Apache-2.0

// binary_gate_testhelpers_test.go owns the small test seams
// the R6-B umbrellas use to drive the production integration
// without re-implementing the B1 build or the gate runner.
// The helpers live in a _test.go file so they are not part of
// the production binary.
package closure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
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
	return filepath.Join(t.TempDir(), "leamas-binary-gate")
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
	path := filepath.Join(t.TempDir(), "leamas-test-binary")
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
			BinaryCommit:              strings.Repeat("a", 40),
			BinaryModified:            false,
			SourceCommit:              strings.Repeat("a", 40),
			SourceTree:                strings.Repeat("b", 40),
			SourceClean:               true,
			SourceDetached:            true,
			OutputOutsideAllWorktrees: true,
			Executable:                true,
		}, nil
	}
}

// makeFakeBinaryBuilderWithCommit returns a BuildFn that
// reports the supplied binary commit OID. The umbrellas use
// this to construct a B1 result whose BinaryCommit is
// explicitly different from the subject commit so the
// "wrong B1 identity" failure row can assert the run fails
// closed.
func makeFakeBinaryBuilderWithCommit(binaryPath, binaryCommit string) func(context.Context, ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
	return func(_ context.Context, _ ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
		data, err := os.ReadFile(binaryPath)
		if err != nil {
			return ExactSubjectBinaryResult{}, err
		}
		sum := sha256.Sum256(data)
		return ExactSubjectBinaryResult{
			BinaryPath:                binaryPath,
			BinarySHA256:              hex.EncodeToString(sum[:]),
			BinaryCommit:              binaryCommit,
			BinaryModified:            false,
			SourceCommit:              strings.Repeat("a", 40),
			SourceTree:                strings.Repeat("b", 40),
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
// truncation).
type r6BRecordingRunner struct {
	argv        []string
	cwd         string
	spawnFail   bool
	timeOut     bool
	nonZero     bool
	stdoutTrunc bool
	stderrTrunc bool
}

func (r *r6BRecordingRunner) Run(ctx context.Context, name string, args []string, dir string, env []string) evidence.CommandResult {
	r.argv = append([]string{name}, args...)
	r.cwd = dir
	if r.spawnFail {
		return evidence.CommandResult{
			ExitCode: 127,
			Err:      context.DeadlineExceeded,
			Stderr:   []byte("spawn failed"),
		}
	}
	if r.timeOut {
		return evidence.CommandResult{
			ExitCode: 124,
			TimedOut: true,
			Stderr:   []byte("timeout"),
		}
	}
	stdout := []byte("OK\n")
	stderr := []byte{}
	if r.stdoutTrunc {
		stdout = []byte("OK")
	}
	if r.stderrTrunc {
		stderr = []byte("warning")
	}
	if r.nonZero {
		return evidence.CommandResult{
			ExitCode: 1,
			Stdout:   []byte("FAILED\n"),
			Stderr:   stderr,
		}
	}
	return evidence.CommandResult{
		ExitCode: 0,
		Stdout:   stdout,
		Stderr:   stderr,
	}
}
