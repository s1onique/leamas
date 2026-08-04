// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverProductionFilesRepoWide walks the repo and returns eligible files.
// Fails closed on traversal errors.
func (p *DupcodeBypassPolicy) DiscoverProductionFilesRepoWide() ([]string, error) {
	var files []string

	err := filepath.WalkDir(p.repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "testdata" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel, err := filepath.Rel(p.repoRoot, path)
		if err != nil {
			return fmt.Errorf("rel %s: %w", path, err)
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	sort.Strings(files)
	return files, nil
}
