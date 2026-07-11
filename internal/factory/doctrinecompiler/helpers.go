package doctrinecompiler

import (
	"os"
	"strings"
)

// stringsContainsPrefix is a small wrapper kept here to avoid an
// extra import in test files that need to detect ".tmp-" prefixes.
func stringsContainsPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// osMkdirAllImpl is a thin wrapper around os.MkdirAll so tests can
// invoke the production syscall via a single name.
func osMkdirAllImpl(p string, mode os.FileMode) error {
	return os.MkdirAll(p, mode)
}

// osWriteFileImpl is a thin wrapper around os.WriteFile so tests can
// invoke the production syscall via a single name.
func osWriteFileImpl(p string, data []byte, mode os.FileMode) error {
	return os.WriteFile(p, data, mode)
}
