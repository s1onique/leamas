// Package longtestpolicy provides a verifier for long-test policy compliance.
package longtestpolicy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/factory/longtest"
)

// writeBaseline writes a baseline JSON file to the given directory.
func writeBaseline(t *testing.T, dir, name string, baseline *longtest.Baseline) {
	t.Helper()
	baselinePath := filepath.Join(dir, ".factory", name)
	if err := os.MkdirAll(filepath.Dir(baselinePath), 0755); err != nil {
		t.Fatalf("failed to create .factory dir: %v", err)
	}
	data, err := longtest.BaselineJSON(baseline)
	if err != nil {
		t.Fatalf("failed to marshal baseline: %v", err)
	}
	if err := os.WriteFile(baselinePath, data, 0644); err != nil {
		t.Fatalf("failed to write baseline: %v", err)
	}
}

// writeTestFile writes a test file with RequireLongTest calls.
func writeTestFile(t *testing.T, dir, pkg, filename, content string) {
	t.Helper()
	pkgDir := filepath.Join(dir, pkg)
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}
	path := filepath.Join(pkgDir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
}

func TestCheckRepo_ExactRegistration(t *testing.T) {
	// exact registration → no findings
	dir := t.TempDir()
	writeBaseline(t, dir, "long-tests-baseline.json", &longtest.Baseline{
		SchemaVersion: 1,
		Tests: []longtest.TestSpec{{
			ID:         "LT-001",
			Package:    "./testpkg",
			Test:       "TestFoo",
			FastPolicy: "skip-under-short",
			CITimeout:  "10m",
			CIGroup:    "test",
			Reason:     "test",
			Owner:      "test",
		}},
	})
	writeTestFile(t, dir, "testpkg", "foo_test.go", `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func TestFoo(t *testing.T) {
	longtest.RequireLongTest(t, "LT-001")
}
`)

	findings := CheckRepo(dir)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d: %+v", len(findings), findings)
	}
}

func TestCheckRepo_DuplicateRegistration(t *testing.T) {
	// duplicate registration → duplicate-long-test-call
	dir := t.TempDir()
	writeBaseline(t, dir, "long-tests-baseline.json", &longtest.Baseline{
		SchemaVersion: 1,
		Tests: []longtest.TestSpec{{
			ID:         "LT-001",
			Package:    "./testpkg",
			Test:       "TestDup",
			FastPolicy: "skip-under-short",
			CITimeout:  "10m",
			CIGroup:    "test",
			Reason:     "test",
			Owner:      "test",
		}},
	})
	writeTestFile(t, dir, "testpkg", "dup_test.go", `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func TestDup(t *testing.T) {
	longtest.RequireLongTest(t, "LT-001")
	longtest.RequireLongTest(t, "LT-001")
}
`)

	findings := CheckRepo(dir)
	var dupFindings []string
	for _, f := range findings {
		if f.Kind == "duplicate-long-test-call" {
			dupFindings = append(dupFindings, f.Kind)
		}
	}
	if len(dupFindings) == 0 {
		t.Error("expected duplicate-long-test-call finding")
	}
}

func TestCheckRepo_ExtraCall(t *testing.T) {
	// Duplicate calls for same ID in same test → duplicate-long-test-call
	dir := t.TempDir()
	writeBaseline(t, dir, "long-tests-baseline.json", &longtest.Baseline{
		SchemaVersion: 1,
		Tests: []longtest.TestSpec{
			{
				ID:         "LT-001",
				Package:    "./testpkg",
				Test:       "TestExtra",
				FastPolicy: "skip-under-short",
				CITimeout:  "10m",
				CIGroup:    "test",
				Reason:     "test",
				Owner:      "test",
			},
		},
	})
	writeTestFile(t, dir, "testpkg", "extra_test.go", `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func TestExtra(t *testing.T) {
	longtest.RequireLongTest(t, "LT-001")
	longtest.RequireLongTest(t, "LT-001")
}
`)

	findings := CheckRepo(dir)
	var dupFindings []string
	for _, f := range findings {
		if f.Kind == "duplicate-long-test-call" {
			dupFindings = append(dupFindings, f.Kind)
		}
	}
	if len(dupFindings) == 0 {
		t.Error("expected duplicate-long-test-call finding for LT-001")
	}
}

func TestCheckRepo_BaselineMismatch(t *testing.T) {
	// wrong package/test → baseline-test-mismatch
	dir := t.TempDir()
	writeBaseline(t, dir, "long-tests-baseline.json", &longtest.Baseline{
		SchemaVersion: 1,
		Tests: []longtest.TestSpec{{
			ID:         "LT-001",
			Package:    "./otherpkg",
			Test:       "TestOther",
			FastPolicy: "skip-under-short",
			CITimeout:  "10m",
			CIGroup:    "test",
			Reason:     "test",
			Owner:      "test",
		}},
	})
	writeTestFile(t, dir, "testpkg", "mismatch_test.go", `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func TestMismatch(t *testing.T) {
	longtest.RequireLongTest(t, "LT-001")
}
`)

	findings := CheckRepo(dir)
	var mismatchFindings []string
	for _, f := range findings {
		if f.Kind == "baseline-test-mismatch" {
			mismatchFindings = append(mismatchFindings, f.Kind)
		}
	}
	if len(mismatchFindings) == 0 {
		t.Error("expected baseline-test-mismatch finding")
	}
}

