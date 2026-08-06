// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_authority.go defines the repository-bound Git
// authority required by ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01.
//
// The interface is the only path the v2 verifier uses to
// observe Git objects. Every operation is permanently bound
// to RepositoryRoot at construction; the resolver never uses
// the process CWD as authority.
//
// Every Git observation is bounded:
//
//   - finite timeout (DefaultGitTimeout)
//   - bounded stdout and stderr (DefaultOutputLimit)
//   - WaitDelay cleanup bound (DefaultGitWaitDelay)
//   - real exit-code extraction
//   - typed timeout, cancellation, output overflow,
//     and spawn-failure errors
//
// The interface is intentionally minimal: ACT 2 (topology) and
// ACT 3 (manifest) build on top of these primitives without
// adding new method signatures.

import (
	"context"
	"strings"
)

// V2ClosureGitAuthority is the Git-observation interface the
// v2 closure verifier uses. Every method is bound to the
// resolver's RepositoryRoot at construction; the resolver
// never reads the process CWD.
//
// All methods accept a context. Implementations must honour
// context cancellation, surface bounded-execution failures
// as typed diagnostics, and never panic.
type V2ClosureGitAuthority interface {
	// ObjectFormat returns the Git object storage format
	// reported by `git rev-parse --show-object-format`.
	// The verifier uses this to enforce the SHA-1 policy
	// before any OID validation. Returns the trimmed
	// format string (e.g. "sha1") on success.
	ObjectFormat(ctx context.Context) (string, error)

	// ResolveCommit resolves the supplied revision to a
	// full commit OID via `git rev-parse --verify
	// <rev>^{commit}`. Returns the resolved OID (40-char
	// lowercase hex) on success.
	ResolveCommit(ctx context.Context, revision string) (string, error)

	// ResolveTree resolves the tree OID of the supplied
	// commit via `git rev-parse --verify <commit>^{tree}`.
	// Returns the resolved tree OID on success.
	ResolveTree(ctx context.Context, commit string) (string, error)

	// IsAncestor reports whether the supplied ancestor
	// commit is reachable from the supplied descendant
	// commit according to `git merge-base --is-ancestor`.
	//
	// Exit code classification per Git's contract:
	//   - 0: ancestor=true (no error)
	//   - 1: ancestor=false (no error)
	//   - other: observation failure (typed error)
	IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error)

	// ResolvePathObject resolves a repository-relative
	// path at the supplied commit to its Git object OID
	// and type. Implementation calls `git rev-parse
	// --verify <commit>:<path>` for the OID and `git
	// cat-file -t <oid>` for the type. Returns the OID
	// and one of "blob" / "tree" / "commit" / "tag" on
	// success.
	ResolvePathObject(ctx context.Context, commit, path string) (oid string, objectType string, err error)

	// ReadBlob returns the exact raw blob bytes for the
	// supplied OID via `git cat-file blob <oid>`. The
	// returned slice is NEVER trimmed: trailing newlines,
	// leading whitespace, and trailing spaces are
	// preserved so SHA-256(raw) equals the manifest's
	// plan_sha256 / committed blob digest.
	ReadBlob(ctx context.Context, oid string) ([]byte, error)
}

// v2ClosureGitAuthority is the production implementation of
// V2ClosureGitAuthority. The struct is permanently bound to a
// single repository root at construction; every method
// routes to that root regardless of the caller's CWD.
//
// The struct embeds the package-private bounded git client
// (gitClient). Production wiring defaults to RealGit{} when
// the caller supplies nil; tests inject recording or
// deterministic fakes via WithV2ClosureGitAuthorityClient.
type v2ClosureGitAuthority struct {
	git      gitClient
	repoRoot string
}

// NewV2ClosureGitAuthority constructs a production
// repository-bound git authority. It refuses:
//
//   - empty repoRoot (typed error)
//   - nil git client (defaults to RealGit{})
//
// The returned authority has no observable side effects: it
// does not touch the repository until a method is invoked.
// The first method invocation that exercises ObjectFormat
// surfaces any Git-availability failure as a typed
// repository_unavailable diagnostic.
func NewV2ClosureGitAuthority(repoRoot string) (V2ClosureGitAuthority, error) {
	return newV2ClosureGitAuthorityWithClient(RealGit{}, repoRoot)
}

