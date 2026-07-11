package doctrinecompiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// LockFile is the decoded contents of .factory/doctrine.lock.json.
//
// The file records the exact managed projection at the time the lock
// was last written. The verifier compares this against the current
// projection to detect drift.
type LockFile struct {
	SchemaVersion   int                     `json:"schema_version"`
	PackID          string                  `json:"pack_id"`
	PackVersion     string                  `json:"pack_version"`
	PackDigest      string                  `json:"pack_digest"`
	ProfileID       string                  `json:"profile_id"`
	CompilerVersion string                  `json:"compiler_version"`
	CompilerCommit  string                  `json:"compiler_commit"`
	ManagedFiles    []ManagedFileEntry      `json:"managed_files"`
	SeededFiles     []SeededFileEntry       `json:"seeded_files"`
	Observed        []ObservedContractEntry `json:"observed_contracts"`
}

// ManagedFileEntry records one managed file's path and digest.
type ManagedFileEntry struct {
	Path   TargetPath    `json:"path"`
	Digest ContentDigest `json:"digest"`
}

// SeededFileEntry records one seeded file's path.
//
// Seeded files are owned by the target repository after their first
// creation; only the path is recorded for reference.
type SeededFileEntry struct {
	Path TargetPath `json:"path"`
}

// ObservedContractEntry records the existence of one observed contract
// declaration in the lock.
type ObservedContractEntry struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Path   string `json:"path,omitempty"`
	Target string `json:"target,omitempty"`
	Dep    string `json:"dependency,omitempty"`
}

// ReadLockFile decodes a lock file from disk and validates it for
// exactness.
//
// All path entries are passed through NormalizeTargetPath so that any
// traversal segment in a corrupted lock file is rejected before it
// can be used to address a filesystem path. This is a hard
// security boundary: the planner and verifier never trust a lock
// entry's path component without normalisation.
//
// In addition, the lock is rejected if any of the following hold:
//
//   - duplicate normalized paths in managed_files
//   - duplicate normalized paths in seeded_files
//   - a normalized path appearing in both managed_files and seeded_files
//   - duplicate IDs in observed_contracts
//   - empty managed or seeded paths
//   - empty observed-contract IDs
//
// Duplicate detection runs after NormalizeTargetPath, ensuring
// alternate accepted spellings cannot bypass duplicate detection.
func ReadLockFile(path string) (*LockFile, error) {
	data, err := readAllFile(path)
	if err != nil {
		return nil, newError("verify", "lock", err.Error())
	}
	var lf LockFile
	if err := strictDecode(bytes.NewReader(data), &lf); err != nil {
		return nil, newError("verify", "lock", err.Error())
	}
	if lf.SchemaVersion != LockSchemaVersion {
		return nil, newError("verify", "lock",
			fmt.Sprintf("unsupported lock schema_version %d", lf.SchemaVersion))
	}
	// Normalize every path entry first; duplicate detection then runs
	// on the canonical spelling so alternates cannot bypass it.
	for i, mf := range lf.ManagedFiles {
		if strings.TrimSpace(string(mf.Path)) == "" {
			return nil, newError("verify", "lock",
				fmt.Sprintf("managed_files[%d].path is empty", i))
		}
		tp, err := NormalizeTargetPath(string(mf.Path))
		if err != nil {
			return nil, newError("verify", "lock",
				fmt.Sprintf("managed_files[%d].path invalid: %v", i, err))
		}
		lf.ManagedFiles[i].Path = tp
	}
	for i, sf := range lf.SeededFiles {
		if strings.TrimSpace(string(sf.Path)) == "" {
			return nil, newError("verify", "lock",
				fmt.Sprintf("seeded_files[%d].path is empty", i))
		}
		tp, err := NormalizeTargetPath(string(sf.Path))
		if err != nil {
			return nil, newError("verify", "lock",
				fmt.Sprintf("seeded_files[%d].path invalid: %v", i, err))
		}
		lf.SeededFiles[i].Path = tp
	}
	if err := validateLockExactness(&lf); err != nil {
		return nil, newError("verify", "lock", err.Error())
	}
	return &lf, nil
}