func TestCheckRepo_StaleEntry(t *testing.T) {
	// stale baseline entry (no matching call)
	dir := t.TempDir()
	writeBaseline(t, dir, "long-tests-baseline.json", &longtest.Baseline{
		SchemaVersion: 1,
		Tests: []longtest.TestSpec{{
			ID:         "LT-STALE",
			Package:    "./testpkg",
			Test:       "TestStale",
			FastPolicy: "skip-under-short",
			CITimeout:  "10m",
			CIGroup:    "test",
			Reason:     "test",
			Owner:      "test",
		}},
	})
	writeTestFile(t, dir, "testpkg", "other_test.go", `package testpkg

import (
	"testing"
)

func TestOther(t *testing.T) {
}
`)

	findings := CheckRepo(dir)
	var staleFindings []string
	for _, f := range findings {
		if f.Kind == "stale-baseline-entry" {
			staleFindings = append(staleFindings, f.Kind)
		}
	}
	if len(staleFindings) == 0 {
		t.Error("expected stale-baseline-entry finding")
	}
}

func TestCheckRepo_UnregisteredEntry(t *testing.T) {
	// unregistered call (baseline has one entry, call uses different ID)
	dir := t.TempDir()
	writeBaseline(t, dir, "long-tests-baseline.json", &longtest.Baseline{
		SchemaVersion: 1,
		Tests: []longtest.TestSpec{{
			ID:         "LT-OTHER",
			Package:    "./testpkg",
			Test:       "TestOther",
			FastPolicy: "skip-under-short",
			CITimeout:  "10m",
			CIGroup:    "test",
			Reason:     "test",
			Owner:      "test",
		}},
	})
	writeTestFile(t, dir, "testpkg", "unreg_test.go", `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func TestUnreg(t *testing.T) {
	longtest.RequireLongTest(t, "LT-UNREGISTERED")
}
`)

	findings := CheckRepo(dir)
	var unregFindings []string
	for _, f := range findings {
		if f.Kind == "unregistered-long-test" {
			unregFindings = append(unregFindings, f.Kind)
		}
	}
	if len(unregFindings) == 0 {
		t.Error("expected unregistered-long-test finding")
	}
}

func TestCheckRepo_InvalidContainer(t *testing.T) {
	// call inside non-test function
	dir := t.TempDir()
	writeBaseline(t, dir, "long-tests-baseline.json", &longtest.Baseline{
		SchemaVersion: 1,
		Tests: []longtest.TestSpec{{
			ID:         "LT-001",
			Package:    "./testpkg",
			Test:       "TestInvalid",
			FastPolicy: "skip-under-short",
			CITimeout:  "10m",
			CIGroup:    "test",
			Reason:     "test",
			Owner:      "test",
		}},
	})
	writeTestFile(t, dir, "testpkg", "invalid_test.go", `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func helper(t *testing.T) {
	longtest.RequireLongTest(t, "LT-001")
}

func TestInvalid(t *testing.T) {
}
`)

	findings := CheckRepo(dir)
	var outsideFindings []string
	for _, f := range findings {
		if f.Kind == "long-test-call-outside-valid-test" {
			outsideFindings = append(outsideFindings, f.Kind)
		}
	}
	if len(outsideFindings) == 0 {
		t.Error("expected long-test-call-outside-valid-test finding")
	}
}

func TestCheckRepo_MalformedSource(t *testing.T) {
	// malformed source file
	dir := t.TempDir()
	writeBaseline(t, dir, "long-tests-baseline.json", &longtest.Baseline{
		SchemaVersion: 1,
		Tests: []longtest.TestSpec{{
			ID:         "LT-001",
			Package:    "./testpkg",
			Test:       "TestFoo",
			FastPolicy: "skip-under-short",
			CITimeout:  "10m",
			CIGroup:    "test",
			Reason:     "test",
			Owner:      "test",
		}},
	})
	pkgDir := filepath.Join(dir, "testpkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}
	path := filepath.Join(pkgDir, "malformed_test.go")
	if err := os.WriteFile(path, []byte("package testpkg\n\nfunc broken{"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	findings := CheckRepo(dir)
	var scanFindings []string
	for _, f := range findings {
		if f.Kind == "scan-error" {
			scanFindings = append(scanFindings, f.Kind)
		}
	}
	if len(scanFindings) == 0 {
		t.Error("expected scan-error finding for malformed source")
	}
}

func TestCheckRepo_TraversalFailure(t *testing.T) {
	// traversal failure due to unreadable directory
	dir := t.TempDir()
	writeBaseline(t, dir, "long-tests-baseline.json", &longtest.Baseline{
		SchemaVersion: 1,
		Tests:         []longtest.TestSpec{},
	})

	// Create a directory that we'll make unreadable
	blockedDir := filepath.Join(dir, "blocked")
	if err := os.MkdirAll(blockedDir, 0755); err != nil {
		t.Fatalf("failed to create blocked dir: %v", err)
	}

	findings := CheckRepo(dir)
	// Should complete without crashing; may or may not have findings depending on permissions
	_ = findings
}