// newV2ClosureGitAuthorityWithClient is the test seam used
// by the foundation ACT's matrix tests. It exists alongside
// NewV2ClosureGitAuthority so the production constructor
// keeps a RealGit{} default without complicating the
// production call sites.
func newV2ClosureGitAuthorityWithClient(git gitClient, repoRoot string) (V2ClosureGitAuthority, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return nil, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierRepositoryUnavailable,
			"repository_root is required",
		))
	}
	if git == nil {
		git = RealGit{}
	}
	return &v2ClosureGitAuthority{git: git, repoRoot: repoRoot}, nil
}

// ObjectFormat executes `git rev-parse --show-object-format`
// against the resolver's bound repoRoot. The function never
// uses the process CWD.
func (a *v2ClosureGitAuthority) ObjectFormat(ctx context.Context) (string, error) {
	if a == nil {
		return "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierObjectFormatUnavailable,
			"git authority is nil",
		))
	}
	result := a.git.Run(ctx, a.repoRoot, "rev-parse", "--show-object-format")
	if result.Err != nil || result.ExitCode != 0 {
		return "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierObjectFormatUnavailable,
			"git rev-parse --show-object-format failed",
		))
	}
	format := strings.TrimSpace(string(result.Stdout))
	if format == "" {
		return "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierObjectFormatUnavailable,
			"object format observation returned empty value",
		))
	}
	return format, nil
}

// ResolveCommit resolves the supplied revision to its full
// commit OID. The resolver uses --end-of-options so a
// revision beginning with "-" cannot be interpreted as a flag.
func (a *v2ClosureGitAuthority) ResolveCommit(ctx context.Context, revision string) (string, error) {
	if a == nil {
		return "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierSubjectUnresolved,
			"git authority is nil",
		))
	}
	if strings.TrimSpace(revision) == "" {
		return "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierSubjectUnresolved,
			"revision is empty",
		))
	}
	result := a.git.Run(ctx, a.repoRoot, "rev-parse", "--verify", "--end-of-options", revision+"^{commit}")
	if result.Err != nil || result.ExitCode != 0 {
		return "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierSubjectUnresolved,
			"git rev-parse --verify "+revision+"^{commit} failed",
		).withObserved(revision))
	}
	value := strings.TrimSpace(string(result.Stdout))
	if !oidPattern.MatchString(value) {
		return "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierSubjectUnresolved,
			"resolved revision is not a 40-char OID",
		).withObserved(value))
	}
	return value, nil
}

// ResolveTree resolves the tree OID for the supplied commit.
// The resolver uses --end-of-options so the commit argument
// cannot be interpreted as a flag.
func (a *v2ClosureGitAuthority) ResolveTree(ctx context.Context, commit string) (string, error) {
	if a == nil {
		return "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierTopologyObservationFailed,
			"git authority is nil",
		))
	}
	if strings.TrimSpace(commit) == "" {
		return "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierTopologyObservationFailed,
			"commit is empty",
		))
	}
	result := a.git.Run(ctx, a.repoRoot, "rev-parse", "--verify", "--end-of-options", commit+"^{tree}")
	if result.Err != nil || result.ExitCode != 0 {
		return "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierTopologyObservationFailed,
			"git rev-parse --verify "+commit+"^{tree} failed",
		).withObserved(commit))
	}
	value := strings.TrimSpace(string(result.Stdout))
	if !oidPattern.MatchString(value) {
		return "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierTopologyObservationFailed,
			"resolved tree is not a 40-char OID",
		).withObserved(value))
	}
	return value, nil
}

