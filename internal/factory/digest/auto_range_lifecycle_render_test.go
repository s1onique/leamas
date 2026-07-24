// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// auto_range_lifecycle_render_test.go covers the lifecycle metadata
// rendering required by the digest contract.
package digest

import (
	"strings"
	"testing"
)

// TestRenderLifecycleContainsAllFields verifies the rendered
// lifecycle section contains the documented field names.
func TestRenderLifecycleContainsAllFields(t *testing.T) {
	r := &ResolvedMode{
		AutoRangeStrategy: StrategyActCommitTrailers,
		ActID:             "ACT-X",
		BaseCommit:        "abc123",
		HeadCommit:        "def456",
		Reason:            "test reason",
		LifecycleSubject:  "subject-oid",
		LifecycleClosure:  "closure-oid",
		IncludedCommits:   []string{"c1", "c2"},
		GeneratorCommit:   "gen",
		GeneratorStale:    false,
	}
	out := RenderLifecycle(r)
	for _, field := range []string{
		LifecycleFieldAutoRangeStrategy,
		LifecycleFieldActID,
		LifecycleFieldRangeBase,
		LifecycleFieldRangeHead,
		LifecycleFieldRangeReason,
		LifecycleFieldFreeze,
		LifecycleFieldSubject,
		LifecycleFieldClosure,
		LifecycleFieldIncludedCommits,
		LifecycleFieldGeneratorCommit,
		LifecycleFieldRepositoryHead,
		LifecycleFieldGeneratorStale,
	} {
		if !strings.Contains(out, field+":") {
			t.Errorf("missing %q in lifecycle section:\n%s", field, out)
		}
	}
}
