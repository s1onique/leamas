// SPDX-License-Identifier: Apache-2.0

package closure

// object_format_resolver.go implements the GitObjectResolver
// interface required by the closure manifest verifier and the
// programmatic SHA-1 policy required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2C-R4.
//
// R2C-R4-R1 design: the resolver is bound to a single
// repository root at construction time. Every call routes to
// that root regardless of the caller's CWD. ObjectFormat()
// executes `git rev-parse --show-object-format` and propagates
// every observation failure so the verifier can emit a typed
// diagnostic before it ever touches an OID. CatFile() executes
// `git cat-file -p <oid>` against the same root.
//
// The EnforceSHA1ObjectFormat helper applies the SHA-1 policy
// programmatically: the verifier rejects any repository whose
// reported storage format is unavailable, empty, or not
// exactly "sha1". The format check happens BEFORE OID
// validation so a SHA-256 repository with 64-char OIDs never
// reaches ValidateOID and never accepts a misleading failure
// from the 40-char length check.

import (
	"context"
	"fmt"
	"strings"
)

// r2crGitObjectResolver implements GitObjectResolver against
// the production RealGit client. It is bound to a single
// repository root at construction time; CatFile and
// ObjectFormat never use the caller's CWD.
type r2crGitObjectResolver struct {
	git      gitClient
	repoRoot string
}

// r2crGitRaw executes `git cat-file blob <oid>` against the
// supplied repository and returns the raw blob bytes. The
// helper deliberately bypasses runGitValue (which TrimsSpace
// the output) so the returned bytes match the literal blob
// stored in the object database. SHA-256 of these bytes
// therefore equals the literal frozen plan SHA-256.
func r2crGitRaw(ctx context.Context, git gitClient, repoRoot, oid string) ([]byte, error) {
	if git == nil {
		return nil, fmt.Errorf("git client is required")
	}
	if strings.TrimSpace(repoRoot) == "" {
		return nil, fmt.Errorf("repository root is empty")
	}
	if strings.TrimSpace(oid) == "" {
		return nil, fmt.Errorf("blob OID is empty")
	}
	result := git.Run(ctx, repoRoot, "cat-file", "blob", oid)
	if result.Err != nil || result.ExitCode != 0 {
		detail := strings.TrimSpace(string(result.Stderr))
		if detail == "" && result.Err != nil {
			detail = result.Err.Error()
		}
		return nil, fmt.Errorf("git cat-file blob %s failed (exit %d): %s",
			oid, result.ExitCode, detail)
	}
	// Do NOT TrimSpace. The byte-authority path requires
	// the literal blob bytes including any trailing newline.
	return append([]byte(nil), result.Stdout...), nil
}

// runR2CRGitRaw is the exported alias used by tests and the
// dogfood path. It exists as a stable name so test code and
// production code share the same byte-authority helper.
func runR2CRGitRaw(ctx context.Context, repoRoot, oid string) ([]byte, error) {
	return r2crGitRaw(ctx, RealGit{}, repoRoot, oid)
}

// CatFile reads a Git object via `git cat-file -p <oid>` from
// the resolver's bound repository root and returns the raw
// bytes. Production wiring uses the default RealGit client.
//
// R2C-R4-R1: this method NEVER uses the process CWD. The
// resolver is bound at construction time so callers cannot
// accidentally mix repositories.
func (r *r2crGitObjectResolver) CatFile(oid string) ([]byte, error) {
	if r == nil || r.git == nil {
		return nil, fmt.Errorf("git object resolver has no client")
	}
	if strings.TrimSpace(r.repoRoot) == "" {
		return nil, fmt.Errorf("git object resolver has no repository root")
	}
	if strings.TrimSpace(oid) == "" {
		return nil, fmt.Errorf("blob OID is empty")
	}
	result := r.git.Run(context.Background(), r.repoRoot, "cat-file", "-p", oid)
	if result.Err != nil || result.ExitCode != 0 {
		detail := strings.TrimSpace(string(result.Stderr))
		if detail == "" && result.Err != nil {
			detail = result.Err.Error()
		}
		return nil, fmt.Errorf("git cat-file -p %s failed (exit %d): %s",
			oid, result.ExitCode, detail)
	}
	return append([]byte(nil), result.Stdout...), nil
}

