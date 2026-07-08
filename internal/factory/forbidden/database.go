package forbidden

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// dbImportPatterns lists database import patterns to detect.
var dbImportPatterns = []struct {
	Pattern string
	Desc    string
}{
	{`"database/sql"`, "database/sql import"},
	{`"github.com/lib/pq"`, "lib/pq PostgreSQL driver"},
	{`"github.com/go-sql-driver/mysql"`, "MySQL driver"},
	{`"github.com/go-sql-driver/sqlite"`, "SQLite driver"},
	{`"github.com/mattn/go-sqlite3"`, "go-sqlite3 driver"},
}

// CheckDatabaseImports checks for database driver imports in cmd/.
func CheckDatabaseImports(root string) []checks.Finding {
	var findings []checks.Finding

	scanDirs := []string{"cmd"}
	for _, dir := range scanDirs {
		scanPath := filepath.Join(root, dir)
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.WalkDir(scanPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if name == "vendor" || name == ".git" || name == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			relPath, _ := filepath.Rel(root, path)
			content := string(data)
			for _, imp := range dbImportPatterns {
				if strings.Contains(content, imp.Pattern) {
					findings = append(findings, checks.Finding{
						Path:     relPath,
						Kind:     "forbidden_import",
						Message:  "database driver import: " + imp.Desc,
						Severity: checks.SeverityError,
					})
				}
			}
			return nil
		})
		if err != nil {
			continue
		}
	}

	checks.SortFindings(findings)
	return findings
}
