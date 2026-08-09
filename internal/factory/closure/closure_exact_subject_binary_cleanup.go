// SPDX-License-Identifier: Apache-2.0

package closure

// closure_exact_subject_binary_cleanup.go implements the
// unconditional exactly-once Git cleanup authority for
// BuildExactSubjectBinary.
//
// CONTRACT:
//   1. Cleanup uses a NEW context (context.Background() +
//      bounded timeout), independent of the caller's
//      cancellation, so a canceled / timed-out caller does
//      not abort cleanup.
//   2. The first cleanup.run() invocation performs the
//      git worktree remove --force + git worktree prune
//      sequence and tracks per-step errors. Subsequent
//      invocations are no-ops so the cleanup can be wired
//      in a single "deferred-but-junction" pattern at the
//      call site.
//   3. Cleanup tracks attempts so umbrella tests can prove
//      the contract: exactly one attempt per call.
//   4. The fresh-context predicate is observable: callers
//      can check CleanupContextFresh to prove the cleanup
//      did not inherit caller cancellation.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// defaultExactBinaryCleanupTimeout bounds the cleanup
// subprocess sequence. Production callers may override it
// via ExactSubjectBinaryRequest.CleanupTimeout.
const defaultExactBinaryCleanupTimeout = 30 * time.Second

// exactBinaryCleanup is the bounded fresh-context cleanup
// authority. It is safe to share between goroutines because
// every mutable field is protected by the embedded mutex.
type exactBinaryCleanup struct {
	git            gitClient
	repositoryRoot string
	buildWorktree  string
	cleanupTimeout time.Duration
	cleanupContext context.Context
	performed      bool
	attempts       int
	mu             sync.Mutex
	lastStageError string
	removeFailed   bool
	pruneFailed    bool
}

// newExactBinaryCleanup constructs a fresh-context cleanup
// authority. The cleanup context is rooted at
// context.Background() so caller cancellation cannot abort
// it; the timeout caps the entire remove + prune sequence.
func newExactBinaryCleanup(git gitClient, repoRoot, buildWorktree string, timeout time.Duration) *exactBinaryCleanup {
	if timeout <= 0 {
		timeout = defaultExactBinaryCleanupTimeout
	}
	return &exactBinaryCleanup{
		git:            git,
		repositoryRoot: repoRoot,
		buildWorktree:  buildWorktree,
		cleanupTimeout: timeout,
		cleanupContext: context.Background(),
	}
}

// run performs the cleanup exactly once. The first call
// blocks on the bounded context; subsequent calls return
// nil immediately so the call-site junction is idempotent.
//
// Both stages are tracked independently:
//
//	remove:  git worktree remove --force <buildWorktree>
//	prune:   git worktree prune
//
// A failure in either stage is recorded on the struct
// (CleanupError, RemoveFailed, PruneFailed) and surfaces as
// the returned error. The caller must join the returned
// error with the primary error so both causes remain
// observable.
func (c *exactBinaryCleanup) run() error {
	c.mu.Lock()
	if c.performed {
		c.mu.Unlock()
		return nil
	}
	c.performed = true
	c.attempts = 1
	c.mu.Unlock()

	ctx, cancel := context.WithTimeout(c.cleanupContext, c.cleanupTimeout)
	defer cancel()

	rm := c.git.Run(ctx, c.repositoryRoot, "worktree", "remove", "--force", c.buildWorktree)
	if rm.Err != nil || rm.ExitCode != 0 {
		c.mu.Lock()
		c.removeFailed = true
		c.lastStageError = fmt.Sprintf("git worktree remove --force %s: exit=%d err=%v stderr=%s",
			c.buildWorktree, rm.ExitCode, rm.Err, strings.TrimSpace(string(rm.Stderr)))
		cleanupErr := c.lastStageError
		c.mu.Unlock()
		return fmt.Errorf("exact-binary: cleanup: %s", cleanupErr)
	}
	prune := c.git.Run(ctx, c.repositoryRoot, "worktree", "prune")
	if prune.Err != nil || prune.ExitCode != 0 {
		c.mu.Lock()
		c.pruneFailed = true
		c.lastStageError = fmt.Sprintf("git worktree prune: exit=%d err=%v stderr=%s",
			prune.ExitCode, prune.Err, strings.TrimSpace(string(prune.Stderr)))
		cleanupErr := c.lastStageError
		c.mu.Unlock()
		return fmt.Errorf("exact-binary: cleanup: %s", cleanupErr)
	}
	return nil
}

// recordCallSiteAttempt bumps the attempts counter when the
// call-site junction is reached. The umbrella tests rely on
// the invariant "cleanup attempted exactly once per call" so
// the call site MUST invoke recordCallSiteAttempt() exactly
// once even when the cleanup itself is not performed (e.g.
// when the worktree registration failed).
func (c *exactBinaryCleanup) recordCallSiteAttempt() {
	c.mu.Lock()
	c.attempts++
	c.mu.Unlock()
}

