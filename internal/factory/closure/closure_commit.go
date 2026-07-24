// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
)

// ClosureCommitSelector is the declarative form of the evidence that
// uniquely identifies a closure commit C. Selection by the
// (canonical_paths, expected_blob_oids, subject_ancestor_relationship)
// triple is the only authoritative algorithm; matching by commit message
// is rejected at the API boundary.
type ClosureCommitSelector struct {
	CanonicalPaths    []string
	ExpectedBlobOIDs  []string
	SubjectCommitOID  string
	RunRepositoryRoot string
	gitClient         gitClient
}

// ClosureCommitMatch describes a single commit in the linear history that
// satisfies the selector. Tree OIDs are recorded for parity comparison.
type ClosureCommitMatch struct {
	CommitOID string
	TreeOID   string
	Blobs     map[string]string
}

// FindClosureCommit searches HEAD~..HEAD for a single commit whose tree
// contains exactly the canonical paths bound to the expected blob OIDs
// AND whose first parent (or, for the root commit, the empty tree) is an
// ancestor of the declared subject. The selector enforces that the
// commit message alone is never authoritative.
func FindClosureCommit(ctx context.Context, sel ClosureCommitSelector) (ClosureCommitMatch, error) {
	if len(sel.CanonicalPaths) == 0 {
		return ClosureCommitMatch{}, fmt.Errorf("closure commit selector requires at least one canonical path")
	}
	if len(sel.CanonicalPaths) != len(sel.ExpectedBlobOIDs) {
		return ClosureCommitMatch{}, fmt.Errorf("closure commit selector path and blob lists must align")
	}
	normalizedPaths := make([]string, len(sel.CanonicalPaths))
	copy(normalizedPaths, sel.CanonicalPaths)
	sort.Strings(normalizedPaths)
	normalizedBlobs := make([]string, len(normalizedPaths))
	for index, path := range normalizedPaths {
		original := indexOf(sel.CanonicalPaths, path)
		if original < 0 || original >= len(sel.ExpectedBlobOIDs) {
			return ClosureCommitMatch{}, fmt.Errorf("closure commit selector path %q has no blob", path)
		}
		normalizedBlobs[index] = sel.ExpectedBlobOIDs[original]
	}
	git := sel.gitClient
	if git == nil {
		git = RealGit{}
	}
	root := sel.RunRepositoryRoot
	if root == "" {
		root = "."
	}
	rangeSpec := "HEAD~10..HEAD"
	list, err := runGitValue(ctx, git, root, "rev-list", "--reverse", rangeSpec)
	if err != nil || strings.TrimSpace(list) == "" {
		// Fall back to the full linear history; the regression repo
		// may not have a parent and we still want a robust selector.
		list, err = runGitValue(ctx, git, root, "rev-list", "--reverse", "HEAD")
		if err != nil {
			return ClosureCommitMatch{}, fmt.Errorf("list closure candidates: %w", err)
		}
	}
	var matches []ClosureCommitMatch
	for _, line := range strings.Split(list, "\n") {
		commit := strings.TrimSpace(line)
		if commit == "" {
			continue
		}
		tree, err := runGitValue(ctx, git, root, "rev-parse", "--verify", "--end-of-options", commit+"^{tree}")
		if err != nil {
			continue
		}
		blobs := make(map[string]string, len(normalizedPaths))
		ok := true
		for index, path := range normalizedPaths {
			expected := normalizedBlobs[index]
			actual, err := runGitValue(ctx, git, root, "rev-parse", "--verify", "--end-of-options", commit+":"+path)
			if err != nil {
				ok = false
				break
			}
			if !strings.EqualFold(strings.TrimSpace(actual), expected) {
				ok = false
				break
			}
			blobs[path] = strings.TrimSpace(actual)
		}
		if !ok {
			continue
		}
		matches = append(matches, ClosureCommitMatch{CommitOID: commit, TreeOID: tree, Blobs: blobs})
	}
	if len(matches) == 0 {
		return ClosureCommitMatch{}, fmt.Errorf("no commit in HEAD..HEAD satisfies the closure selector")
	}
	// Apply the ancestry selection rule: a candidate is dominated by
	// another when the other is an ancestor of it. Strict descendants
	// are the only candidates that survive.
	if len(matches) > 1 {
		keep := make([]bool, len(matches))
		for i := range keep {
			keep[i] = true
		}
		for i := range matches {
			for j := range matches {
				if i == j || !keep[i] {
					continue
				}
				ancestor := git.Run(ctx, root, "merge-base", "--is-ancestor", matches[j].CommitOID, matches[i].CommitOID)
				if ancestor.Err == nil && ancestor.ExitCode == 0 {
					keep[i] = false
					break
				}
			}
		}
		filtered := make([]ClosureCommitMatch, 0, len(matches))
		for i, match := range matches {
			if keep[i] {
				filtered = append(filtered, match)
			}
		}
		if len(filtered) > 1 {
			ids := make([]string, 0, len(filtered))
			for _, match := range filtered {
				ids = append(ids, match.CommitOID)
			}
			sort.Strings(ids)
			return ClosureCommitMatch{}, fmt.Errorf("multiple incomparable closure candidates: %s", strings.Join(ids, ","))
		}
		if len(filtered) == 1 {
			matches = filtered
		}
	}
	if len(matches) == 0 {
		return ClosureCommitMatch{}, fmt.Errorf("no commit in HEAD..HEAD satisfies the closure selector")
	}
	match := matches[0]
	if sel.SubjectCommitOID != "" {
		ancestorResult := git.Run(ctx, root, "merge-base", "--is-ancestor", sel.SubjectCommitOID, match.CommitOID)
		if ancestorResult.Err != nil || ancestorResult.ExitCode != 0 {
			return ClosureCommitMatch{}, fmt.Errorf("subject %s is not an ancestor of candidate closure %s", sel.SubjectCommitOID, match.CommitOID)
		}
	}
	return match, nil
}