// validateLockExactness rejects ambiguous or duplicate entries in a
// decoded lock. The function is total: it always inspects every
// section of the lock.
func validateLockExactness(lf *LockFile) error {
	seenManaged := make(map[TargetPath]int, len(lf.ManagedFiles))
	for i, mf := range lf.ManagedFiles {
		if _, dup := seenManaged[mf.Path]; dup {
			return fmt.Errorf("duplicate normalized managed path %q at index %d (first seen at index %d)",
				mf.Path, i, seenManaged[mf.Path])
		}
		seenManaged[mf.Path] = i
	}
	seenSeeded := make(map[TargetPath]int, len(lf.SeededFiles))
	for i, sf := range lf.SeededFiles {
		if _, dup := seenSeeded[sf.Path]; dup {
			return fmt.Errorf("duplicate normalized seeded path %q at index %d (first seen at index %d)",
				sf.Path, i, seenSeeded[sf.Path])
		}
		seenSeeded[sf.Path] = i
	}
	// Cross-ownership collision: a path may appear in both managed
	// and seeded lists, which is always ambiguous regardless of
	// intent. Identify the first such conflict deterministically.
	type conflict struct {
		path    TargetPath
		managed int
		seeded  int
	}
	var conflicts []conflict
	for path, mi := range seenManaged {
		if si, ok := seenSeeded[path]; ok {
			conflicts = append(conflicts, conflict{path: path, managed: mi, seeded: si})
		}
	}
	if len(conflicts) > 0 {
		sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].path < conflicts[j].path })
		first := conflicts[0]
		return fmt.Errorf("path %q appears in both managed_files (index %d) and seeded_files (index %d); cross-ownership collision",
			first.path, first.managed, first.seeded)
	}
	seenObs := make(map[string]int, len(lf.Observed))
	for i, oc := range lf.Observed {
		if strings.TrimSpace(oc.ID) == "" {
			return fmt.Errorf("observed_contracts[%d].id is empty", i)
		}
		if _, dup := seenObs[oc.ID]; dup {
			return fmt.Errorf("duplicate observed-contract id %q at index %d (first seen at index %d)",
				oc.ID, i, seenObs[oc.ID])
		}
		seenObs[oc.ID] = i
	}
	return nil
}

// FormatLockFile renders a LockFile to deterministic canonical JSON.
//
// The encoder sorts managed files, seeded files, and observed
// contracts lexicographically by their stable sort key. Map-iteration
// effects do not influence the output.
func FormatLockFile(lf LockFile) ([]byte, error) {
	// Sort copies so we don't mutate caller state.
	mfs := append([]ManagedFileEntry(nil), lf.ManagedFiles...)
	sort.Slice(mfs, func(i, j int) bool { return mfs[i].Path < mfs[j].Path })
	sfs := append([]SeededFileEntry(nil), lf.SeededFiles...)
	sort.Slice(sfs, func(i, j int) bool { return sfs[i].Path < sfs[j].Path })
	obs := append([]ObservedContractEntry(nil), lf.Observed...)
	sort.Slice(obs, func(i, j int) bool { return obs[i].ID < obs[j].ID })
	out := LockFile{
		SchemaVersion:   lf.SchemaVersion,
		PackID:          lf.PackID,
		PackVersion:     lf.PackVersion,
		PackDigest:      lf.PackDigest,
		ProfileID:       lf.ProfileID,
		CompilerVersion: lf.CompilerVersion,
		CompilerCommit:  lf.CompilerCommit,
		ManagedFiles:    mfs,
		SeededFiles:     sfs,
		Observed:        obs,
	}
	buf, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	out2 := append([]byte(nil), buf...)
	out2 = append(out2, '\n')
	return out2, nil
}

// BuildLockFromPlan materialises a LockFile from a ProjectionPlan and
// the desired entries.
func BuildLockFromPlan(plan ProjectionPlan, managed []ProjectionEntry, seeded []TargetPath, observed []ObservedContract, compilerCommit string) LockFile {
	mf := make([]ManagedFileEntry, 0, len(managed))
	for _, e := range managed {
		mf = append(mf, ManagedFileEntry{
			Path:   e.Path,
			Digest: e.Digest,
		})
	}
	sf := make([]SeededFileEntry, 0, len(seeded))
	for _, p := range seeded {
		sf = append(sf, SeededFileEntry{Path: p})
	}
	obs := make([]ObservedContractEntry, 0, len(observed))
	for _, c := range observed {
		obs = append(obs, ObservedContractEntry{
			ID:     c.Id,
			Kind:   string(c.Kind),
			Path:   string(c.Path),
			Target: c.Target,
			Dep:    c.Dep,
		})
	}
	return LockFile{
		SchemaVersion:   LockSchemaVersion,
		PackID:          string(plan.PackId),
		PackVersion:     string(plan.PackVersion),
		PackDigest:      string(plan.PackDigest),
		ProfileID:       string(plan.ProfileId),
		CompilerVersion: plan.CompilerVersion,
		CompilerCommit:  compilerCommit,
		ManagedFiles:    mf,
		SeededFiles:     sf,
		Observed:        obs,
	}
}

// readAllFile reads the entire file at path.
func readAllFile(path string) ([]byte, error) {
	return readFS(path)
}