// ObjectFormat executes `git rev-parse --show-object-format`
// in the resolver's bound repository root and returns the
// trimmed storage format. The function propagates every
// observation failure (no git binary, no repository, malformed
// output) so the verifier can emit V2CodeObjectFormatUnavailable.
//
// R2C-R4-R1: this method NEVER uses the process CWD. The
// resolver is bound at construction time so callers cannot
// accidentally read the wrong repository.
func (r *r2crGitObjectResolver) ObjectFormat() (string, error) {
	if r == nil || r.git == nil {
		return "", fmt.Errorf("git object resolver has no client")
	}
	if strings.TrimSpace(r.repoRoot) == "" {
		return "", fmt.Errorf("git object resolver has no repository root")
	}
	result := r.git.Run(context.Background(), r.repoRoot, "rev-parse", "--show-object-format")
	if result.Err != nil || result.ExitCode != 0 {
		detail := strings.TrimSpace(string(result.Stderr))
		if detail == "" && result.Err != nil {
			detail = result.Err.Error()
		}
		return "", fmt.Errorf("git rev-parse --show-object-format failed (exit %d): %s",
			result.ExitCode, detail)
	}
	format := strings.TrimSpace(string(result.Stdout))
	if format == "" {
		return "", fmt.Errorf("git rev-parse --show-object-format returned empty output")
	}
	return format, nil
}

// NewR2CRGitObjectResolver constructs a repository-bound
// production resolver. It returns a typed error (not a V2Error)
// when repoRoot is empty or git is nil because the caller
// MUST supply a real repoRoot to make the resolver meaningful.
//
// A nil git client defaults to RealGit{}. The resolver will
// route every call to repoRoot; the CWD of the process is
// ignored for every operation.
func NewR2CRGitObjectResolver(g gitClient, repoRoot string) (GitObjectResolver, error) {
	if g == nil {
		g = RealGit{}
	}
	if strings.TrimSpace(repoRoot) == "" {
		return nil, fmt.Errorf("repository root is required for git object resolver")
	}
	return &r2crGitObjectResolver{git: g, repoRoot: repoRoot}, nil
}

// EnforceSHA1ObjectFormat validates that the resolver reports
// "sha1" as its storage format. The check is fail-closed and
// runs BEFORE any OID validation. It returns a typed
// V2Error that distinguishes between observation failure
// (V2CodeObjectFormatUnavailable) and unsupported repository
// state (V2CodeUnsupportedObjectFormat).
//
// The function NEVER inspects an OID's length as a format
// detector; a 64-char OID can still indicate an unsupported
// repository, so length-based shortcuts are forbidden.
func EnforceSHA1ObjectFormat(resolver GitObjectResolver) error {
	if resolver == nil {
		return NewV2ErrorWith(V2CodeObjectFormatUnavailable,
			"git object resolver is required for object-format policy",
			"object_format", "")
	}
	format, err := resolver.ObjectFormat()
	if err != nil {
		return NewV2ErrorWith(V2CodeObjectFormatUnavailable,
			fmt.Sprintf("object format observation failed: %s", err.Error()),
			"object_format", err.Error())
	}
	if format == "" {
		return NewV2ErrorWith(V2CodeObjectFormatUnavailable,
			"object format observation returned empty value",
			"object_format", "")
	}
	if format != "sha1" {
		return NewV2ErrorWith(V2CodeUnsupportedObjectFormat,
			fmt.Sprintf("unsupported object format %q: leamas supports sha1 only", format),
			"object_format", format)
	}
	return nil
}
