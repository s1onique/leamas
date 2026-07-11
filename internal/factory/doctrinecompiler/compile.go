package doctrinecompiler

import (
	"os"
	"sort"
)

// CompilerOptions controls compiler behaviour.
type CompilerOptions struct {
	CompilerVersion string
	CompilerCommit  string
	// DryRun, when true, computes the plan and a would-be lock without
	// performing any filesystem writes. Useful for tests.
	DryRun bool
	// FailAfter, when set to a non-empty action class, causes the
	// compiler to inject a failure AFTER that action has been recorded.
	// It is intended solely for tests that need to observe
	// transactional rollback behaviour.
	FailAfter ActionClass
}

// Compile applies a ProjectionPlan to the target repository.
//
// The compiler uses a transactional apply model:
//
//  1. Snapshot every affected path (existing files we will replace
//     or delete; the prior lock if any).
//  2. Apply all writes and deletions.
//  3. Write the lock file LAST.
//  4. On any failure, restore created, updated, deleted, and prior
//     lock paths exactly.
//
// Compile refuses to run if any action has class ActionReject.
func Compile(pack *Pack, profile Profile, target string, opts CompilerOptions) (ProjectionPlan, error) {
	if pack == nil {
		return ProjectionPlan{}, newError("compile", "pack", "nil pack")
	}
	// Use the supplied compiler version; fall back to "dev" so
	// PlanWithOptions has a stable value when none is supplied.
	version := canonicalVersion(opts.CompilerVersion)
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

	// SNAPSHOT.
	snapshot := make(map[string][]byte)
	record := func(abs string) {
		if _, ok := snapshot[abs]; ok {
			return
		}
		data, err := os.ReadFile(abs)
		if err == nil {
			snapshot[abs] = data
			return
		}
		snapshot[abs] = nil
	}
	for _, a := range plan.Actions {
		switch a.Class {
		case ActionCreateManaged, ActionCreateSeeded, ActionUpdateManaged:
			record(resolver.Resolve(a.Path))
		case ActionRemoveObsoleteManaged:
			record(resolver.Resolve(a.Path))
		}
	}
	record(resolver.Resolve(TargetPath(".factory/doctrine.lock.json")))

	// APPLY.
	failed := false
	for _, a := range plan.Actions {
		switch a.Class {
		case ActionUnchanged, ActionPreserveSeeded:
			continue
		case ActionReject:
			failed = true
			continue
		}
		if opts.FailAfter != ActionInvalid && a.Class == opts.FailAfter {
			failed = true
			continue
		}
		entry, ok := desired[a.Path]
		if !ok {
			abs := resolver.Resolve(a.Path)
			if !resolver.Contains(abs) {
				failed = true
				continue
			}
			if sym, _ := resolver.HasSymlinkEscape(a.Path); sym != "" {
				failed = true
				continue
			}
			if err := removeFileIfExists(abs); err != nil {
				failed = true
				continue
			}
			continue
		}
		abs := resolver.Resolve(a.Path)
		if !resolver.Contains(abs) {
			failed = true
			continue
		}
		if sym, escaped := resolver.HasSymlinkEscape(a.Path); escaped {
			_ = sym
			failed = true
			continue
		}
		if err := ensureParentDir(abs); err != nil {
			failed = true
			continue
		}
		if _, err := writeAtomicFile(abs, entry.Content, 0o644); err != nil {
			failed = true
			continue
		}
	}
	if failed {
		restoreSnapshot(resolver.Root, snapshot)
		return plan, newError("compile", "apply",
			"transactional apply failed; target restored to pre-compile state")
	}

	// LOCK (last).
	lock := BuildLockFromPlan(plan, managed, seeded, profile.ObservedContracts, canonicalCommit(opts.CompilerCommit))
	lockBytes, err := FormatLockFile(lock)
	if err != nil {
		restoreSnapshot(resolver.Root, snapshot)
		return plan, newError("compile", "lock", err.Error())
	}
	lockPath := resolver.Resolve(TargetPath(".factory/doctrine.lock.json"))
	if err := ensureParentDir(lockPath); err != nil {
		restoreSnapshot(resolver.Root, snapshot)
		return plan, newError("compile", "lock", err.Error())
	}
	if _, err := writeAtomicFile(lockPath, lockBytes, 0o644); err != nil {
		restoreSnapshot(resolver.Root, snapshot)
		return plan, newError("compile", "lock", err.Error())
	}
	return plan, nil
}

// restoreSnapshot rewrites every recorded path to its prior bytes. If
// the prior state was a missing file, the path is removed. Any I/O
// failure during restore is silently ignored: rollback must be best
// effort, the original error has already been recorded.
func restoreSnapshot(root string, snapshot map[string][]byte) {
	keys := make([]string, 0, len(snapshot))
	for k := range snapshot {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })
	for _, abs := range keys {
		prior := snapshot[abs]
		if prior == nil {
			_ = removeFileIfExists(abs)
			continue
		}
		_ = ensureParentDir(abs)
		_, _ = writeAtomicFile(abs, prior, 0o644)
	}
}

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
