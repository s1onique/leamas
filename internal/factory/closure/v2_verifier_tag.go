// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_tag.go implements Phase 4 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-CLI-TAG-STATE01:
//
// the optional annotated-tag assertion that ties a verifier
// invocation to a closure-protocol tag that targets the
// externally supplied closure commit C.
//
// The tag path is purely optional:
//
//   - When the caller does not set --expected-tag, the
//     verifier MUST NOT touch refs and MUST emit no tag
//     diagnostics.
//   - When the caller supplies --expected-tag, the verifier
//     MUST assert that an annotated tag with that exact
//     name exists in the target repository, that the tag is
//     annotated (not lightweight), and that the tag targets
//     the closure commit C.
//
// Lightweight tags are rejected; signature verification is
// out of scope per ACT 4 (the ACT text says: "Signature
// verification is out of scope unless already mandated by
// current Leamas doctrine").
//
// All tag operations run through the bound git authority
// (the same authority used for topology) so the process CWD
// never influences the lookup. The tag inspection never
// creates, deletes, or moves refs.

import (
	"context"
	"fmt"
	"strings"
)

// V2VerifierTagAssertion is the structured verdict for the
// optional tag-assertion phase. The fields are populated by
// ResolveV2ClosureTagAssertion; tests assert on the verdict
// fields rather than on the raw diagnostic list.
//
// Fields are zero-valued when no tag was expected. The
// verifier MUST NOT touch refs when Expected is empty.
type V2VerifierTagAssertion struct {
	Expected    bool
	Found       bool
	Annotated   bool
	Target      string
	Diagnostics V2VerifierDiagnostics
}

// ResolveV2ClosureTagAssertion inspects the optional
// --expected-tag name and binds it to the supplied closure
// commit C. The function is fail-closed: any failure to
// classify the tag emits a typed diagnostic.
//
// Classification contract:
//
//   - Expected empty       -> Expected=false, Found=false,
//     Annotated=false, Target="",
//     Diagnostics=nil (no Git lookup).
//   - ref missing          -> closure_tag_missing
//   - ref is lightweight   -> closure_tag_lightweight
//   - target != C          -> closure_tag_target_mismatch
//   - annotated but tag is not Git-parseable
//     -> closure_tag_unreadable
//
// Success only when the ref resolves, the tag object is
// annotated, and the OID the tag points at equals the
// externally supplied closure commit C.
//
// The function never modifies refs; it only reads
// `git rev-parse --verify --end-of-options <name>` and
// `git cat-file -t <name>` and (for annotated tags)
// `git rev-parse --verify <annotated-oid>^{commit}`.
func ResolveV2ClosureTagAssertion(
	ctx context.Context,
	authority V2ClosureGitAuthority,
	expectedTag string,
	closureCommit string,
) V2VerifierTagAssertion {
	if strings.TrimSpace(expectedTag) == "" {
		return V2VerifierTagAssertion{Expected: false}
	}
	if authority == nil {
		return V2VerifierTagAssertion{
			Expected: true,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierClosureTagUnreadable,
				"git authority is required for tag assertion",
			).withProperty("expected_tag")},
		}
	}
	if strings.TrimSpace(closureCommit) == "" {
		return V2VerifierTagAssertion{
			Expected: true,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierClosureTagTargetMismatch,
				"closure commit is empty",
			).withProperty("expected_tag")},
		}
	}

	// Step 1: resolve the ref to its object OID. A missing
	// ref or non-tag object produces closure_tag_missing.
	oidResult := authorityRunGitRevParse(ctx, authority, "--verify", "--end-of-options", expectedTag)
	if oidResult.exitCode != 0 {
		return V2VerifierTagAssertion{
			Expected: true,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierClosureTagMissing,
				"tag "+stringOrQuote(expectedTag)+" does not resolve",
			).withProperty("expected_tag")},
		}
	}
	oid := strings.TrimSpace(string(oidResult.stdout))
	if !oidPattern.MatchString(oid) {
		return V2VerifierTagAssertion{
			Expected: true,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierClosureTagUnreadable,
				"tag "+stringOrQuote(expectedTag)+" did not resolve to a 40-char OID",
			).withProperty("expected_tag").withObserved(oid)},
		}
	}

	// Step 2: classify the object type via cat-file -t.
	typeResult := authorityRunGitCatFileT(ctx, authority, oid)
	typeName := strings.TrimSpace(string(typeResult.stdout))
	if typeResult.exitCode != 0 || typeName != "tag" {
		return V2VerifierTagAssertion{
			Expected:  true,
			Found:     true,
			Target:    oid,
			Annotated: false,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierClosureTagLightweight,
				"tag "+stringOrQuote(expectedTag)+" is not an annotated tag object",
			).withProperty("expected_tag").withObjectOID(oid)},
		}
	}

	// Step 3: dereference the tag object to the underlying
	// commit. Annotated tags carry an object pointer in
	// their header; <oid>^{commit} dereferences through
	// tag -> commit (or tag -> tag -> commit).
	targetResult := authorityRunGitRevParse(ctx, authority, "--verify", "--end-of-options", oid+"^{commit}")
	if targetResult.exitCode != 0 {
		return V2VerifierTagAssertion{
			Expected:  true,
			Found:     true,
			Annotated: true,
			Target:    oid,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierClosureTagUnreadable,
				"annotated tag "+stringOrQuote(expectedTag)+" does not dereference to a commit",
			).withProperty("expected_tag").withObjectOID(oid)},
		}
	}
	target := strings.TrimSpace(string(targetResult.stdout))
	if target != closureCommit {
		return V2VerifierTagAssertion{
			Expected:  true,
			Found:     true,
			Annotated: true,
			Target:    target,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierClosureTagTargetMismatch,
				"annotated tag "+stringOrQuote(expectedTag)+" targets "+trunc8ForDiag(target)+
					", expected "+trunc8ForDiag(closureCommit),
			).withProperty("expected_tag").withExpected(closureCommit).withObjectOID(oid)},
		}
	}

	return V2VerifierTagAssertion{
		Expected:  true,
		Found:     true,
		Annotated: true,
		Target:    target,
	}
}