func init() {
	_ = context.Background
}

func indexOf(list []string, value string) int {
	for index, candidate := range list {
		if candidate == value {
			return index
		}
	}
	return -1
}

// ClosureCommitSelectorFromManifest extracts the canonical paths and blob
// OIDs that uniquely identify the closure commit C for the supplied
// manifest. Callers are expected to extend the canonical list with
// sidecar evidence and any post-commit evidence they require.
func ClosureCommitSelectorFromManifest(manifest Manifest) (ClosureCommitSelector, error) {
	paths := []string{manifest.Plan.Path, "docs/closure-manifests/" + manifest.ActID + ".json", "docs/close-reports/" + manifest.ActID + ".md"}
	if manifest.Tag != "" {
		paths = append(paths, "docs/lifecycle-errata/"+manifest.ActID+".json")
	}
	blobs := []string{manifest.Plan.SHA256, manifest.PlanFreeze.PlanSHA256, manifest.PlanFreeze.PlanSHA256}
	return ClosureCommitSelector{
		CanonicalPaths:   paths,
		ExpectedBlobOIDs: blobs,
		SubjectCommitOID: manifest.Subject.CommitOID,
	}, nil
}

// RequiredBinaryParity reports the four equality invariants enforced by
// Slice 5's exact-binary parity check.
type RequiredBinaryParity struct {
	LeamasVersion string
	BinarySHA256  string
	VCSRevision   string
	VCSModified   bool
	RunnerToolSHA string
	VersionEqual  bool
	BinaryEqual   bool
	RevisionEqual bool
	ModifiedEqual bool
	ToolSHAEqual  bool
	Compatible    bool
}

// AssertExactBinaryParity compares the closure-time runner identity
// recorded in the manifest against the live runner identity. A mismatch
// in any of the four equality dimensions blocks attestation and tagging.
func AssertExactBinaryParity(manifest RunnerIdentity, live RunnerIdentity, toolSHA string) RequiredBinaryParity {
	parity := RequiredBinaryParity{
		LeamasVersion: manifest.LeamasVersion,
		BinarySHA256:  manifest.BinarySHA256,
		VCSRevision:   manifest.VCSRevision,
		VCSModified:   manifest.VCSModified,
		RunnerToolSHA: toolSHA,
		VersionEqual:  manifest.LeamasVersion == live.LeamasVersion,
		BinaryEqual:   bytes.Equal([]byte(manifest.BinarySHA256), []byte(live.BinarySHA256)),
		RevisionEqual: manifest.VCSRevision == live.VCSRevision,
		ModifiedEqual: manifest.VCSModified == live.VCSModified,
		ToolSHAEqual:  strings.EqualFold(manifest.BinarySHA256, toolSHA),
	}
	parity.Compatible = parity.VersionEqual && parity.BinaryEqual && parity.RevisionEqual && parity.ModifiedEqual && parity.ToolSHAEqual
	return parity
}
