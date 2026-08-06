// SPDX-License-Identifier: Apache-2.0

package closure

// verifier_output_worktrees.go implements the linked-worktree
// inventory authority required by Phase 1 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02B.
//
// The CLI cannot decide whether a candidate --output path is
// safely detached from the target repository until it has the
// canonical, absolute list of every worktree attached to that
// repository. The inventory is observed via the bounded
// execution gateway so it shares the same fail-closed
// guarantees as the rest of the verifier.
//
// Inventory source:
//
//	git worktree list --porcelain -z
//
// Upstream git emits one record per worktree, terminated by a
// double-NUL sequence in `-z` mode (because NUL is the field
// terminator and the empty field between records renders as a
// pair of NUL bytes on the wire). Records carry NUL-separated
// fields starting with `worktree <path>` followed by optional
// `HEAD <oid>`, `branch <ref>`, `detached`, `locked`, and
// `prunable` trailers. Importantly, the `-z` flag is the only
// way git allows newline characters inside field values
// (including worktree paths), so the parser MUST NOT treat
// newlines as evidence that `-z` was dropped.
//
// The function fails closed on every error path:
//
//   - spawn failure, timeout, cancellation: rejected
//   - non-zero exit code from git: rejected
//   - output overflow: rejected
//   - empty inventory: rejected
//   - record missing the leading `worktree ` field: rejected
//   - record with an empty worktree path: rejected
//   - record whose worktree path is not absolute: rejected
//   - record whose worktree path cannot be canonicalized
//     (non-existent, broken links, permission errors): rejected
//
// A path-rejected invocation MUST never observe a half-built
// inventory, so the resolver never falls back to a default
// inventory, never silently drops a worktree, and never
// continues with a partial result.

import (
	"bytes"
	"context"
	"fmt"

	"github.com/s1onique/leamas/internal/execution"
)

// worktreeInventoryCommand returns the literal canonical
// invocation for the repository-bound worktree inventory. Tests
// stamp this slice so the link to upstream git is auditable.
func worktreeInventoryCommand() []string {
	return []string{"worktree", "list", "--porcelain", "-z"}
}

// worktreeInventoryMaxBytes bounds the expected inventory
// output. The bounded execution gateway enforces its own
// 8 MiB cap, but this constant documents the design intent.
const worktreeInventoryMaxBytes = 1 << 20 // 1 MiB

// GitRunner is the abstract dependency on the bounded execution
// gateway. Production code satisfies it with execution.RunGit;
// tests substitute a fake or canned-driver implementation.
type GitRunner interface {
	RunGit(ctx context.Context, dir string, args ...string) (execution.GitResult, error)
}

// gitRunnerFunc adapts a plain function to the GitRunner
// interface. Used by the nil-check default in
// InventoryRepositoryWorktrees so callers that supply their own
// runner do not need to construct a type.
type gitRunnerFunc func(ctx context.Context, dir string, args ...string) (execution.GitResult, error)

func (f gitRunnerFunc) RunGit(ctx context.Context, dir string, args ...string) (execution.GitResult, error) {
	return f(ctx, dir, args...)
}

// CanonicalWorktree is one canonical, absolute worktree root
// produced by the inventory authority. The struct is the
// single form the resolver accepts so callers cannot pass
// a relative path by accident.
type CanonicalWorktree struct {
	Path string
}

// RepositoryWorktreeInventory is the canonical, absolute list
// of worktree roots attached to the target repository. The
// inventory always includes the main worktree plus every
// linked worktree git reported. The slice of roots is held
// privately; callers iterate through RootsView() to obtain
// read-only copies.
type RepositoryWorktreeInventory struct {
	roots []string
}

// NewRepositoryWorktreeInventoryForTest is a same-package test
// entry point that lets the test files construct an inventory
// from canonical roots without going through the production
// inventory observation path. The function is exported only
// because Go's test packages live in the same package; it
// must NOT be used from non-test code. The function does no
// validation beyond the empty-input check; tests are
// responsible for producing canonical paths the same way the
// production parser does.
func NewRepositoryWorktreeInventoryForTest(roots []string) (RepositoryWorktreeInventory, error) {
	if len(roots) == 0 {
		return RepositoryWorktreeInventory{}, NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierWorktreeInventoryUnavailable,
			Message:      "verifier output preparation requires at least one canonical worktree root",
			PropertyName: "worktree_inventory",
		})
	}
	copied := make([]string, len(roots))
	copy(copied, roots)
	return RepositoryWorktreeInventory{roots: copied}, nil
}

