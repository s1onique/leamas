package doctrinecompiler

import "strings"

// stringsContainsPrefix is a small wrapper kept here to avoid an
// extra import in test files that need to detect ".tmp-" prefixes.
func stringsContainsPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}
