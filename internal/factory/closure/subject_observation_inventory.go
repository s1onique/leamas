// SPDX-License-Identifier: Apache-2.0

package closure

// subject_observation_inventory.go implements the
// R6-A subject worktree inventory snapshot helper. The
// helper captures `git worktree list --porcelain -z` from
// the supplied repository root and parses it into
// (Path, HEAD) registrations so the executor can bind the
// actual worktree path to the actual subject commit.
//
// The act requires the parser to use the -z flag (NUL
// framing) through the existing bounded Git authority.
// The bounded authority is the production gitClient
// interface; the -z variant is the chosen R6-A wire format
// because it is the same wire format the verifier output
// authority uses (verifier_output_worktrees.go) and because
// it pairs with a NUL terminator that cannot be embedded
// in a path on POSIX.
//
// Splitting this from subject_observation.go keeps both
// files under the LLM-friendly 400-line threshold while
// preserving the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// subjectWorktreeInventoryPorcelainV2ZField is the record
// prefix emitted by `git worktree list --porcelain -z`. Each
// record begins with the literal NUL-terminated token
// "worktree ", then the path, then the NUL terminator.
const subjectWorktreeInventoryPorcelainV2ZField = "worktree "

// observeSubjectWorktreeInventory runs
// `git worktree list --porcelain -z` from the supplied
// repository root and parses the registrations into
// (Path, HEAD) pairs. The function is fail-closed: an
// observation failure leaves Available=false and populates
// Diagnostics with a typed V2Diagnostic.
//
// The repository root is the input the bounded Git authority
// already requires; the helper runs the same command the
// verifier_output_worktrees.go parser uses so the wire
// format is shared with the existing authority.
func observeSubjectWorktreeInventory(ctx context.Context, git gitClient, repoRoot string) SubjectWorktreeInventory {
	inv := SubjectWorktreeInventory{}
	if strings.TrimSpace(repoRoot) == "" || git == nil {
		inv.Diagnostics = V2Diagnostics{{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      "subject worktree inventory observation failed: no repository root or Git client",
			PropertyName: "subject_worktree_inventory",
		}}
		return inv
	}
	res := git.Run(ctx, repoRoot, "worktree", "list", "--porcelain", "-z")
	if res.Err != nil || res.ExitCode != 0 {
		inv.Diagnostics = V2Diagnostics{{
			Code: V2CodeSubjectObservationUnavailable,
			Message: fmt.Sprintf("subject worktree inventory observation failed: exit=%d stderr=%q",
				res.ExitCode, strings.TrimSpace(string(res.Stderr))),
			PropertyName: "subject_worktree_inventory",
		}}
		return inv
	}
	regs, diags := parseSubjectWorktreeInventoryPorcelainZ(res.Stdout)
	if len(diags) > 0 {
		// Map the parser diagnostics into a uniform
		// subject_observation_unavailable so the typed code
		// family is consistent for the executor.
		out := make(V2Diagnostics, 0, len(diags))
		for _, d := range diags {
			out = append(out, V2Diagnostic{
				Code:         V2CodeSubjectObservationUnavailable,
				Message:      d.Message,
				PropertyName: d.PropertyName,
			})
		}
		inv.Diagnostics = out
		return inv
	}
	inv.Available = true
	inv.Registrations = regs
	return inv
}

