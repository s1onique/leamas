// SPDX-License-Identifier: Apache-2.0

// Package authority: version.go is a thin shim that reads the
// running binary's commit from the version package. The function is
// isolated so tests can swap the source via VersionReader.
package authority

import (
	"sync"

	"github.com/s1onique/leamas/internal/version"
)

// VersionReader returns the binary's embedded VCS commit. The
// default implementation reads from internal/version; tests can
// override VersionReader to inject fixtures.
var VersionReader = defaultVersionReader

var versionMu sync.Mutex

// readVersionCommit is a small wrapper used by checker.go. It is
// separate from VersionReader so tests can stub it without touching
// the public hook.
func readVersionCommit() string {
	versionMu.Lock()
	defer versionMu.Unlock()
	return VersionReader()
}

func defaultVersionReader() string {
	return version.Get().Commit
}
