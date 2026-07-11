package doctrinecompiler

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CompilerOptions controls compiler behaviour.
type CompilerOptions struct {
	CompilerVersion string
	CompilerCommit  string
	// DryRun, when true, computes the plan and a would-be lock without
	// performing any filesystem writes. Useful for tests.
	DryRun bool
	// FailAfterN, when non-zero, causes the compiler to inject a
	// failure AFTER exactly N mutations have succeeded. The fault
	// injection point is positioned so that at least one mutation
	// has already been recorded in the journal, ensuring rollback
	// has real work to do. It is intended solely for tests that
	// need to observe transactional rollback behaviour. The
	// production default is zero (no injection).
	FailAfterN int
}

// Compile applies a ProjectionPlan to the target repository.
//
// The compiler uses an explicit transactional apply model:
//
//  1. Finish planning and all validation.
//  2. Identify every path that may be created, updated, removed, or
//     replaced, and snapshot its pre-state (existence, mode, bytes).
//  3. Apply mutations in deterministic order, journal each success.
//  4. Write the lock file LAST.
//  5. On any failure, walk the journal in reverse and restore the
//     pre-state. Rollback errors are surfaced via errors.Join rather
//     than silently discarded.
//
// Compile refuses to run if any action has class ActionReject.
func Compile(pack *Pack, profile Profile, target string, opts CompilerOptions) (ProjectionPlan, error) {
	if pack == nil {
		return ProjectionPlan{}, newError("compile", "pack", "nil pack")
	}
	// Compiler-version compatibility check. When the caller supplies
	// no version, fall back to the runtime source so library tests
	// using withCompilerVersion(...) still work.
	version := opts.CompilerVersion
	if version == "" {
		version = compilerVersionSource()
	}
	version = canonicalVersion(version)
	if err := CheckCompilerCompatibility(pack.CompilerVersion, version); err != nil {
		return ProjectionPlan{}, newError("compile", "compiler_version", err.Error())
	}
	plan, err := PlanWithOptions(pack, profile, target, PlannerOptions{
		Providers:       CoreContentProviders(),
		CompilerVersion: version,
	})
	if err != nil {
		return plan, err
	}
	if hasReject(plan.Actions) {
		return plan, newError("compile", "plan",
			"plan contains reject actions; refusing to compile")
	}
	if opts.DryRun {
		return plan, nil
	}
	resolver, err := NewResolver(target)
	if err != nil {
		return plan, err
	}
	desired, err := resolveDesiredFor(pack, profile)
	if err != nil {
		return plan, err
	}
	var managed []ProjectionEntry
	var seeded []TargetPath
	for _, o := range profile.Outputs {
		managed = append(managed, desired[o.Path])
	}
	for _, s := range profile.Seeds {
		seeded = append(seeded, s.Path)
	}
	sort.Slice(managed, func(i, j int) bool { return managed[i].Path < managed[j].Path })
	sort.Slice(seeded, func(i, j int) bool { return seeded[i] < seeded[j] })

	// Identify every path that may be touched.
	absPaths := make([]string, 0, len(plan.Actions)+1)
	for _, a := range plan.Actions {
		if a.Class == ActionUnchanged || a.Class == ActionPreserveSeeded || a.Class == ActionReject {
			continue
		}
		absPaths = append(absPaths, resolver.Resolve(a.Path))
	}
	lockPath := resolver.Resolve(TargetPath(".factory/doctrine.lock.json"))
	absPaths = append(absPaths, lockPath)

	// Snapshot pre-state (existence, mode, bytes) before any write.
	trans, err := beginTransaction(resolver.Root, absPaths)
	if err != nil {
		return plan, newError("compile", "snapshot", err.Error())
	}

	// APPLY.
	mutationsDone := 0
	for _, a := range plan.Actions {
		switch a.Class {
		case ActionUnchanged, ActionPreserveSeeded:
			continue
		case ActionReject:
			rerr := trans.rollback()
			return plan, applyFailure("plan contained reject action", rerr)
		}
		abs := resolver.Resolve(a.Path)
		if !resolver.Contains(abs) {
			rerr := trans.rollback()
			return plan, applyFailure("path escapes target root: "+abs, rerr)
		}
		if sym, escaped := resolver.HasSymlinkEscape(a.Path); escaped {
			_ = sym
			rerr := trans.rollback()
			return plan, applyFailure("symlink escape: "+abs, rerr)
		}
		entry, ok := desired[a.Path]
		if !ok {
			// Removal of an obsolete managed file.
			pre := trans.filePre[abs]
			if !pre.existed {
				continue
			}
			if err := removeFileIfExists(abs); err != nil {
				rerr := trans.rollback()
				return plan, applyFailure("remove "+string(a.Path)+": "+err.Error(), rerr)
			}
			trans.recordRemove(abs)
			mutationsDone++
			if opts.FailAfterN > 0 && mutationsDone >= opts.FailAfterN {
				rerr := trans.rollback()
				return plan, applyFailure("injected failure after mutation", rerr)
			}
			continue
		}
		// Create or replace.
		pre := trans.filePre[abs]
		if err := ensureParentDirTracking(trans, abs); err != nil {
			rerr := trans.rollback()
			return plan, applyFailure("mkdir for "+string(a.Path)+": "+err.Error(), rerr)
		}
		if _, err := writeAtomicFile(abs, entry.Content, 0o644); err != nil {
			rerr := trans.rollback()
			return plan, applyFailure("write "+string(a.Path)+": "+err.Error(), rerr)
		}
		if !pre.existed {
			trans.recordCreate(abs, 0o644, entry.Content)
		} else {
			trans.recordReplace(abs, 0o644, entry.Content)
		}
		mutationsDone++
		if opts.FailAfterN > 0 && mutationsDone >= opts.FailAfterN {
			rerr := trans.rollback()
			return plan, applyFailure("injected failure after mutation", rerr)
		}
	}
	// LOCK (last).
	lock := BuildLockFromPlan(plan, managed, seeded, profile.ObservedContracts, canonicalCommit(opts.CompilerCommit))
	lockBytes, err := FormatLockFile(lock)
	if err != nil {
		rerr := trans.rollback()
		return plan, applyFailure("lock format: "+err.Error(), rerr)
	}
	if err := ensureParentDirTracking(trans, lockPath); err != nil {
		rerr := trans.rollback()
		return plan, applyFailure("lock parent: "+err.Error(), rerr)
	}
	preLock := trans.filePre[lockPath]
	if _, err := writeAtomicFile(lockPath, lockBytes, 0o644); err != nil {
		rerr := trans.rollback()
		return plan, applyFailure("write lock: "+err.Error(), rerr)
	}
	if !preLock.existed {
		trans.recordCreate(lockPath, 0o644, lockBytes)
	} else {
		trans.recordReplace(lockPath, 0o644, lockBytes)
	}
	return plan, nil
}

