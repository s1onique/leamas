// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"github.com/s1onique/leamas/internal/factory/checks"
)

// DupcodeProtectedPrefixes defines protected package prefixes.
var DupcodeProtectedPrefixes = []string{
	"github.com/s1onique/leamas/internal/factory/dupcode",
}

// DupcodeAllowedPaths defines allowed caller package paths.
var DupcodeAllowedPaths = []string{
	"github.com/s1onique/leamas/internal/factory/protectedverifier",
	"github.com/s1onique/leamas/internal/factory/dupcode",
}

// CheckDupcodeBypass is the legacy V1 implementation.
// Deprecated: Use CanonicalCheckDupcodeBypass instead.
func CheckDupcodeBypass(root string) []checks.Finding {
	return nil // Legacy: no findings
}