// IsAncestor classifies the supplied pair against Git's
// documented exit-code contract:
//
//	0 -> ancestor=true
//	1 -> ancestor=false
//	other -> typed observation failure
func (a *v2ClosureGitAuthority) IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error) {
	if a == nil {
		return false, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierTopologyObservationFailed,
			"git authority is nil",
		))
	}
	if strings.TrimSpace(ancestor) == "" {
		return false, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierTopologyObservationFailed,
			"ancestor is empty",
		))
	}
	if strings.TrimSpace(descendant) == "" {
		return false, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierTopologyObservationFailed,
			"descendant is empty",
		))
	}
	result := a.git.Run(ctx, a.repoRoot, "merge-base", "--is-ancestor", ancestor, descendant)
	if result.ExitCode == 0 {
		return true, nil
	}
	if result.ExitCode == 1 {
		return false, nil
	}
	return false, NewV2VerifierError(NewV2VerifierDiagnostic(
		V2VerifierTopologyObservationFailed,
		"git merge-base --is-ancestor observation failed",
	))
}

// ResolvePathObject resolves a repository-relative path at the
// supplied commit. The resolver first asks git for the OID
// (using --end-of-options so the path cannot be confused with
// a flag), then asks for the object type. Type lookup runs
// only after the OID resolves successfully so a missing path
// never wastes a `cat-file -t` round-trip.
func (a *v2ClosureGitAuthority) ResolvePathObject(ctx context.Context, commit, path string) (string, string, error) {
	if a == nil {
		return "", "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanMissing,
			"git authority is nil",
		))
	}
	if strings.TrimSpace(commit) == "" {
		return "", "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanMissing,
			"commit is empty",
		))
	}
	if strings.TrimSpace(path) == "" {
		return "", "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanMissing,
			"path is empty",
		))
	}
	spec := commit + ":" + path
	oidResult := a.git.Run(ctx, a.repoRoot, "rev-parse", "--verify", "--end-of-options", spec)
	if oidResult.Err != nil || oidResult.ExitCode != 0 {
		return "", "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanMissing,
			"git rev-parse --verify "+spec+" failed",
		).withObjectPath(path))
	}
	oid := strings.TrimSpace(string(oidResult.Stdout))
	if !oidPattern.MatchString(oid) {
		return "", "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanMissing,
			"resolved object is not a 40-char OID",
		).withObjectPath(path).withObserved(oid))
	}
	typeResult := a.git.Run(ctx, a.repoRoot, "cat-file", "-t", oid)
	if typeResult.Err != nil || typeResult.ExitCode != 0 {
		return "", "", NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanMissing,
			"git cat-file -t "+oid+" failed",
		).withObjectPath(path).withObjectOID(oid))
	}
	objectType := strings.TrimSpace(string(typeResult.Stdout))
	return oid, objectType, nil
}

// ReadBlob returns the literal blob bytes via `git cat-file
// blob <oid>`. The function never trims the result: trailing
// newlines, leading whitespace, and trailing spaces are
// preserved so SHA-256(bytes) equals the manifest's binding.
//
// The function uses the type-explicit `cat-file blob` form (not
// `cat-file -p`) so the output is the raw uncompressed blob
// contents with no pretty-printer modifications.
func (a *v2ClosureGitAuthority) ReadBlob(ctx context.Context, oid string) ([]byte, error) {
	if a == nil {
		return nil, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanReadFailed,
			"git authority is nil",
		))
	}
	if strings.TrimSpace(oid) == "" {
		return nil, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanReadFailed,
			"blob OID is empty",
		))
	}
	result := a.git.Run(ctx, a.repoRoot, "cat-file", "blob", oid)
	if result.Err != nil || result.ExitCode != 0 {
		return nil, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanReadFailed,
			"git cat-file blob failed",
		).withObjectOID(oid))
	}
	// Do NOT TrimSpace. SHA-256 of the literal bytes must
	// equal the manifest's binding hash; trimming would
	// silently strip a trailing newline and corrupt the
	// byte-authority path.
	return append([]byte(nil), result.Stdout...), nil
}

// withObjectPath attaches the path anchor to a diagnostic.
func (d V2VerifierDiagnostic) withObjectPath(path string) V2VerifierDiagnostic {
	d.ObjectPath = path
	return d
}

// withObjectOID attaches the OID anchor to a diagnostic.
func (d V2VerifierDiagnostic) withObjectOID(oid string) V2VerifierDiagnostic {
	d.ObjectOID = oid
	return d
}

// withObserved attaches the observed value to a diagnostic.
func (d V2VerifierDiagnostic) withObserved(value string) V2VerifierDiagnostic {
	d.Observed = value
	return d
}