// RootsView returns a stable, deduplicated copy of the
// canonical worktree roots so callers can iterate without
// exposing the underlying slice to mutation.
func (inv RepositoryWorktreeInventory) RootsView() []string {
	out := make([]string, len(inv.roots))
	copy(out, inv.roots)
	return out
}

// Contains reports whether canonical is equal to or strictly
// inside any worktree root. The function is intended to be
// called only with canonical absolute paths; the inventory
// builder canonicalizes every root before storing it.
func (inv RepositoryWorktreeInventory) Contains(canonical string) bool {
	for _, root := range inv.roots {
		if pathIsInsideOrEqual(root, canonical) {
			return true
		}
	}
	return false
}

// InventoryMainRoot returns the canonical main worktree root
// stored in the inventory. The CLI uses this root to bind
// --repository to the inventory result.
func (inv RepositoryWorktreeInventory) InventoryMainRoot() (string, bool) {
	if len(inv.roots) == 0 {
		return "", false
	}
	return inv.roots[0], true
}

// ContainsRoot reports whether the inventory lists the
// supplied canonical root. Used by PrepareVerifierOutput to
// bind the caller's repositoryRoot into the inventory.
func (inv RepositoryWorktreeInventory) ContainsRoot(canonical string) bool {
	for _, root := range inv.roots {
		if root == canonical {
			return true
		}
	}
	return false
}

// RootsAsCanonicalWorktrees converts the inventory roots to the
// legacy CanonicalWorktree slice. Kept for back-compat with
// callers that have not yet adopted the inventory type.
func (inv RepositoryWorktreeInventory) RootsAsCanonicalWorktrees() []CanonicalWorktree {
	out := make([]CanonicalWorktree, len(inv.roots))
	for i, r := range inv.roots {
		out[i] = CanonicalWorktree{Path: r}
	}
	return out
}

// InventoryRepositoryWorktrees observes the canonical worktree
// inventory for the supplied repository via git's bounded
// execution gateway. The function fails closed: every error
// path produces a typed *V2VerifierError with code
// V2VerifierWorktreeInventoryUnavailable so the CLI can route
// the rejection without parsing message text.
func InventoryRepositoryWorktrees(ctx context.Context, repoRoot string, runner GitRunner) (RepositoryWorktreeInventory, error) {
	if runner == nil {
		runner = gitRunnerFunc(execution.RunGit)
	}
	res, err := runner.RunGit(ctx, repoRoot, worktreeInventoryCommand()...)
	if err != nil {
		return RepositoryWorktreeInventory{}, newWorktreeInventoryError("git worktree list failed: %s", err.Error())
	}
	if res.ExitCode != 0 {
		return RepositoryWorktreeInventory{}, newWorktreeInventoryError(
			"git worktree list exited %d: %s",
			res.ExitCode, bytes.TrimSpace(res.Stderr))
	}
	if int64(len(res.Stdout)) > worktreeInventoryMaxBytes {
		return RepositoryWorktreeInventory{}, newWorktreeInventoryError(
			"git worktree list output exceeded %d bytes", worktreeInventoryMaxBytes)
	}
	roots, err := parseWorktreeInventory(res.Stdout)
	if err != nil {
		return RepositoryWorktreeInventory{}, err
	}
	return NewRepositoryWorktreeInventoryForTest(roots)
}

// worktreeRecordSeparator is the byte sequence that ends a
// single worktree record under `git worktree list --porcelain
// -z`. Upstream git emits an empty field between records, so
// the on-the-wire separator is two consecutive NUL bytes.
var worktreeRecordSeparator = []byte{0x00, 0x00}