// authorityGitValue is the transport shape returned by the
// private helpers below. The struct keeps the four fields
// ACT 4 needs (stdout bytes, stderr bytes, exit code, err)
// without leaking the package-private gitCommandResult
// type across the helper boundary.
type authorityGitValue struct {
	stdout   []byte
	stderr   []byte
	exitCode int
	err      error
}

// errAsError reports the supplied command error, returning
// nil when no error was recorded. The helper exists so the
// public read paths do not depend on the package-private
// gitCommandResult type while still distinguishing spawn
// failure from non-zero exit code.
func (v authorityGitValue) errAsError() error {
	return v.err
}

// authorityRunGitRevParse runs `git rev-parse` against the
// bound authority. The helper passes every argument through
// the same end-of-options flag handling the topology
// resolver uses, so a path or ref beginning with "-" cannot
// be interpreted as a flag.
func authorityRunGitRevParse(ctx context.Context, authority V2ClosureGitAuthority, args ...string) authorityGitValue {
	return authorityRunGit(ctx, authority, "rev-parse", args...)
}

// authorityRunGitCatFileT runs `git cat-file -t` against
// the bound authority.
func authorityRunGitCatFileT(ctx context.Context, authority V2ClosureGitAuthority, oid string) authorityGitValue {
	return authorityRunGit(ctx, authority, "cat-file", "-t", oid)
}

// authorityRunGitCatFileTag runs `git cat-file tag <oid>`
// against the bound authority. The helper is the single
// read path for raw annotated-tag object bytes used by
// the metadata parser (Phase 5 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C).
func authorityRunGitCatFileTag(ctx context.Context, authority V2ClosureGitAuthority, oid string) ([]byte, error) {
	result := authorityRunGit(ctx, authority, "cat-file", "tag", oid)
	if result.exitCode != 0 || result.errAsError() != nil {
		return nil, fmt.Errorf("git cat-file tag %s failed", oid)
	}
	return append([]byte(nil), result.stdout...), nil
}

// authorityRunGit is the single helper used by both
// rev-parse and cat-file calls. It routes the call through
// the bound git client so process CWD is never used as
// authority. The helper returns the raw value; the call
// site is responsible for translating exit non-zero into
// the right typed diagnostic.
//
// The helper widens the access by type-asserting the
// production concrete type. If the authority is not the
// production type (for example a test double), we surface
// a synthetic non-zero exit so the verifier reports the
// missing interface explicitly.
func authorityRunGit(ctx context.Context, authority V2ClosureGitAuthority, command string, args ...string) authorityGitValue {
	if runner, ok := authority.(*v2ClosureGitAuthority); ok {
		allArgs := append([]string{command}, args...)
		result := runner.git.Run(ctx, runner.repoRoot, allArgs...)
		return authorityGitValue{
			stdout:   result.Stdout,
			stderr:   result.Stderr,
			exitCode: result.ExitCode,
		}
	}
	return authorityGitValue{
		stdout:   []byte{},
		stderr:   []byte("v2 verifier authority does not expose a git runner"),
		exitCode: 1,
	}
}

// trunc8ForDiag returns the first 8 hex chars of oid-or-hex
// for inclusion in diagnostic messages. It returns the full
// string for inputs shorter than 8 chars so non-OID values
// still render sensibly.
func trunc8ForDiag(oid string) string {
	if len(oid) < 8 {
		return oid
	}
	return oid[:8]
}
