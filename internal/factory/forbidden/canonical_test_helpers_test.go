// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

type canonicalFixture struct {
	t      *testing.T
	root   string
	module string
}

func newCanonicalFixture(t *testing.T) *canonicalFixture {
	t.Helper()
	fixture := &canonicalFixture{
		t:      t,
		root:   t.TempDir(),
		module: "example.test/policy",
	}
	fixture.write("go.mod", "module "+fixture.module+"\n\ngo 1.25\n")
	if err := os.MkdirAll(filepath.Join(fixture.root, ".git"), 0o755); err != nil {
		t.Fatalf("create .git: %v", err)
	}
	return fixture
}

func (f *canonicalFixture) write(rel, content string) {
	f.t.Helper()
	path := filepath.Join(f.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		f.t.Fatalf("write %s: %v", rel, err)
	}
}

func (f *canonicalFixture) packagePath(rel string) string {
	if rel == "" {
		return f.module
	}
	return f.module + "/" + strings.Trim(rel, "/")
}

func (f *canonicalFixture) run(symbols []ProtectedSymbol, approvals []ApprovedCaller) canonicalResult {
	f.t.Helper()
	return runCanonicalAnalysis(f.root, f.module, canonicalConfig{
		protected: symbols,
		approvals: approvals,
	})
}

func fixtureSymbol(layer AuthorityLayer, pkg, name string, kind ProtectedSymbolKind, receiver string) ProtectedSymbol {
	return ProtectedSymbol{
		Layer:       layer,
		PackagePath: pkg,
		Name:        name,
		Kind:        kind,
		Receiver:    receiver,
	}
}

func fixtureApproval(callerPkg, function, receiver, kind string, callee ProtectedSymbol, class ReferenceClass) ApprovedCaller {
	return ApprovedCaller{
		PackagePath:    callerPkg,
		Function:       function,
		Receiver:       receiver,
		CallerKind:     kind,
		Callee:         callee,
		ReferenceClass: class,
		Cardinality:    1,
	}
}

func findingKinds(findings []checks.Finding) []string {
	kinds := make([]string, len(findings))
	for i := range findings {
		kinds[i] = findings[i].Kind
	}
	sort.Strings(kinds)
	return kinds
}

func requireFindingKind(t *testing.T, findings []checks.Finding, kind string) checks.Finding {
	t.Helper()
	for _, finding := range findings {
		if finding.Kind == kind {
			return finding
		}
	}
	t.Fatalf("missing finding kind %q; got kinds %v", kind, findingKinds(findings))
	return checks.Finding{}
}

func rejectFindingKind(t *testing.T, findings []checks.Finding, kind string) {
	t.Helper()
	for _, finding := range findings {
		if finding.Kind == kind {
			t.Fatalf("unexpected finding %s: %#v", kind, finding)
		}
	}
}

func edgeClasses(edges []ObservedEdge) []ReferenceClass {
	classes := make([]ReferenceClass, len(edges))
	for i := range edges {
		classes[i] = edges[i].ReferenceClass
	}
	sort.Slice(classes, func(i, j int) bool { return classes[i] < classes[j] })
	return classes
}