// parseSubjectWorktreeInventoryPorcelainZ parses the
// NUL-framed output of `git worktree list --porcelain -z`.
//
// The wire format is a sequence of records, each:
//
//	"worktree " <path> \x00
//	"HEAD " <oid> \x00
//	[<other fields> \x00]*
//
// R6-A-CORRECTION01 enforces strict framing: the entire
// payload MUST end in a NUL terminator (the -z flag is
// mandatory and a truncated final record is rejected).
// The parser also enforces one worktree + one HEAD per
// record, rejects duplicate paths, and validates the HEAD
// token against the repository object format. The
// bounded NUL framing is what makes lossless path
// preservation possible; the previous prose omitted
// these guards.
//
// The parser is fail-closed: every framing violation
// produces a typed V2Diagnostic. The parser enforces:
//
//   - non-empty input MUST contain at least one NUL
//     terminator (the -z flag is mandatory)
//   - the final byte MUST be NUL (truncated records are
//     rejected; the canonical exit-0 wire is NUL-LF when
//     line-oriented, but the -z variant terminates every
//     record with NUL)
//   - each record contains exactly one "worktree " line
//     and exactly one "HEAD " line
//   - path bytes are preserved verbatim (no TrimSpace;
//     the -z form exists specifically so paths
//     containing whitespace or newline characters
//     round-trip without lossy normalization)
//   - path is a non-empty, absolute, cleaned path
//   - HEAD is a non-empty 40- or 64-character lowercase
//     hex object identifier
//   - duplicate worktree paths across records are
//     rejected so the canonical (Path, HEAD) pair
//     identity is unambiguous
//
// Registrations preserve the canonical order returned by
// the parser. Equal/FindByPath on SubjectWorktreeInventory
// do not depend on order.
func parseSubjectWorktreeInventoryPorcelainZ(raw []byte) ([]SubjectWorktreeRegistration, V2Diagnostics) {
	if len(raw) == 0 {
		// An empty buffer means the porcelain output was
		// truncated; the -z flag guarantees at least one
		// NUL terminator. We refuse to fabricate a row.
		return nil, V2Diagnostics{{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      "subject worktree inventory observation failed: empty porcelain output",
			PropertyName: "subject_worktree_inventory",
		}}
	}
	if !bytes.Contains(raw, []byte{0x00}) {
		return nil, V2Diagnostics{{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      "subject worktree inventory observation failed: porcelain output missing NUL framing",
			PropertyName: "subject_worktree_inventory",
		}}
	}
	if raw[len(raw)-1] != 0x00 {
		return nil, V2Diagnostics{{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      "subject worktree inventory observation failed: porcelain output missing terminal NUL",
			PropertyName: "subject_worktree_inventory",
		}}
	}
	// Split on NUL so the trailing NUL produces an empty
	// trailing segment that the parser records as "record
	// end" without emitting a structural field.
	segments := bytes.Split(raw, []byte{0x00})
	// The trailing NUL is mandatory; bytes.Split produces
	// one fewer segment than NUL count, so the last segment
	// is "" only when the final NUL was present. Any other
	// value here would mean the wire was malformed.
	if len(segments) == 0 || string(segments[len(segments)-1]) != "" {
		return nil, V2Diagnostics{{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      "subject worktree inventory observation failed: porcelain output missing terminal NUL",
			PropertyName: "subject_worktree_inventory",
		}}
	}
	seen := make(map[string]struct{}, 0)
	var out []SubjectWorktreeRegistration
	for _, seg := range segments {
		token := string(seg)
		if token == "" {
			continue
		}
		switch {
		case strings.HasPrefix(token, subjectWorktreeInventoryPorcelainV2ZField):
			path := strings.TrimPrefix(token, subjectWorktreeInventoryPorcelainV2ZField)
			// Do NOT TrimSpace the path: the -z form
			// exists specifically to preserve field
			// bytes including embedded newlines and
			// trailing whitespace.
			if path == "" {
				return nil, V2Diagnostics{{
					Code:         V2CodeSubjectObservationUnavailable,
					Message:      "subject worktree inventory observation failed: empty worktree path in porcelain",
					PropertyName: "subject_worktree_inventory",
				}}
			}
			cleaned := filepath.Clean(path)
			if !filepath.IsAbs(cleaned) {
				return nil, V2Diagnostics{{
					Code:         V2CodeSubjectObservationUnavailable,
					Message:      fmt.Sprintf("subject worktree inventory observation failed: relative worktree path %q in porcelain", path),
					PropertyName: "subject_worktree_inventory",
				}}
			}
			if _, dup := seen[cleaned]; dup {
				return nil, V2Diagnostics{{
					Code:         V2CodeSubjectObservationUnavailable,
					Message:      fmt.Sprintf("subject worktree inventory observation failed: duplicate worktree path %q in porcelain", cleaned),
					PropertyName: "subject_worktree_inventory",
				}}
			}
			seen[cleaned] = struct{}{}
			out = append(out, SubjectWorktreeRegistration{Path: cleaned})
		case strings.HasPrefix(token, "HEAD "):
			head := strings.TrimPrefix(token, "HEAD ")
			if !isValidSubjectHeadObjectFormat(head) {
				return nil, V2Diagnostics{{
					Code:         V2CodeSubjectObservationUnavailable,
					Message:      fmt.Sprintf("subject worktree inventory observation failed: HEAD %q is not a 40- or 64-character lowercase hex OID", head),
					PropertyName: "subject_worktree_inventory",
				}}
			}
			if len(out) == 0 {
				return nil, V2Diagnostics{{
					Code:         V2CodeSubjectObservationUnavailable,
					Message:      "subject worktree inventory observation failed: HEAD record before any worktree record",
					PropertyName: "subject_worktree_inventory",
				}}
			}
			if out[len(out)-1].Head != "" {
				return nil, V2Diagnostics{{
					Code:         V2CodeSubjectObservationUnavailable,
					Message:      fmt.Sprintf("subject worktree inventory observation failed: duplicate HEAD record for path %q", out[len(out)-1].Path),
					PropertyName: "subject_worktree_inventory",
				}}
			}
			out[len(out)-1].Head = head
		default:
			// R6-A-CORRECTION01: a canonical "worktree"
			// record can carry known Git fields
			// (branch, detached, locked, prunable) which
			// the parser tolerates by ignoring. Any other
			// token is rejected so upstream protocol
			// additions cannot silently change the
			// canonical (Path, HEAD) identity.
			if isKnownPorcelainAnnotation(token) {
				continue
			}
			return nil, V2Diagnostics{{
				Code:         V2CodeSubjectObservationUnavailable,
				Message:      fmt.Sprintf("subject worktree inventory observation failed: unknown structural token %q in porcelain", token),
				PropertyName: "subject_worktree_inventory",
			}}
		}
	}
	// Every emitted registration MUST carry a HEAD; an
	// in-flight worktree without a HEAD record is a wire
	// violation.
	for _, reg := range out {
		if reg.Head == "" {
			return nil, V2Diagnostics{{
				Code:         V2CodeSubjectObservationUnavailable,
				Message:      fmt.Sprintf("subject worktree inventory observation failed: worktree path %q missing HEAD record", reg.Path),
				PropertyName: "subject_worktree_inventory",
			}}
		}
	}
	if len(out) == 0 {
		return nil, V2Diagnostics{{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      "subject worktree inventory observation failed: parser produced no registrations",
			PropertyName: "subject_worktree_inventory",
		}}
	}
	return out, nil
}

