package doctrinecompiler

import (
	"sort"
)

// PlannerOptions controls planner behaviour. Defaults are safe; tests
// may override fields such as NowFn to obtain determinism.
type PlannerOptions struct {
	// Providers maps content_id to a content provider. The core
	// providers are wired by LoadCorePack; tests may inject extras.
	Providers map[string]ContentProvider
	// CompilerVersion is recorded in the plan and the lock.
	CompilerVersion string
	// WriteFn, when non-nil, lets tests intercept write attempts. The
	// default implementation never persists anything; the planner is
	// strictly read-only.
	WriteFn func(path TargetPath, data []byte) error
}

// defaultPlannerOptions returns the canonical planner configuration.
func defaultPlannerOptions() PlannerOptions {
	return PlannerOptions{
		Providers:       CoreContentProviders(),
		CompilerVersion: "dev",
	}
}

// Plan computes the desired ProjectionPlan for the given pack, profile,
// and target root. It performs no writes.
//
// The plan classifies every desired action. It also reports Reject
// actions when the target shape is unsafe (for example when an existing
// seeded file would be overwritten by a managed write).
func Plan(pack *Pack, profile Profile, target string) (ProjectionPlan, error) {
	return PlanWithOptions(pack, profile, target, defaultPlannerOptions())
}

// PlanWithOptions is Plan with explicit options.
func PlanWithOptions(pack *Pack, profile Profile, target string, opts PlannerOptions) (ProjectionPlan, error) {
	if pack == nil {
		return ProjectionPlan{}, newError("plan", "pack", "nil pack")
	}
	if string(profile.ID) == "" {
		return ProjectionPlan{}, newError("plan", "profile", "empty profile id")
	}
	resolver, err := NewResolver(target)
	if err != nil {
		return ProjectionPlan{}, err
	}
	// Build the desired projection from the profile, resolving content.
	desired, err := resolveDesiredFor(pack, profile)
	if err != nil {
		return ProjectionPlan{}, err
	}
	// Read the existing projection from the lock file (if present).
	existing, err := readLockManagedPaths(resolver)
	if err != nil {
		return ProjectionPlan{}, err
	}
	// Inspect each desired path against the target.
	actions := make([]ProjectionAction, 0, len(desired)+len(existing))
	unsafe := false
	for _, path := range sortedTargetPaths(desired) {
		entry := desired[path]
		kind, sym, ierr := resolver.InspectPath(path)
		if ierr != nil {
			return ProjectionPlan{}, newError("plan", string(path),
				"inspect: "+ierr.Error())
		}
		if sym != "" {
			actions = append(actions, ProjectionAction{
				Path:      path,
				Ownership: entry.Ownership,
				Class:     ActionReject,
				Origin:    entry.Origin,
				Reason:    "symlink at " + sym,
			})
			unsafe = true
			continue
		}
		switch kind {
		case PathMissing:
			if entry.Ownership == OwnershipSeeded {
				actions = append(actions, ProjectionAction{
					Path:      path,
					Ownership: entry.Ownership,
					Class:     ActionCreateSeeded,
					Origin:    entry.Origin,
					Reason:    "missing",
				})
			} else {
				actions = append(actions, ProjectionAction{
					Path:      path,
					Ownership: entry.Ownership,
					Class:     ActionCreateManaged,
					Origin:    entry.Origin,
					Reason:    "missing",
				})
			}
		case PathDirectory:
			actions = append(actions, ProjectionAction{
				Path:      path,
				Ownership: entry.Ownership,
				Class:     ActionReject,
				Origin:    entry.Origin,
				Reason:    "expected file, found directory",
			})
			unsafe = true
		case PathSymlink:
			actions = append(actions, ProjectionAction{
				Path:      path,
				Ownership: entry.Ownership,
				Class:     ActionReject,
				Origin:    entry.Origin,
				Reason:    "path is a symlink",
			})
			unsafe = true
		case PathOther:
			actions = append(actions, ProjectionAction{
				Path:      path,
				Ownership: entry.Ownership,
				Class:     ActionReject,
				Origin:    entry.Origin,
				Reason:    "unsupported file type",
			})
			unsafe = true
		case PathRegularFile:
			currentDigest, err := DigestFile(resolver.Resolve(path))
			if err != nil {
				return ProjectionPlan{}, newError("plan", string(path),
					"digest current: "+err.Error())
			}
			if entry.Ownership == OwnershipSeeded {
				// Seeded files are target-owned. Existing seeds are
				// preserved regardless of digest.
				actions = append(actions, ProjectionAction{
					Path:      path,
					Ownership: entry.Ownership,
					Class:     ActionPreserveSeeded,
					Origin:    entry.Origin,
					Reason:    "existing seeded file preserved",
				})
				_ = currentDigest
				continue
			}
			if currentDigest == entry.Digest {
				actions = append(actions, ProjectionAction{
					Path:      path,
					Ownership: entry.Ownership,
					Class:     ActionUnchanged,
					Origin:    entry.Origin,
					Reason:    "managed file already current",
				})
			} else {
				actions = append(actions, ProjectionAction{
					Path:      path,
					Ownership: entry.Ownership,
					Class:     ActionUpdateManaged,
					Origin:    entry.Origin,
					Reason:    "managed file drift",
				})
			}
		}
	}
	// Obsolete managed files: present in lock, absent from desired.
	for _, path := range sortedLockPaths(existing) {
		if _, keep := desired[path]; keep {
			continue
		}
		kind, _, err := resolver.InspectPath(path)
		if err != nil {
			return ProjectionPlan{}, newError("plan", string(path),
				"inspect obsolete: "+err.Error())
		}
		if kind == PathMissing {
			// Already gone: do nothing.
			continue
		}
		actions = append(actions, ProjectionAction{
			Path:      path,
			Ownership: OwnershipManaged,
			Class:     ActionRemoveObsoleteManaged,
			Origin:    "lock",
			Reason:    "previously recorded managed file no longer in projection",
		})
	}
	if unsafe {
		return ProjectionPlan{
			PackId:          pack.PackID,
			PackVersion:     pack.PackVersion,
			ProfileId:       profile.ID,
			CompilerVersion: opts.CompilerVersion,
			PackDigest:      pack.PackDigest(),
			Actions:         SortedActions(actions),
		}, newError("plan", "target", "unsafe target shape; refusing to plan")
	}
	return ProjectionPlan{
		PackId:          pack.PackID,
		PackVersion:     pack.PackVersion,
		ProfileId:       profile.ID,
		CompilerVersion: opts.CompilerVersion,
		PackDigest:      pack.PackDigest(),
		Actions:         SortedActions(actions),
	}, nil
}

