// Package longtestpolicy provides a verifier for long-test policy compliance.
package longtestpolicy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

type entryKey struct {
	ID      string
	Package string
	Test    string
}

func CheckRepo(root string) []checks.Finding {
	var findings []checks.Finding

	baseline, err := longtest.LoadBaseline(root)
	if err != nil {
		return appendFinding(findings, ".factory/long-tests-baseline.json", "baseline-error", err.Error())
	}
	if err := longtest.ValidateBaseline(baseline); err != nil {
		return appendFinding(findings, ".factory/long-tests-baseline.json", "baseline-validation", err.Error())
	}

	callsByKey := make(map[entryKey][]CallSite)
	callsByID := make(map[string][]CallSite)
	var scanErrs []string

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			findings = append(findings, checks.Finding{
				Path: path, Kind: "traversal-error", Message: err.Error(),
			})
			scanErrs = append(scanErrs, path)
			return nil
		}
		if info.IsDir() {
			if path == root {
				return nil
			}
			base := filepath.Base(path)
			if base == "vendor" || base == "testdata" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") || strings.Contains(path, "/testdata/") {
			return nil
		}

		relDir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			findings = append(findings, checks.Finding{
				Path: path, Kind: "path-error", Message: err.Error(),
			})
			return nil
		}
		pkgPath := "./" + filepath.ToSlash(relDir)

		result, err := scanTestFileAST(path, pkgPath)
		if err != nil {
			findings = append(findings, checks.Finding{
				Path: path, Kind: "scan-error", Message: err.Error(),
			})
			return nil
		}

		for _, call := range result.Malformed {
			kind := "invalid-require-longtest-call"
			msg := fmt.Sprintf("RequireLongTest at line %d has invalid arguments (test=%s)", call.Line, call.TestFunc)
			findings = append(findings, checks.Finding{Path: call.File, Kind: kind, Message: msg})
		}

		for _, call := range result.LiteralCalls {
			if !call.ValidTest {
				findings = append(findings, checks.Finding{
					Path: call.File, Kind: "long-test-call-outside-valid-test",
					Message: fmt.Sprintf("RequireLongTest at line %d is inside %q, which is not a valid Go test", call.Line, call.TestFunc),
				})
				continue
			}
			key := entryKey{ID: call.ID, Package: call.PkgPath, Test: call.TestFunc}
			callsByKey[key] = append(callsByKey[key], call)
			callsByID[call.ID] = append(callsByID[call.ID], call)
		}
		return nil
	})

	if len(scanErrs) > 0 {
		findings = append(findings, checks.Finding{
			Path: ".", Kind: "scan-incomplete", Message: fmt.Sprintf("scan errors in %d paths", len(scanErrs)),
		})
	}

	for _, tt := range baseline.Tests {
		key := entryKey{ID: tt.ID, Package: tt.Package, Test: tt.Test}
		exactSites := callsByKey[key]
		idSites := callsByID[tt.ID]

		switch {
		case len(exactSites) > 1:
			for _, site := range exactSites {
				findings = append(findings, checks.Finding{
					Path: site.File, Kind: "duplicate-long-test-call",
					Message: fmt.Sprintf("Multiple calls for ID %q in same package/test (line %d)", tt.ID, site.Line),
				})
			}
		case len(exactSites) == 1 && len(idSites) == 1:
			// Valid exact match
		case len(exactSites) == 1 && len(idSites) > 1:
			// Exact call plus extra unauthorized calls
			for _, site := range idSites {
				if site.File == exactSites[0].File && site.Line == exactSites[0].Line {
					continue
				}
				findings = append(findings, checks.Finding{
					Path: site.File, Kind: "extra-long-test-call",
					Message: fmt.Sprintf("Extra call for ID %q at line %d (baseline expects only one)", tt.ID, site.Line),
				})
			}
		case len(exactSites) == 0 && len(idSites) > 0:
			// ID exists but wrong package or test
			for _, site := range idSites {
				findings = append(findings, checks.Finding{
					Path: site.File, Kind: "baseline-test-mismatch",
					Message: fmt.Sprintf("ID %q at line %d but baseline expects %q, test %q", tt.ID, site.Line, tt.Package, tt.Test),
				})
			}
		default:
			findings = append(findings, checks.Finding{
				Path: ".factory/long-tests-baseline.json", Kind: "stale-baseline-entry",
				Message: fmt.Sprintf("ID %q has no RequireLongTest call", tt.ID),
			})
		}
	}

	for id, sites := range callsByID {
		if !isRegistered(id, baseline) {
			for _, site := range sites {
				findings = append(findings, checks.Finding{
					Path: site.File, Kind: "unregistered-long-test",
					Message: fmt.Sprintf("Unregistered ID %q at line %d", id, site.Line),
				})
			}
		}
	}

	return findings
}

func isRegistered(id string, baseline *longtest.Baseline) bool {
	for _, tt := range baseline.Tests {
		if tt.ID == id {
			return true
		}
	}
	return false
}

func appendFinding(findings []checks.Finding, path, kind, msg string) []checks.Finding {
	return append(findings, checks.Finding{Path: path, Kind: kind, Message: msg})
}
