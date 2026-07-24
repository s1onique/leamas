// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/authority"
)

func TestExplicitRangeGenerationBindsAuthorityClassification(t *testing.T) {
	repo := t.TempDir()
	initGit(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "seed.txt"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "seed.txt")
	runGit(t, repo, "commit", "-m", "seed")
	if err := os.WriteFile(filepath.Join(repo, "change.txt"), []byte("change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "change.txt")
	runGit(t, repo, "commit", "-m", "change")
	content, err := Generate(Options{RepoRoot: repo, Mode: ModeRange, Range: "HEAD~1..HEAD"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, want := range []string{"AUTHORITY_STATUS: ExplicitRange", "RESOLUTION_SOURCE: explicit_cli"} {
		if !strings.Contains(content, want) {
			t.Fatalf("Generate output missing %q:\n%s", want, content)
		}
	}
}

func TestLifecycleRenderIncludesAuthorityClassification(t *testing.T) {
	rendered := RenderLifecycle(&ResolvedMode{
		AuthorityStatus:  authority.AuthorityExplicitRange,
		ResolutionSource: "explicit_cli",
	})
	for _, want := range []string{
		"AUTHORITY_STATUS: ExplicitRange",
		"RESOLUTION_SOURCE: explicit_cli",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("RenderLifecycle output missing %q:\n%s", want, rendered)
		}
	}
}
