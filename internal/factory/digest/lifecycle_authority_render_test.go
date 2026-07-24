// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/authority"
)

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
