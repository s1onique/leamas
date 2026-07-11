package doctrinecompiler

import (
	"fmt"
	"sort"
)

// VerifyResult is the typed output of a verify pass.
type VerifyResult struct {
	OK       bool
	Findings []VerifyFinding
}

// VerifyFinding is one issue found while verifying a projection.
type VerifyFinding struct {
	Path    string
	Kind    string
	Message string
}

// compilerVersionSource returns the current compiler version. It is
// swappable for tests; the default returns "dev" so production
// binaries without -ldflags injection still pass compatibility checks.
var compilerVersionSource = func() string { return "dev" }

// SetCompilerVersionSource overrides the compiler version source.
// It is intended for use by the CLI dispatcher so that production
// builds inject their real version, and by tests so that the
// verifier can be exercised against a known compiler identity.
func SetCompilerVersionSource(fn func() string) {
	if fn == nil {
		compilerVersionSource = func() string { return "dev" }
		return
	}
	compilerVersionSource = fn
}

// Verify inspects the target repository against the committed lock and
// the canonical pack. It performs no writes.
//
// The verifier enforces three-way consistency:
//
//	canonical desired digest == lock digest == actual file digest
//
// It also detects missing or corrupted selector, exact managed-path
// set drift, exact seeded-path set drift (in both directions), exact
// observed-contract set drift, duplicate normalised paths in the
// lock, lock entries whose paths escape the target root, and
// compiler-version incompatibility against the pack's constraint.
func Verify(pack *Pack, profile Profile, target string) (VerifyResult, error) {
	resolver, err := NewResolver(target)
	if err != nil {
		return VerifyResult{}, err
	}
	result := VerifyResult{OK: true}
	desired, err := resolveDesiredFor(pack, profile)
	if err != nil {
		return result, err
	}
	// Selector load.
	sel, err := readSelector(resolver.Resolve(SelectorPath))
	if err != nil {
		result.OK = false
		result.Findings = append(result.Findings, VerifyFinding{
			Path:    string(SelectorPath),
			Kind:    "selector_invalid",
			Message: err.Error(),
		})
	} else {
		// Selector-pack fidelity: until a pack registry exists, only
		// the loaded pack is supported. Reject any selector that
		// names a different pack, and identify both the requested
		// and the available pack in the error.
		if sel.Pack != string(pack.PackID) {
			result.OK = false
			result.Findings = append(result.Findings, VerifyFinding{
				Path:    string(SelectorPath),
				Kind:    "selector_pack_mismatch",
				Message: fmt.Sprintf("selector requests unsupported pack %q; available pack is %q", sel.Pack, pack.PackID),
			})
			// Do not silently fall through to profile matching
			// against the loaded pack: a foreign selector is
			// always rejected first.
			sortFindings(result.Findings)
			return result, nil
		}
		if sel.Profile != string(profile.ID) {
			result.OK = false
			result.Findings = append(result.Findings, VerifyFinding{
				Path:    string(SelectorPath),
				Kind:    "selector_profile_mismatch",
				Message: fmt.Sprintf("selector profile=%q does not match profile %q", sel.Profile, profile.ID),
			})
		}
	}
	kind, _, err := resolver.InspectPath(TargetPath(".factory/doctrine.lock.json"))
	if err != nil {
		return result, err
	}
	if kind == PathMissing {
		result.OK = false
		result.Findings = append(result.Findings, VerifyFinding{
			Path:    ".factory/doctrine.lock.json",
			Kind:    "lock_missing",
			Message: "no committed lock file; run `leamas factory doctrine compile`",
		})
		sortFindings(result.Findings)
		return result, nil
	}
	lock, err := ReadLockFile(lockPath(resolver))
	if err != nil {
		return result, err
	}
	// Identity.
	if lock.PackID != string(pack.PackID) {
		result.OK = false
		result.Findings = append(result.Findings, VerifyFinding{
			Path:    ".factory/doctrine.lock.json",
			Kind:    "pack_mismatch",
			Message: fmt.Sprintf("lock pack_id=%q does not match pack %q", lock.PackID, pack.PackID),
		})
	}
	if lock.ProfileID != string(profile.ID) {
		result.OK = false
		result.Findings = append(result.Findings, VerifyFinding{
			Path:    ".factory/doctrine.lock.json",
			Kind:    "profile_mismatch",
			Message: fmt.Sprintf("lock profile_id=%q does not match profile %q", lock.ProfileID, profile.ID),
		})
	}
	if lock.PackDigest != string(pack.PackDigest()) {
		result.OK = false
		result.Findings = append(result.Findings, VerifyFinding{
			Path:    ".factory/doctrine.lock.json",
			Kind:    "pack_digest_mismatch",
			Message: "lock pack_digest does not match canonical pack bytes",
		})
	}
	// Three-way digest check.
	wantDigests := make(map[TargetPath]ContentDigest, len(desired))
	for p, e := range desired {
		wantDigests[p] = e.Digest
	}
	managed := append([]ManagedFileEntry(nil), lock.ManagedFiles...)
	sort.Slice(managed, func(i, j int) bool { return managed[i].Path < managed[j].Path })
	for _, mf := range managed {
		if !resolver.Contains(resolver.Resolve(mf.Path)) {
			result.OK = false
			result.Findings = append(result.Findings, VerifyFinding{
				Path:    string(mf.Path),
				Kind:    "managed_escape",
				Message: "lock path escapes target root",
			})
			continue
		}
		canonical, ok := wantDigests[mf.Path]
		if !ok {
			result.OK = false
			result.Findings = append(result.Findings, VerifyFinding{
				Path:    string(mf.Path),
				Kind:    "managed_unexpected",
				Message: "file recorded as managed but absent from current projection",
			})
			continue
		}
		if canonical != mf.Digest {
			result.OK = false
			result.Findings = append(result.Findings, VerifyFinding{
				Path:    string(mf.Path),
				Kind:    "lock_digest_mismatch",
				Message: fmt.Sprintf("lock digest=%s does not match canonical desired digest=%s", mf.Digest, canonical),
			})
			continue
		}
		mkind, _, err := resolver.InspectPath(mf.Path)
		if err != nil {
			return result, err
		}
		if mkind == PathMissing {
			result.OK = false
			result.Findings = append(result.Findings, VerifyFinding{
				Path:    string(mf.Path),
				Kind:    "managed_missing",
				Message: "managed file recorded in lock is missing from target",
			})
			continue
		}
		actual, err := DigestFile(resolver.Resolve(mf.Path))
		if err != nil {
			return result, err
		}
		if actual != mf.Digest {
			result.OK = false
			result.Findings = append(result.Findings, VerifyFinding{
				Path:    string(mf.Path),
				Kind:    "managed_drift",
				Message: fmt.Sprintf("digest mismatch: actual=%s expected=%s", actual, mf.Digest),
			})
		}
	}
	// Managed-path set check: lock must record every canonical managed
	// path. Symmetric detection of missing entries.
	wantManaged := make(map[TargetPath]struct{})
	for p, e := range desired {
		if e.Ownership == OwnershipManaged {
			wantManaged[p] = struct{}{}
		}
	}
	for _, mf := range lock.ManagedFiles {
		delete(wantManaged, mf.Path)
	}
	missing := make([]TargetPath, 0, len(wantManaged))
	for p := range wantManaged {
		missing = append(missing, p)
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
	for _, p := range missing {
		result.OK = false
		result.Findings = append(result.Findings, VerifyFinding{
			Path:    string(p),
			Kind:    "lock_missing_entry",
			Message: "canonical desired managed file is not recorded in the lock",
		})
	}
	// Seeded-path set check: lock must record exactly the canonical
	// seeded path set.
	wantSeeded := make(map[TargetPath]struct{})
	for _, s := range profile.Seeds {
		wantSeeded[s.Path] = struct{}{}
	}
	unexpectedSeeded := make([]TargetPath, 0)
	for _, sf := range lock.SeededFiles {
		if _, ok := wantSeeded[sf.Path]; !ok {
			unexpectedSeeded = append(unexpectedSeeded, sf.Path)
		}
		delete(wantSeeded, sf.Path)
	}
	missingSeeded := make([]TargetPath, 0, len(wantSeeded))
	for p := range wantSeeded {
		missingSeeded = append(missingSeeded, p)
	}
	sort.Slice(unexpectedSeeded, func(i, j int) bool { return unexpectedSeeded[i] < unexpectedSeeded[j] })
	sort.Slice(missingSeeded, func(i, j int) bool { return missingSeeded[i] < missingSeeded[j] })
	for _, p := range unexpectedSeeded {
		result.OK = false
		result.Findings = append(result.Findings, VerifyFinding{
			Path:    string(p),
			Kind:    "lock_unexpected_seeded",
			Message: "lock records a seeded file not in the canonical projection",
		})
	}
	for _, p := range missingSeeded {
		result.OK = false
		result.Findings = append(result.Findings, VerifyFinding{
			Path:    string(p),
			Kind:    "lock_missing_seeded",
			Message: "canonical desired seeded file is not recorded in the lock",
		})
	}
	// Compiler compatibility: compare current compiler version to
	// pack.CompilerVersion, the compatibility constraint declared by
	// the canonical pack.
	if err := CheckCompilerCompatibility(pack.CompilerVersion, compilerVersionSource()); err != nil {
		result.OK = false
		result.Findings = append(result.Findings, VerifyFinding{
			Path:    ".factory/doctrine.lock.json",
			Kind:    "compiler_incompatible",
			Message: err.Error(),
		})
	}
	// Observed contracts.
	for _, c := range profile.ObservedContracts {
		switch c.Kind {
		case ObservedMakefileInclude:
			if err := verifyMakefileInclude(resolver, c.Path, c.Id); err != nil {
				result.OK = false
				result.Findings = append(result.Findings, VerifyFinding{
					Path:    string(c.Path),
					Kind:    "observed_contract",
					Message: err.Error(),
				})
			}
		case ObservedMakefileTargetDep:
			if err := verifyMakefileTargetDep(resolver, c.Path, c.Target, c.Dep); err != nil {
				result.OK = false
				result.Findings = append(result.Findings, VerifyFinding{
					Path:    string(c.Path),
					Kind:    "observed_contract",
					Message: err.Error(),
				})
			}
		case ObservedFileExists, ObservedFileDigestEquals:
			// Reserved for future packs; not exercised by factory-core-v1.
		}
	}
	// Exact observed-contract set drift check.
	compareObservedSet(profile, lock, &result)
	sortFindings(result.Findings)
	return result, nil
}

// lockPath returns the absolute path of the doctrine lock.
func lockPath(resolver *Resolver) string {
	return resolver.Resolve(TargetPath(".factory/doctrine.lock.json"))
}

// sortFindings sorts findings deterministically by path, kind, message.
func sortFindings(findings []VerifyFinding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		return findings[i].Message < findings[j].Message
	})
}