// isValidSubjectHeadObjectFormat reports whether s is a
// 40- or 64-character lowercase hex string. The R6-A
// authority accepts both SHA-1 and SHA-256 object formats
// because the repository object-format authority is the
// canonical source of truth; the parser enforces shape
// only so unknown future formats cannot leak through.
func isValidSubjectHeadObjectFormat(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, ch := range s {
		switch {
		case ch >= '0' && ch <= '9':
		case ch >= 'a' && ch <= 'f':
		default:
			return false
		}
	}
	return true
}

// isKnownPorcelainAnnotation reports whether the supplied
// token is a recognised Git worktree-list annotation that
// the parser tolerates by ignoring. The canonical
// "worktree" record can carry:
//
//	branch <refname>     : the current branch of the worktree
//	detached <oid>       : a detached HEAD
//	locked <reason>      : the worktree is administratively locked
//	prunable <reason>    : the worktree is administratively prunable
//	pruned               : the worktree has been pruned
//
// R6-A-CORRECTION01 tolerates these because they are
// non-structural and do not change the canonical
// (Path, HEAD) identity the executor relies on. The
// annotation set is intentionally narrow: any other token
// is rejected so a future Git protocol addition cannot
// silently introduce a structural field.
func isKnownPorcelainAnnotation(token string) bool {
	switch {
	case strings.HasPrefix(token, "branch "):
		return true
	case strings.HasPrefix(token, "detached"):
		// `detached` may appear as either the prefix
		// `detached <oid>` or the bare token `detached`
		// depending on the Git version.
		return true
	case strings.HasPrefix(token, "locked"):
		return true
	case strings.HasPrefix(token, "prunable"):
		return true
	case token == "pruned":
		return true
	}
	return false
}