// parseWorktreeInventory parses the output of
// `git worktree list --porcelain -z`. Records are NUL-paired;
// each record starts with "worktree <path>" optionally
// followed by "HEAD <oid>", "branch <ref>", "detached",
// "locked", "prunable" trailers. Only the worktree path is
// consumed; the resolver does not need HEAD, branch, or
// status trailers to classify confinement.
//
// The parser is intentionally strict. Upstream git emits a
// "worktree <path>" line per record; the parser refuses:
//
//   - empty output
//   - records missing the "worktree " prefix
//   - empty worktree paths
//   - relative worktree paths (production git is required to
//     emit absolute paths; a relative path is forensic evidence
//     the upstream command was hijacked)
//   - paths that cannot be canonicalized (non-existent, broken
//     links, permission errors): the resolver refuses to treat
//     a lexical stub as authoritative because the canonical
//     containment check would silently underflow.
//
// Importantly, the parser DOES NOT reject newlines or carriage
// returns embedded inside a worktree path. The `-z` flag is
// upstream's explicit opt-in for newlines-in-field-values;
// treating them as malformed would make repositories with such
// paths unverifiable.
func parseWorktreeInventory(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, newWorktreeInventoryError("empty inventory output")
	}
	if !bytes.Contains(raw, []byte{0x00}) {
		return nil, newWorktreeInventoryError(
			"malformed porcelain (missing NUL separators; -z flag absent?)")
	}
	// Find every record boundary. Upstream git emits a trailing
	// double-NUL too; trim it so SplitRecord does not see an
	// extra empty record.
	trimmed := bytes.TrimRight(raw, "\x00")
	records := splitWorktreeRecords(trimmed)
	roots := make([]string, 0, len(records))
	seen := make(map[string]bool, len(records))
	for recIdx, record := range records {
		// Upstream guarantees every record contains a `worktree `
		// prefix field; an absent prefix is forensic evidence the
		// command output was tampered with, so the inventory MUST
		// refuse rather than skip the record.
		pathRaw, ok := extractWorktreeField(record, "worktree ")
		if !ok {
			return nil, newWorktreeInventoryErrorf(
				"record %d: missing worktree prefix in porcelain", recIdx)
		}
		if pathRaw == "" {
			return nil, newWorktreeInventoryErrorf(
				"record %d: empty worktree path in porcelain", recIdx)
		}
		canonical, err := canonicalizeWorktreeRoot(pathRaw)
		if err != nil {
			return nil, newWorktreeInventoryErrorf(
				"record %d: canonicalize worktree %q: %s", recIdx, pathRaw, err.Error())
		}
		if !seen[canonical] {
			seen[canonical] = true
			roots = append(roots, canonical)
		}
	}
	if len(roots) == 0 {
		return nil, newWorktreeInventoryError("parsed inventory is empty")
	}
	return roots, nil
}

// splitWorktreeRecords splits the NUL-bounded porcelain payload
// into per-worktree records. Each record contains NUL-separated
// fields and is terminated by an empty field between records;
// in -z mode the empty field renders as a pair of NUL bytes
// on the wire so a single NUL separates fields inside a
// record and a pair of NULs separates records.
//
// The function walks the payload byte by byte identifying
// record boundaries at the FIRST \0 of any \0\0 pair. Bytes
// inside a single record are not split. The function does
// not interpret newline characters (they may legitimately
// appear inside worktree path field values).
func splitWorktreeRecords(trimmed []byte) [][]byte {
	if len(trimmed) == 0 {
		return nil
	}
	var records [][]byte
	start := 0
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] != 0x00 {
			continue
		}
		// Check whether this NUL is the first half of a \0\0
		// pair that ends the current record.
		if i+1 >= len(trimmed) || trimmed[i+1] != 0x00 {
			continue
		}
		// Record ends at the second NUL (inclusive of the first).
		// Emit the slice [start, i+2) and step the cursor past
		// the pair.
		records = append(records, trimmed[start:i+2])
		start = i + 2
		i++ // Skip the second NUL on the next iteration.
	}
	if start < len(trimmed) {
		records = append(records, trimmed[start:])
	}
	return records
}

// extractWorktreeField returns the value associated with the
// supplied field prefix within a single porcelain record.
// Records are NUL-separated; the function returns the bytes
// between the prefix and the next NUL terminator (or the end
// of the record when the field is the last one). Returns
// false when no field with that prefix is present.
func extractWorktreeField(record []byte, prefix string) (string, bool) {
	prefixBytes := []byte(prefix)
	for i := 0; i < len(record); {
		end := bytes.IndexByte(record[i:], 0x00)
		if end < 0 {
			end = len(record) - i
		}
		field := record[i : i+end]
		if bytes.HasPrefix(field, prefixBytes) {
			return string(field[len(prefixBytes):]), true
		}
		i += end + 1
	}
	return "", false
}

// newWorktreeInventoryError constructs a typed *V2VerifierError
// that carries the supplied message under the canonical
// V2VerifierWorktreeInventoryUnavailable code. PropertyName
// is fixed to "worktree_inventory" so downstream tools can
// route the rejection without parsing message text.
func newWorktreeInventoryError(format string, args ...any) error {
	return NewV2VerifierError(V2VerifierDiagnostic{
		Code:         V2VerifierWorktreeInventoryUnavailable,
		Message:      fmt.Sprintf(format, args...),
		PropertyName: "worktree_inventory",
	})
}

// newWorktreeInventoryErrorf is a thin wrapper that formats
// the message from a caller-supplied format string and
// arguments before constructing the typed diagnostic.
func newWorktreeInventoryErrorf(format string, args ...any) error {
	return newWorktreeInventoryError(format, args...)
}
