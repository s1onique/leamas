// SPDX-License-Identifier: Apache-2.0

package closure

// closure_exact_subject_binary_build.go implements the
// per-stage build + canonical-identity sequence executed
// by BuildExactSubjectBinary after the source authority is
// proven and before the cleanup junction.
//
// This file is intentionally narrow: it owns ONLY the
// bounded build invocation, the post-build source
// verification, the binary stat + SHA-256, the canonical
// identity read, and the auxiliary native buildinfo read.
//
// Cleanup, inventory, and the call-site junction remain in
// closure_exact_subject_binary.go so each file owns one
// responsibility.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

// exactBinarySubjectEnv returns the environment slice passed
// to the build + identity subprocesses. The slice is the
// process environment with the LEAMAS_EXEC_* re-entry
// markers stripped: those markers cause the produced
// binary's `main()` to abort with a re-entry fuse error
// when invoked inside the factory test harness, which is
// itself a Leamas execution. We append CGO_ENABLED=0 so the
// build uses the static-binary policy the rest of the
// repository enforces.
//
// The LEAMAS_EXEC_* keys are added explicitly with empty
// values so the executor's mergeEnvironment helper applies
// them as overrides for any inherited values: an empty
// value satisfies the produced binary's
// `if os.Getenv(EnvRootID) != ""` re-entry check, which is
// the gate the binary's `main()` runs at startup.
func exactBinarySubjectEnv() []string {
	env := make([]string, 0, len(os.Environ())+4)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "LEAMAS_EXEC_") {
			continue
		}
		env = append(env, entry)
	}
	env = append(env,
		"LEAMAS_EXEC_ROOT_ID=",
		"LEAMAS_EXEC_PARENT_PID=",
		"LEAMAS_EXEC_GENERATION=",
		"CGO_ENABLED=0",
	)
	return env
}

// runExactBinaryBuildAndCleanup performs the
// source-verify / build / identity-read sequence and returns
// the typed result + primary error. Cleanup is performed by
// the caller via the junction pattern; this function never
// runs git worktree remove or git worktree prune.
//
// On every internal failure this function returns a partial
// ExactSubjectBinaryResult carrying the per-stage bounded
// proof so the caller can populate the cleanup proof
// fields. The primary error is returned separately.
func runExactBinaryBuildAndCleanup(
	ctx context.Context,
	git gitClient,
	_ *exactBinaryCleanup,
	req ExactSubjectBinaryRequest,
	absOutputRoot, outputName, buildWorktreePath string,
) (ExactSubjectBinaryResult, error) {
	// Verify source BEFORE build.
	if err := exactBinaryVerifySource(ctx, git, buildWorktreePath, req.SubjectCommit, req.SubjectTree); err != nil {
		return ExactSubjectBinaryResult{}, err
	}
	binaryPath := filepath.Join(absOutputRoot, outputName)

	// Build the real Leamas binary using the existing
	// production LDFLAGS mechanism. We inject the EXACT
	// subject commit so the binary's own version output
	// reports it as the canonical authority. cmd/go's VCS
	// stamping is NOT relied on: linked worktrees do not
	// stamp reliably, so the canonical authority MUST be
	// the linker-injected value.
	ldflags := exactBinaryBuildLDFlags(req.SubjectCommit)

	buildEx, err := execution.NewExecutor(exactBinaryBuildBudget(), nil)
	if err != nil {
		return ExactSubjectBinaryResult{}, fmt.Errorf("exact-binary: create build executor: %w", err)
	}
	defer buildEx.Close()
	buildResult := buildEx.Execute(ctx, &execution.Request{
		Name: "exact-binary build",
		Args: []string{"go", "build", "-trimpath", "-ldflags", ldflags, "-o", binaryPath, "./cmd/leamas"},
		Dir:  buildWorktreePath,
		Env:  exactBinarySubjectEnv(),
		// Timeout is governed by exactBinaryBuildBudget() so
		// the umbrella test sees the production bounded
		// shape. OutputCap matches the canonical helper
		// below.
		Timeout:   10 * time.Minute,
		OutputCap: 64 * 1024,
	})

	// Verify the source state did NOT drift during the
	// build. The check uses the same canonical predicates
	// as the before-build check.
	if err := exactBinaryVerifySourceAfterBuild(ctx, git, buildWorktreePath, req.SubjectCommit, req.SubjectTree); err != nil {
		return ExactSubjectBinaryResult{}, err
	}

	// Bounded result predicate.
	buildBounded := buildResult.Error == nil && buildResult.ExitCode == 0 &&
		!buildResult.OutputTruncated && !buildResult.OutputIncomplete
	buildErrCode := ""
	if buildResult.Error != nil {
		buildErrCode = buildResult.Error.Code
	}

	// Stat + SHA-256 of the binary at rest.
	binaryInfo, statErr := os.Stat(binaryPath)
	if statErr != nil {
		return ExactSubjectBinaryResult{
			BuildBounded:   buildBounded,
			BuildErrorCode: buildErrCode,
			SourceCommit:   req.SubjectCommit,
			SourceTree:     req.SubjectTree,
			SourceClean:    true,
			SourceDetached: true,
			BinaryPath:     binaryPath,
		}, fmt.Errorf("exact-binary: stat binary: %w", statErr)
	}
	if binaryInfo.Size() == 0 {
		return ExactSubjectBinaryResult{}, errors.New("exact-binary: binary is empty")
	}
	executable := binaryInfo.Mode()&0o111 != 0
	if !executable {
		return ExactSubjectBinaryResult{}, errors.New("exact-binary: binary is not executable")
	}
	sha256sum, err := hashBinaryAtRest(binaryPath)
	if err != nil {
		return ExactSubjectBinaryResult{}, fmt.Errorf("exact-binary: hash binary: %w", err)
	}

	// Canonical identity: invoke the produced binary's
	// own `version --json` surface. This is the same
	// surface the production release artefacts expose; the
	// B1 authority MUST NOT introduce a second version
	// scheme.
	identity, identityResult, err := exactBinaryReadIdentity(ctx, binaryPath)
	if err != nil {
		return ExactSubjectBinaryResult{
			BuildBounded:   buildBounded,
			BuildErrorCode: buildErrCode,
			SourceCommit:   req.SubjectCommit,
			SourceTree:     req.SubjectTree,
			SourceClean:    true,
			SourceDetached: true,
			BinaryPath:     binaryPath,
			BinarySHA256:   sha256sum,
			Executable:     executable,
		}, fmt.Errorf("exact-binary: %w", err)
	}
	identityBounded := identityResult != nil &&
		identityResult.Error == nil && identityResult.ExitCode == 0 &&
		!identityResult.OutputTruncated && !identityResult.OutputIncomplete
	identityErrCode := ""
	if identityResult != nil && identityResult.Error != nil {
		identityErrCode = identityResult.Error.Code
	}

	// Auxiliary native buildinfo. ABSENCE MUST NOT FAIL.
	// The error is captured for diagnostics but the
	// authority predicates only check the canonical
	// identity.
	native, _ := exactBinaryReadNativeBuildInfo(ctx, binaryPath)

	// Canonical identity predicates.
	if identity.Commit != req.SubjectCommit {
		return exactBinaryResultWithIdentity(
			buildBounded, buildErrCode, identityBounded, identityErrCode,
			req, binaryPath, sha256sum, identity, executable, native,
		), fmt.Errorf("exact-binary: binary commit %s != subject %s", identity.Commit, req.SubjectCommit)
	}
	if identity.Modified {
		res := exactBinaryResultWithIdentity(
			buildBounded, buildErrCode, identityBounded, identityErrCode,
			req, binaryPath, sha256sum, identity, executable, native,
		)
		res.BinaryModified = true
		return res, errors.New("exact-binary: binary reports modified=true (source is dirty)")
	}

	return exactBinaryResultWithIdentity(
		buildBounded, buildErrCode, identityBounded, identityErrCode,
		req, binaryPath, sha256sum, identity, executable, native,
	), nil
}