// resolveContent renders canonical bytes for a declaration and returns
// the typed ProjectionEntry. The ContentID must be registered in the
// provider map. The caller is responsible for assigning the entry's
// Path field after the fact; resolveContent intentionally leaves it
// unset to keep one helper usable for both managed and seeded
// declarations without coupling to a single path type.
func resolveContent(pack *Pack, profile Profile, declID, contentID string, ownership OwnershipMode) (ProjectionEntry, error) {
	provider, ok := CoreContentProviders()[contentID]
	if !ok {
		// Allow injected providers via opts.Providers for tests.
		provider, ok = defaultPlannerOptions().Providers[contentID]
		if !ok {
			return ProjectionEntry{}, newError("plan", declID,
				"unknown content_id: "+contentID)
		}
	}
	data, err := provider(pack, &profile)
	if err != nil {
		return ProjectionEntry{}, err
	}
	return ProjectionEntry{
		Ownership: ownership,
		Content:   data,
		Digest:    ComputeDigest(data),
		Origin:    declID,
	}, nil
}

// resolveDesiredFor builds the desired projection entries from a
// profile, keyed by TargetPath. Each entry has its Path set so that
// downstream callers can index the map by path and recover a fully
// populated entry.
func resolveDesiredFor(pack *Pack, profile Profile) (map[TargetPath]ProjectionEntry, error) {
	out := make(map[TargetPath]ProjectionEntry, len(profile.Outputs)+len(profile.Seeds))
	for _, o := range profile.Outputs {
		entry, err := resolveContent(pack, profile, o.ID, o.ContentID, OwnershipManaged)
		if err != nil {
			return nil, err
		}
		entry.Path = o.Path
		out[o.Path] = entry
	}
	for _, s := range profile.Seeds {
		entry, err := resolveContent(pack, profile, s.ID, s.ContentID, OwnershipSeeded)
		if err != nil {
			return nil, err
		}
		entry.Path = s.Path
		out[s.Path] = entry
	}
	return out, nil
}

// readLockManagedPaths returns the set of managed paths recorded in the
// committed lock file. Returns an empty map when the lock is missing
// or invalid; the planner treats a missing lock as "no prior state".
func readLockManagedPaths(resolver *Resolver) (map[TargetPath]struct{}, error) {
	lockPath := TargetPath(".factory/doctrine.lock.json")
	kind, _, err := resolver.InspectPath(lockPath)
	if err != nil {
		return nil, newError("plan", string(lockPath), "inspect lock: "+err.Error())
	}
	if kind == PathMissing {
		return map[TargetPath]struct{}{}, nil
	}
	lock, err := ReadLockFile(resolver.Resolve(lockPath))
	if err != nil {
		return nil, err
	}
	out := make(map[TargetPath]struct{}, len(lock.ManagedFiles))
	for _, mf := range lock.ManagedFiles {
		out[mf.Path] = struct{}{}
	}
	return out, nil
}

// sortedTargetPaths returns the keys of m in deterministic order.
func sortedTargetPaths(m map[TargetPath]ProjectionEntry) []TargetPath {
	keys := make([]TargetPath, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// sortedLockPaths returns the keys of a lock-managed-path set in
// deterministic order.
func sortedLockPaths(m map[TargetPath]struct{}) []TargetPath {
	keys := make([]TargetPath, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// FormatPlan renders a ProjectionPlan as deterministic human-readable
// text. The format is stable across runs and suitable for golden-file
// tests.
func FormatPlan(plan ProjectionPlan) []byte {
	var b []byte
	b = append(b, "plan:"...)
	b = append(b, Newline...)
	b = append(b, "  pack: "...)
	b = append(b, string(plan.PackId)...)
	b = append(b, " ("...)
	b = append(b, string(plan.PackVersion)...)
	b = append(b, ")"...)
	b = append(b, Newline...)
	b = append(b, "  profile: "...)
	b = append(b, string(plan.ProfileId)...)
	b = append(b, Newline...)
	b = append(b, "  compiler: "...)
	b = append(b, plan.CompilerVersion...)
	b = append(b, Newline...)
	b = append(b, "  pack_digest: "...)
	b = append(b, string(plan.PackDigest)...)
	b = append(b, Newline...)
	b = append(b, "  actions:"...)
	b = append(b, Newline...)
	for _, a := range plan.Actions {
		b = append(b, "    - "...)
		b = append(b, a.Class.String()...)
		b = append(b, " "...)
		b = append(b, string(a.Path)...)
		b = append(b, " ("...)
		b = append(b, a.Ownership.String()...)
		b = append(b, ", "...)
		b = append(b, a.Reason...)
		b = append(b, ")"...)
		b = append(b, Newline...)
	}
	return b
}