// applyFailure builds the error returned by Compile when the apply
// phase fails. The apply error always wraps ErrApplyFailed; when the
// subsequent rollback also fails, the rollback error is wrapped with
// ErrRollbackFailed and the two are joined via errors.Join so the
// caller can inspect either failure with errors.Is.
func applyFailure(applyMsg string, rollbackErr error) error {
	applyErr := &CompileError{
		Phase:   "compile",
		Subject: "apply",
		Reason:  applyMsg,
	}
	wrappedApply := &sentinelErr{marker: ErrApplyFailed, inner: applyErr}
	if rollbackErr == nil {
		// Rollback succeeded; the target is safely restored. The
		// message documents that for human readers but the
		// underlying chain remains inspectable.
		applyErr.Reason = "transactional apply failed; target restored to pre-compile state: " + applyMsg
		return wrappedApply
	}
	// Restoration was incomplete. The message says so explicitly.
	applyErr.Reason = "transactional apply failed; target restoration INCOMPLETE: " + applyMsg +
		"; rollback reported: " + rollbackErr.Error()
	wrappedRollback := &sentinelErr{marker: ErrRollbackFailed, inner: rollbackErr}
	return errors.Join(wrappedApply, wrappedRollback)
}

// sentinelErr wraps an inner error with a sentinel marker so that
// errors.Is can match the marker while preserving the underlying
// error chain.
type sentinelErr struct {
	marker error
	inner  error
}

func (e *sentinelErr) Error() string {
	if e.inner == nil {
		return e.marker.Error()
	}
	return e.marker.Error() + ": " + e.inner.Error()
}

func (e *sentinelErr) Unwrap() error { return e.inner }

func (e *sentinelErr) Is(target error) bool {
	return errors.Is(e.marker, target)
}

// ensureParentDirTracking creates the parent directory of path and
// records every newly created directory in the transaction. It walks
// from the deepest missing ancestor upward, then creates the
// directories in shallowest-to-deepest order so each Mkdir finds its
// parent already in place.
func ensureParentDirTracking(t *transaction, abs string) error {
	cur := filepath.Dir(abs)
	needed := make([]string, 0)
	for cur != "" && cur != "." && cur != string(filepath.Separator) {
		rel, err := filepath.Rel(t.root, cur)
		if err != nil || strings.HasPrefix(rel, "..") {
			break
		}
		if rel == "" {
			break
		}
		lst, lerr := os.Lstat(cur)
		if lerr == nil {
			if lst.IsDir() {
				break
			}
			return errors.New("path exists but is not a directory: " + cur)
		}
		needed = append(needed, cur)
		cur = filepath.Dir(cur)
	}
	for i := len(needed) - 1; i >= 0; i-- {
		if err := os.Mkdir(needed[i], 0o755); err != nil {
			return err
		}
		t.noteDirCreated(needed[i])
	}
	return nil
}

// Apply and rollback sentinels. The transaction guarantees that
// these sentinels wrap the returned error chain on the relevant
// failure path, so callers can use errors.Is to distinguish the
// failure categories without parsing strings.
var (
	ErrApplyFailed    = errors.New("apply failed")
	ErrRollbackFailed = errors.New("rollback failed")
)

// canonicalCommit normalises the compiler_commit field.
func canonicalCommit(c string) string {
	if c == "" {
		return "unknown"
	}
	return c
}

// canonicalVersion normalises the compiler_version field.
func canonicalVersion(v string) string {
	if v == "" {
		return "dev"
	}
	return v
}

// hasReject reports whether any action in the slice is a reject.
func hasReject(actions []ProjectionAction) bool {
	for _, a := range actions {
		if a.Class == ActionReject {
			return true
		}
	}
	return false
}

// fileExists is a convenience wrapper around os.Stat.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
