// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"

	"golang.org/x/tools/go/packages"
)

func deterministicFixture(t *testing.T) (*canonicalFixture, canonicalConfig) {
	t.Helper()
	fixture, protected, _ := approvalFixture(t, `package caller
import p "example.test/policy/protected"
func Unapproved() { p.Cap() }
`)
	return fixture, canonicalConfig{protected: []ProtectedSymbol{protected}}
}

func TestCanonicalDeterministicNormalizedFindings(t *testing.T) {
	fixture, config := deterministicFixture(t)
	var baseline canonicalResult
	for run := 0; run < 3; run++ {
		result := runCanonicalAnalysis(fixture.root, fixture.module, config)
		if run == 0 {
			baseline = result
			continue
		}
		if !reflect.DeepEqual(result.Findings, baseline.Findings) {
			t.Fatalf("run %d findings differ:\nfirst=%#v\nnext=%#v", run, baseline.Findings, result.Findings)
		}
		if result.NormalizedFindingHash != baseline.NormalizedFindingHash {
			t.Fatalf("run %d hash = %s, want %s", run, result.NormalizedFindingHash, baseline.NormalizedFindingHash)
		}
	}
	if baseline.NormalizedFindingHash == "" {
		t.Fatal("normalized finding hash is empty")
	}
}

func TestCanonicalConcurrentAnalysesRemainIsolated(t *testing.T) {
	fixture, config := deterministicFixture(t)
	var loads atomic.Int64
	config.loader = func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
		loads.Add(1)
		return packages.Load(cfg, patterns...)
	}

	const analyses = 8
	results := make([]canonicalResult, analyses)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for i := range results {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			results[index] = runCanonicalAnalysis(fixture.root, fixture.module, config)
		}(i)
	}
	close(start)
	wait.Wait()

	if got := loads.Load(); got != analyses {
		t.Fatalf("packages.Load calls = %d, want one per analysis (%d)", got, analyses)
	}
	for i := 1; i < analyses; i++ {
		if !reflect.DeepEqual(results[i].Findings, results[0].Findings) ||
			results[i].NormalizedFindingHash != results[0].NormalizedFindingHash {
			t.Fatalf("concurrent result %d differs from result 0", i)
		}
		if len(results[i].ObservedEdges) != 1 {
			t.Fatalf("concurrent result %d edges = %d, want 1", i, len(results[i].ObservedEdges))
		}
	}
}

func TestCanonicalPackageIterationOrderDoesNotAffectOutput(t *testing.T) {
	fixture, config := deterministicFixture(t)
	forward := runCanonicalAnalysis(fixture.root, fixture.module, config)
	config.loader = func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
		loaded, err := packages.Load(cfg, patterns...)
		sort.Slice(loaded, func(i, j int) bool { return loaded[i].ID > loaded[j].ID })
		return loaded, err
	}
	reversed := runCanonicalAnalysis(fixture.root, fixture.module, config)
	if !reflect.DeepEqual(forward.Findings, reversed.Findings) {
		t.Fatalf("package-order findings differ:\nforward=%#v\nreversed=%#v", forward.Findings, reversed.Findings)
	}
}

func TestCanonicalHasNoProcessGlobalClassifierState(t *testing.T) {
	files := []string{
		"canonical_policy_v2.go",
		"canonical_declarations.go",
		"canonical_resolve.go",
		"canonical_traversal.go",
		"canonical_analysis.go",
		"canonical_callers.go",
		"canonical_approvals.go",
	}
	for _, path := range files {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.VAR {
				continue
			}
			for _, spec := range general.Specs {
				value := spec.(*ast.ValueSpec)
				for _, name := range value.Names {
					if name.Name == "packageDirectCalleeRanges" {
						t.Errorf("%s retains process-global classifier state %s", path, name.Name)
					}
				}
			}
		}
	}
}