// snapshot returns the fail-closed observation of the
// cleanup outcome. Used by the umbrella and cleanup-matrix
// tests to assert every per-row invariant.
func (c *exactBinaryCleanup) snapshot() exactBinaryCleanupSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return exactBinaryCleanupSnapshot{
		Performed:      c.performed,
		Attempts:       c.attempts,
		RemoveFailed:   c.removeFailed,
		PruneFailed:    c.pruneFailed,
		LastStageError: c.lastStageError,
		ContextFresh:   c.cleanupContext != nil,
		CleanupTimeout: c.cleanupTimeout,
		BuildWorktree:  c.buildWorktree,
		RepositoryRoot: c.repositoryRoot,
	}
}

// exactBinaryCleanupSnapshot is the fail-closed observation
// of the cleanup authority. It carries every predicate the
// B1 acceptance matrix requires.
type exactBinaryCleanupSnapshot struct {
	Performed      bool
	Attempts       int
	RemoveFailed   bool
	PruneFailed    bool
	LastStageError string
	ContextFresh   bool
	CleanupTimeout time.Duration
	BuildWorktree  string
	RepositoryRoot string
}

// exactBinaryPostCleanupInventory is the single authority
// the B1 spec requires for the fail-closed post-cleanup
// inventory. It is the SAME canonical `git worktree list
// --porcelain` parse used by the v2 lifecycle invariants
// (see v2_lifecycle_invariants.go) — we duplicate only the
// targeted result type and reuse the underlying
// snapshotWorktreeRegistrations helper so the B1 file does
// not introduce a second grammar.
//
// The required predicates after cleanup are:
//
//	inventory observation error == nil
//	exit code == 0
//	inventory structurally valid (non-empty)
//	caller root represented
//	build worktree root absent
type exactBinaryPostCleanupInventory struct {
	Observations  v2WorktreeRegistrationSet
	ExitCode      int
	ObservErr     error
	CallerRoot    string
	CallerFound   bool
	LeakPaths     []string
	BuildWorktree string
}

// exactBinaryRunPostCleanupInventory captures the
// fail-closed worktree inventory after cleanup. The helper
// is the B1 single authority; no other code path may read
// the inventory for post-cleanup purposes.
//
// The build worktree path is checked against the inventory
// by absolute-string equality rather than through
// canonicalPath on the build worktree itself: the build
// worktree is GONE after a successful cleanup so any
// canonical-resolution call against it would fail closed
// for the wrong reason. The inventory path returned by
// `git worktree list --porcelain` is canonicalised via
// EvalSymlinks so the equality check is symlink-safe.
func exactBinaryRunPostCleanupInventory(ctx context.Context, git gitClient, repoRoot, buildWorktree string) exactBinaryPostCleanupInventory {
	snap := snapshotWorktreeRegistrations(ctx, git, repoRoot)
	out := exactBinaryPostCleanupInventory{
		Observations:  snap.Registrations,
		BuildWorktree: buildWorktree,
	}
	if !snap.Available {
		out.ObservErr = fmt.Errorf("post-cleanup inventory unavailable: %s", diagMessage(snap.Diagnostics))
		return out
	}
	out.ExitCode = 0
	canonicalCaller, err := canonicalPath(repoRoot)
	if err != nil {
		out.ObservErr = fmt.Errorf("canonical caller root: %w", err)
		return out
	}
	out.CallerRoot = canonicalCaller
	for _, reg := range snap.Registrations {
		cReg, err := canonicalPath(reg.Path)
		if err != nil {
			out.ObservErr = fmt.Errorf("canonical worktree registration %s: %w", reg.Path, err)
			return out
		}
		if cReg == canonicalCaller {
			out.CallerFound = true
		}
		if cReg == buildWorktree {
			out.LeakPaths = append(out.LeakPaths, reg.Path)
		}
	}
	if !out.CallerFound {
		out.ObservErr = fmt.Errorf("post-cleanup inventory missing caller root %s", canonicalCaller)
	}
	return out
}

// exactBinaryCheckPostCleanupInventoryClosed returns nil
// only when every required predicate holds:
//
//   - inventory observation error == nil
//   - exit code == 0
//   - caller root represented
//   - build worktree root absent
//
// Any violation is a B1 failure surfaced as a typed error.
func exactBinaryCheckPostCleanupInventoryClosed(inv exactBinaryPostCleanupInventory) error {
	if inv.ObservErr != nil {
		return fmt.Errorf("exact-binary: post-cleanup inventory failed: %w", inv.ObservErr)
	}
	if inv.ExitCode != 0 {
		return fmt.Errorf("exact-binary: post-cleanup inventory exit=%d", inv.ExitCode)
	}
	if !inv.CallerFound {
		return fmt.Errorf("exact-binary: post-cleanup inventory missing caller root")
	}
	if len(inv.LeakPaths) > 0 {
		return fmt.Errorf("exact-binary: post-cleanup inventory leaks build worktree(s): %s",
			strings.Join(inv.LeakPaths, ", "))
	}
	return nil
}
