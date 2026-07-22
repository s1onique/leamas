// Package gate provides test helpers for dupcode shared analysis tests.
package gate

import (
	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// testInput creates a DupcodeInput with the given config values.
// It canonicalizes the config using effectiveDupcodeConfig.
func testInput(root string, minLines, minTokens int, excludeDirs, excludeSuffixes []string, ignoreGenerated bool) DupcodeInput {
	return newDupcodeInput(dupcode.Config{
		Root:                root,
		MinLines:            minLines,
		MinTokens:           minTokens,
		ExcludeDirs:         excludeDirs,
		ExcludeFileSuffixes: excludeSuffixes,
		IgnoreGenerated:     ignoreGenerated,
	})
}
