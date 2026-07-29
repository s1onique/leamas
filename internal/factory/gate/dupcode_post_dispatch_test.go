// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// protectedCallNames is the set of symbol names that must NOT appear in the
// post-Dispatch region of any typed entry-point function. They map to
// method calls or factory invocations that would re-execute protected work.
var protectedCallNames = []string{
	"binder.run",
	"BindRunner",
	"deps.NewRunner",
	"NewDupcodeRunner",
	"LoadBaseline",
	"RunCheckReport",
	"VerifyBaseline",
	"WriteBaseline",
	"CompareToBaseline",
}

// typedEntryPointNames lists the typed dispatch entry-point functions
// whose post-Dispatch body must not contain protected calls. The
// package-internal *With variants own the dispatcher.Dispatch call,
// while the production entry points just delegate to them.
var typedEntryPointNames = []string{
	"dispatchDupcodeVerifyTypedWith",
	"dispatchDupcodeBaselineVerifyTypedWith",
	"dispatchDupcodeUpdateBaselineTypedWith",
	"DispatchDupcodeVerifyTyped",
	"DispatchDupcodeBaselineVerifyTyped",
	"DispatchDupcodeUpdateBaselineTyped",
}

// TestDupcodeTypedEntryPointsNoPostDispatchProtectedCalls is a
// structural guard over the typed dispatch entry points. It uses simple
// source-text scanning so it does not depend on full AST inspection
// (which can pass nil children to visitor callbacks). The behavioral
// exactly-once test remains authoritative; this guard is a regression
// sentinel against re-introducing protected work after DispatcherForVerifier.
func TestDupcodeTypedEntryPointsNoPostDispatchProtectedCalls(t *testing.T) {
	root, err := findModuleRootForGateTest()
	if err != nil {
		t.Fatal(err)
	}
	srcDir := filepath.Join(root, "internal", "factory", "gate")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(srcDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		for _, want := range typedEntryPointNames {
			funcHeader := "func " + want + "("
			idx := strings.Index(src, funcHeader)
			if idx < 0 {
				continue
			}
			// Isolate the function body up to the next top-level "func " or end of file.
			rest := src[idx+len(funcHeader):]
			bodyStart := strings.Index(rest, "{")
			if bodyStart < 0 {
				continue
			}
			body := rest[bodyStart:]
			end := strings.Index(body, "\nfunc ")
			if end < 0 {
				end = len(body)
			}
			region := body[:end]
			// Find the LAST dispatcher.Dispatch(...) call inside this body.
			lastIdx := -1
			searchFrom := 0
			for searchFrom < len(region) {
				dIdx := strings.Index(region[searchFrom:], ".Dispatch(")
				if dIdx < 0 {
					break
				}
				lastIdx = searchFrom + dIdx
				searchFrom = lastIdx + 1
			}
			if lastIdx < 0 {
				continue
			}
			// Anything after the closing paren of the LAST .Dispatch( call
			// is post-Dispatch. We walk parens to find the matching close,
			// since .Dispatch's argument list may itself contain parens
			// (e.g. binder.BindRunner()).
			after := region[lastIdx+len(".Dispatch("):]
			depth := 1
			endIdx := -1
			for i, r := range after {
				switch r {
				case '(':
					depth++
				case ')':
					depth--
					if depth == 0 {
						endIdx = i
						break
					}
				}
			}
			if endIdx < 0 {
				continue
			}
			afterDispatch := after[endIdx+1:]
			for _, bad := range protectedCallNames {
				if strings.Contains(afterDispatch, bad+"(") {
					t.Errorf("%s in %s: post-Dispatch contains protected call %q", want, e.Name(), bad)
				}
			}
		}
	}
}

// writeFakeBaseline writes a minimal valid baseline file to dir/name and
// returns its path. It exists so admission tests can drive the verify
// lane past the missing_baseline early-return into LoadBaseline/Compare.

// writeFakeBaseline writes a minimal valid baseline file to dir/name and
// returns its path. It exists so admission tests can drive the verify
// lane past the missing_baseline early-return into LoadBaseline/Compare.
func writeFakeBaseline(t *testing.T, dir, name string, minLines, minTokens int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	content := []byte(fmt.Sprintf(`{"schema_version":1,"generated_at":"test","tool":"test","thresholds":{"min_lines":%d,"min_tokens":%d},"findings":[]}`, minLines, minTokens))
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	return path
}

func findModuleRootForGateTest() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

// reflect.DeepEqual is a small wrapper that lets tests compare
// struct values without importing the reflect package at the top of this
// file (avoiding pull-in across test files that might add shadowing).
func init() {
	// Compile-time check that the verifierdispatch package is reachable.
	_ = verifierdispatch.Result{}
}
