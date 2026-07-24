// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"fmt"
	"github.com/s1onique/leamas/internal/version"
	"strings"
)

// populateGeneratorMetadata stamps generator/freshness fields on the
// resolution. The generator commit is taken from version.Get().Commit;
// the repository HEAD is already known.
func populateGeneratorMetadata(repoRoot string, r *LifecycleResolution) {
	r.GeneratorCommit = version.Get().Commit
	r.RepositoryHead = r.RangeHead
	if r.GeneratorCommit == "" || r.GeneratorCommit == "unknown" {
		r.GeneratorStale = true
		r.StaleReason = "embedded LEAMAS_COMMIT is unknown"
		r.GeneratorIsAncestor = false
		return
	}
	_, err := runGitValueTrimmed(repoRoot, "merge-base", "--is-ancestor", r.GeneratorCommit, r.RepositoryHead)
	if err != nil {
		r.GeneratorIsAncestor = false
		r.GeneratorStale = true
		r.StaleReason = "embedded LEAMAS_COMMIT " + shortSHA(r.GeneratorCommit) +
			" is not an ancestor of repository HEAD " + shortSHA(r.RepositoryHead)
		return
	}
	r.GeneratorIsAncestor = true
	r.GeneratorStale = false
	r.StaleReason = ""
}

// ensureRangeBaseReachable verifies the resolved base commit is
// reachable in the local repository. Shallow clones that lack the
// base commit are detected here.
func ensureRangeBaseReachable(repoRoot string, r *LifecycleResolution) error {
	if r.RangeBase == "" || r.RangeBase == emptyTreeBaseline {
		return nil
	}
	_, err := runGitValueTrimmed(repoRoot, "rev-parse", "--verify", "--end-of-options", r.RangeBase)
	if err != nil {
		return fmt.Errorf("%w: base %s is not reachable; rerun with --range or unshallow the repository",
			ErrShallowBaseline, shortSHA(r.RangeBase))
	}
	return nil
}

// populateIncludedCommits fills the IncludedCommits slice via
// git rev-list base..head.
func populateIncludedCommits(repoRoot string, r *LifecycleResolution) error {
	output, err := runGitOutput(repoRoot, "rev-list", r.RangeBase+".."+r.RangeHead)
	if err != nil {
		if r.RangeBase == emptyTreeBaseline {
			r.IncludedCommits = []string{r.RangeHead}
			return nil
		}
		return fmt.Errorf("rev-list %s: %w", r.Range(), err)
	}
	commits := []string{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			commits = append(commits, line)
		}
	}
	if len(commits) == 0 {
		commits = []string{r.RangeHead}
	}
	r.IncludedCommits = commits
	return nil
}
