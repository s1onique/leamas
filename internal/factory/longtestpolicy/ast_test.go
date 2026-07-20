// Package longtestpolicy provides a verifier for long-test policy compliance.
package longtestpolicy

import (
	"fmt"
	"os"
	"testing"
)

func TestScanTestFileAST_LiteralCall(t *testing.T) {
	src := `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func TestFoo(t *testing.T) {
	longtest.RequireLongTest(t, "LT-ID-001")
}
`
	result, err := scanTestFileASTFromSrc(t, "testpkg", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Malformed) != 0 {
		t.Errorf("expected no malformed, got %d", len(result.Malformed))
	}
	if len(result.LiteralCalls) != 1 {
		t.Fatalf("expected 1 literal call, got %d", len(result.LiteralCalls))
	}
	call := result.LiteralCalls[0]
	if call.ID != "LT-ID-001" {
		t.Errorf("expected ID LT-ID-001, got %s", call.ID)
	}
	if call.TestFunc != "TestFoo" {
		t.Errorf("expected TestFoo, got %s", call.TestFunc)
	}
	if !call.ValidTest {
		t.Error("expected ValidTest=true")
	}
}

func TestScanTestFileAST_NonliteralID(t *testing.T) {
	src := `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func TestBar(t *testing.T) {
	id := "LT-ID-001"
	longtest.RequireLongTest(t, id)
}
`
	result, err := scanTestFileASTFromSrc(t, "testpkg", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.LiteralCalls) != 0 {
		t.Errorf("expected 0 literal calls, got %d", len(result.LiteralCalls))
	}
	if len(result.Malformed) != 1 {
		t.Errorf("expected 1 malformed, got %d", len(result.Malformed))
	}
}

func TestScanTestFileAST_MissingArgs(t *testing.T) {
	src := `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func TestBaz(t *testing.T) {
	longtest.RequireLongTest(t)
}
`
	result, err := scanTestFileASTFromSrc(t, "testpkg", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.LiteralCalls) != 0 {
		t.Errorf("expected 0 literal calls, got %d", len(result.LiteralCalls))
	}
	if len(result.Malformed) != 1 {
		t.Errorf("expected 1 malformed, got %d", len(result.Malformed))
	}
}

func TestScanTestFileAST_InvalidTestParam(t *testing.T) {
	src := `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func TestInvalid(x int) {
	longtest.RequireLongTest(t, "LT-ID-001")
}
`
	result, err := scanTestFileASTFromSrc(t, "testpkg", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.LiteralCalls) != 1 {
		t.Errorf("expected 1 literal call, got %d", len(result.LiteralCalls))
	}
	if result.LiteralCalls[0].ValidTest {
		t.Error("expected ValidTest=false")
	}
}

func TestScanTestFileAST_Method(t *testing.T) {
	src := `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

type Suite struct{}

func (s *Suite) TestMethod(t *testing.T) {
	longtest.RequireLongTest(t, "LT-ID-001")
}
`
	result, err := scanTestFileASTFromSrc(t, "testpkg", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.LiteralCalls) != 1 {
		t.Errorf("expected 1 literal call, got %d", len(result.LiteralCalls))
	}
	if result.LiteralCalls[0].ValidTest {
		t.Error("expected ValidTest=false for method")
	}
}

func TestScanTestFileAST_HelperFunction(t *testing.T) {
	src := `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func helper(t *testing.T) {
	longtest.RequireLongTest(t, "LT-ID-001")
}

func TestHelper(t *testing.T) {
	helper(t)
}
`
	result, err := scanTestFileASTFromSrc(t, "testpkg", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.LiteralCalls) != 1 {
		t.Errorf("expected 1 literal call, got %d", len(result.LiteralCalls))
	}
	if result.LiteralCalls[0].ValidTest {
		t.Error("expected ValidTest=false for helper")
	}
}

func TestScanTestFileAST_DuplicateCalls(t *testing.T) {
	src := `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func TestDup(t *testing.T) {
	longtest.RequireLongTest(t, "LT-ID-001")
	longtest.RequireLongTest(t, "LT-ID-001")
}
`
	result, err := scanTestFileASTFromSrc(t, "testpkg", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.LiteralCalls) != 2 {
		t.Errorf("expected 2 literal calls, got %d", len(result.LiteralCalls))
	}
}

func TestScanTestFileAST_AliasedImport(t *testing.T) {
	src := `package testpkg

import (
	"testing"
	lt "github.com/s1onique/leamas/internal/factory/longtest"
)

func TestAliased(t *testing.T) {
	lt.RequireLongTest(t, "LT-ID-001")
}
`
	result, err := scanTestFileASTFromSrc(t, "testpkg", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.LiteralCalls) != 1 {
		t.Errorf("expected 1 literal call, got %d", len(result.LiteralCalls))
	}
	if result.LiteralCalls[0].ID != "LT-ID-001" {
		t.Errorf("expected ID LT-ID-001, got %s", result.LiteralCalls[0].ID)
	}
}

func TestScanTestFileAST_LowercaseTestName(t *testing.T) {
	src := `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func Testlowercase(t *testing.T) {
	longtest.RequireLongTest(t, "LT-ID-001")
}
`
	result, err := scanTestFileASTFromSrc(t, "testpkg", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.LiteralCalls) != 1 {
		t.Errorf("expected 1 literal call, got %d", len(result.LiteralCalls))
	}
	if result.LiteralCalls[0].ValidTest {
		t.Error("expected ValidTest=false for lowercase test name")
	}
}

func TestScanTestFileAST_GroupedParameters(t *testing.T) {
	// Regression test: func TestBad(a, b *testing.T) has one Field with two Names.
	// This should NOT be a valid test - it must have exactly one logical parameter.
	src := `package testpkg

import (
	"testing"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

func TestBad(a, b *testing.T) {
	longtest.RequireLongTest(t, "LT-ID-001")
}
`
	result, err := scanTestFileASTFromSrc(t, "testpkg", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.LiteralCalls) != 1 {
		t.Errorf("expected 1 literal call, got %d", len(result.LiteralCalls))
	}
	if result.LiteralCalls[0].ValidTest {
		t.Error("expected ValidTest=false for grouped parameters (a, b *testing.T)")
	}
}

// scanTestFileASTFromSrc is a test helper that creates a temp file and scans it.
func scanTestFileASTFromSrc(t *testing.T, pkgPath, src string) (*ScanResult, error) {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/test_" + pkgPath + ".go"
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	return scanTestFileAST(tmpFile, pkgPath)
}