// exactBinaryResultWithIdentity is the small helper that
// packs the per-stage bounded + identity fields into the
// final ExactSubjectBinaryResult. Centralising the
// construction here keeps the build sequence readable.
func exactBinaryResultWithIdentity(
	buildBounded bool,
	buildErrCode string,
	identityBounded bool,
	identityErrCode string,
	req ExactSubjectBinaryRequest,
	binaryPath string,
	sha256sum string,
	identity exactBinaryIdentity,
	executable bool,
	native exactBinaryNativeBuildInfo,
) ExactSubjectBinaryResult {
	return ExactSubjectBinaryResult{
		BuildBounded:              buildBounded,
		BuildErrorCode:            buildErrCode,
		IdentityBounded:           identityBounded,
		IdentityErrorCode:         identityErrCode,
		SourceCommit:              req.SubjectCommit,
		SourceTree:                req.SubjectTree,
		SourceClean:               true,
		SourceDetached:            true,
		BinaryPath:                binaryPath,
		BinarySHA256:              sha256sum,
		BinaryCommit:              identity.Commit,
		BinaryModified:            identity.Modified,
		Executable:                executable,
		OutputOutsideAllWorktrees: true,
		NativeVCSRevision:         native.Revision,
		NativeVCSRevisionPresent:  native.RevisionPresent,
		NativeVCSModified:         native.Modified,
		NativeVCSModifiedPresent:  native.ModifiedPresent,
	}
}

// exactBinaryBuildLDFlags builds the canonical LDFLAGS
// string used to inject the EXACT subject commit and a
// deterministic build timestamp into the binary.
//
// The flag set is intentionally identical to the production
// Makefile shape (see internal/version/version.go for the
// package-level variables the linker addresses):
//
//	-X '<module>/internal/version.Version=0.1.0'
//	-X '<module>/internal/version.DeclaredVersion=0.1.0'
//	-X '<module>/internal/version.Commit=<S>'
//	-X '<module>/internal/version.BuildTime=<RFC3339>'
//
// `0.1.0` is the placeholder SemVer used by the canonical
// identity surface; the binary's `version` CLI reports the
// same SemVer. Real release artefacts override VERSION
// at link time.
func exactBinaryBuildLDFlags(subjectCommit string) string {
	return strings.Join([]string{
		fmt.Sprintf("-X '%s/internal/version.Version=%s'", leamasModulePath, leamasBuildVersion),
		fmt.Sprintf("-X '%s/internal/version.DeclaredVersion=%s'", leamasModulePath, leamasBuildVersion),
		fmt.Sprintf("-X '%s/internal/version.Commit=%s'", leamasModulePath, subjectCommit),
		fmt.Sprintf("-X '%s/internal/version.BuildTime=%s'", leamasModulePath, leamasBuildTime()),
	}, " ")
}