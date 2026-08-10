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
// The parser is fail-closed: every framing violation
// produces a typed V2Diagnostic. The parser enforces:
//
//   - non-empty input contains at least one NUL terminator
//     (the -z flag is mandatory)
//   - each "worktree " record is immediately followed by a
//     non-empty HEAD record (the canonical identity Phase 16
//     requires)
//   - path is a non-empty, cleaned, absolute path
//   - HEAD is a non-empty token (OID validation is the caller's
//     responsibility: the verifier authority also leaves OID
//     shape checks to the consumer)
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
	records := bytes.Split(raw, []byte{0x00})
	var (
		out      []SubjectWorktreeRegistration
		path     string
		head     string
		flushReg = func() error {
			if path == "" {
				return nil
			}
			cleaned := filepath.Clean(path)
			if !filepath.IsAbs(cleaned) {
				return fmt.Errorf("relative worktree path %q in porcelain", path)
			}
			if head == "" {
				return fmt.Errorf("worktree path %q in porcelain missing HEAD record", path)
			}
			out = append(out, SubjectWorktreeRegistration{Path: cleaned, Head: head})
			path = ""
			head = ""
			return nil
		}
	)
	for _, rec := range records {
		token := string(bytes.TrimRight(rec, "\r"))
		switch {
		case strings.HasPrefix(token, subjectWorktreeInventoryPorcelainV2ZField):
			if err := flushReg(); err != nil {
				return nil, V2Diagnostics{{
					Code:         V2CodeSubjectObservationUnavailable,
					Message:      fmt.Sprintf("subject worktree inventory observation failed: %s", err.Error()),
					PropertyName: "subject_worktree_inventory",
				}}
			}
			path = strings.TrimSpace(strings.TrimPrefix(token, subjectWorktreeInventoryPorcelainV2ZField))
		case strings.HasPrefix(token, "HEAD "):
			head = strings.TrimSpace(strings.TrimPrefix(token, "HEAD "))
		case token == "":
			// Separator or trailing empty record: ignore.
		}
	}
	if err := flushReg(); err != nil {
		return nil, V2Diagnostics{{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      fmt.Sprintf("subject worktree inventory observation failed: %s", err.Error()),
			PropertyName: "subject_worktree_inventory",
		}}
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