// compareObservedSet asserts that the lock records exactly the
// observed-contract declarations declared by the active profile.
func compareObservedSet(profile Profile, lock *LockFile, result *VerifyResult) {
	want := make(map[string]ObservedContractEntry, len(profile.ObservedContracts))
	for _, c := range profile.ObservedContracts {
		want[c.Id] = ObservedContractEntry{
			ID:     c.Id,
			Kind:   string(c.Kind),
			Path:   string(c.Path),
			Target: c.Target,
			Dep:    c.Dep,
		}
	}
	got := make(map[string]ObservedContractEntry, len(lock.Observed))
	for _, e := range lock.Observed {
		got[e.ID] = e
	}
	for id, w := range want {
		g, ok := got[id]
		if !ok {
			result.OK = false
			result.Findings = append(result.Findings, VerifyFinding{
				Path:    ".factory/doctrine.lock.json",
				Kind:    "observed_contract_missing",
				Message: fmt.Sprintf("observed contract %q missing from lock", id),
			})
			continue
		}
		if g != w {
			result.OK = false
			result.Findings = append(result.Findings, VerifyFinding{
				Path:    ".factory/doctrine.lock.json",
				Kind:    "observed_contract_drift",
				Message: fmt.Sprintf("observed contract %q drift: lock=%+v canonical=%+v", id, g, w),
			})
		}
	}
	for id := range got {
		if _, ok := want[id]; !ok {
			result.OK = false
			result.Findings = append(result.Findings, VerifyFinding{
				Path:    ".factory/doctrine.lock.json",
				Kind:    "observed_contract_unexpected",
				Message: fmt.Sprintf("observed contract %q present in lock but absent from profile", id),
			})
		}
	}
}
