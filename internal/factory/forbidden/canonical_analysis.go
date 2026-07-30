// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
	"golang.org/x/tools/go/packages"
)

type packageLoader func(*packages.Config, ...string) ([]*packages.Package, error)

type canonicalConfig struct {
	protected    []ProtectedSymbol
	approvals    []ApprovedCaller
	layerDomains authorityLayerDomains
	loader       packageLoader
}

type canonicalStats struct {
	ConfiguredProtectedSymbols int
	ResolvedProtectedObjects   int
	ConfiguredApprovals        int
	ObservedEdges              int
	ValidatedApprovals         int
}

type canonicalResult struct {
	Findings              []checks.Finding
	ObservedEdges         []ObservedEdge
	Stats                 canonicalStats
	NormalizedFindingHash string
}

// ObservedEdge is one globally typed protected source reference.
type ObservedEdge struct {
	Caller         CallerIdentity
	Callee         ProtectedSymbol
	ReferenceClass ReferenceClass
	Path           string
	Position       token.Position

	callerObject types.Object
	calleeObject types.Object
}

type sourceRange struct {
	start token.Pos
	end   token.Pos
}

type canonicalFinding struct {
	finding  checks.Finding
	position token.Position
	caller   CallerIdentity
	callee   ProtectedSymbol
	class    ReferenceClass
}

type canonicalAnalysis struct {
	policy       *DupcodeBypassPolicy
	config       canonicalConfig
	packages     []*packages.Package
	layerDomains authorityLayerDomains

	protectedByObject map[types.Object]ProtectedSymbol
	objectByProtected map[ProtectedSymbol]types.Object
	callersByIdentity map[CallerIdentity]types.Object
	callerCandidates  map[CallerIdentity][]types.Object

	directCalleeRanges map[*token.FileSet][]sourceRange
	observedEdges      []ObservedEdge
	findings           []canonicalFinding
	approvalStates     []resolvedApproval

	typedFiles   map[string]bool
	ignoredFiles map[string]bool
	invalid      bool
}

func productionCanonicalConfig() canonicalConfig {
	return canonicalConfig{
		protected:    append([]ProtectedSymbol(nil), allProtectedSymbols()...),
		approvals:    append([]ApprovedCaller(nil), allApprovedCallers()...),
		layerDomains: productionAuthorityLayerDomains(),
	}
}

func runCanonicalAnalysis(root, modulePath string, config canonicalConfig) canonicalResult {
	policy, err := NewDupcodeBypassPolicy(root, modulePath)
	if err != nil {
		finding := checks.Finding{Path: root, Kind: "canonical_root_error", Message: err.Error(), Severity: checks.SeverityError}
		return canonicalResult{Findings: []checks.Finding{finding}, NormalizedFindingHash: normalizedFindingHash([]checks.Finding{finding})}
	}
	analysis := newCanonicalAnalysis(policy, config)
	discovered, err := policy.DiscoverProductionFilesRepoWide()
	if err != nil {
		analysis.addFinding(policy.repoRoot, "dupcode_discovery_error", err.Error(), token.Position{}, CallerIdentity{}, ProtectedSymbol{}, "")
		return analysis.result()
	}
	analysis.loadAndValidatePackages()
	if !analysis.invalid {
		analysis.resolveProtectedDeclarations()
		analysis.resolveCallerDeclarations()
		analysis.resolveConfiguredApprovals()
		analysis.scanProtectedReferences()
		analysis.validateObservedEdges()
		analysis.validateConfiguredApprovals()
	}
	analysis.reconcileCoverage(discovered)
	return analysis.result()
}

func newCanonicalAnalysis(policy *DupcodeBypassPolicy, config canonicalConfig) *canonicalAnalysis {
	config.protected = append([]ProtectedSymbol(nil), config.protected...)
	config.approvals = append([]ApprovedCaller(nil), config.approvals...)
	if config.loader == nil {
		config.loader = packages.Load
	}
	return &canonicalAnalysis{
		policy:             policy,
		config:             config,
		layerDomains:       config.layerDomains,
		protectedByObject:  make(map[types.Object]ProtectedSymbol),
		objectByProtected:  make(map[ProtectedSymbol]types.Object),
		callersByIdentity:  make(map[CallerIdentity]types.Object),
		callerCandidates:   make(map[CallerIdentity][]types.Object),
		directCalleeRanges: make(map[*token.FileSet][]sourceRange),
		typedFiles:         make(map[string]bool),
		ignoredFiles:       make(map[string]bool),
	}
}

func (a *canonicalAnalysis) loadAndValidatePackages() {
	cfg := &packages.Config{
		Dir: a.policy.repoRoot,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedImports | packages.NeedDeps | packages.NeedModule,
		Tests: false,
		Fset:  token.NewFileSet(),
	}
	roots, err := a.config.loader(cfg, "./...")
	if err != nil {
		a.addFinding(a.policy.repoRoot, "dupcode_package_load_error", err.Error(), token.Position{}, CallerIdentity{}, ProtectedSymbol{}, "")
		a.invalid = true
		return
	}
	a.packages = repositoryPackages(roots, a.policy.modulePath)
	for _, pkg := range a.packages {
		a.validatePackage(pkg)
	}
}

func repositoryPackages(roots []*packages.Package, modulePath string) []*packages.Package {
	seen := make(map[string]*packages.Package)
	var visit func(*packages.Package)
	visit = func(pkg *packages.Package) {
		if pkg == nil {
			return
		}
		if _, ok := seen[pkg.ID]; ok {
			return
		}
		seen[pkg.ID] = pkg
		keys := make([]string, 0, len(pkg.Imports))
		for key := range pkg.Imports {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			visit(pkg.Imports[key])
		}
	}
	for _, root := range roots {
		visit(root)
	}
	out := make([]*packages.Package, 0, len(seen))
	for _, pkg := range seen {
		if pkg.PkgPath == modulePath || strings.HasPrefix(pkg.PkgPath, modulePath+"/") {
			out = append(out, pkg)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PkgPath != out[j].PkgPath {
			return out[i].PkgPath < out[j].PkgPath
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (a *canonicalAnalysis) validatePackage(pkg *packages.Package) {
	for _, ignored := range pkg.IgnoredFiles {
		a.ignoredFiles[a.relativePath(ignored)] = true
	}
	for _, packageErr := range pkg.Errors {
		a.addFinding(a.policy.repoRoot, "dupcode_package_metadata_error", fmt.Sprintf("%s: %v", pkg.PkgPath, packageErr), token.Position{}, CallerIdentity{}, ProtectedSymbol{}, "")
		a.invalid = true
	}
	for _, typeErr := range pkg.TypeErrors {
		a.addFinding(a.policy.repoRoot, "dupcode_type_error", fmt.Sprintf("%s: %v", pkg.PkgPath, typeErr), token.Position{}, CallerIdentity{}, ProtectedSymbol{}, "")
		a.invalid = true
	}
	if pkg.IllTyped {
		a.addFinding(a.policy.repoRoot, "dupcode_type_error", fmt.Sprintf("%s: ill-typed package", pkg.PkgPath), token.Position{}, CallerIdentity{}, ProtectedSymbol{}, "")
		a.invalid = true
	}
	if pkg.Types == nil || pkg.TypesInfo == nil || pkg.Fset == nil {
		a.addFinding(a.policy.repoRoot, "dupcode_type_info_error", fmt.Sprintf("%s: missing Types, TypesInfo, or Fset", pkg.PkgPath), token.Position{}, CallerIdentity{}, ProtectedSymbol{}, "")
		a.invalid = true
		return
	}
	if len(pkg.Syntax) != len(pkg.CompiledGoFiles) {
		message := fmt.Sprintf(
			"%s: syntax/file coverage mismatch: syntax=%d compiled=%d",
			pkg.PkgPath, len(pkg.Syntax), len(pkg.CompiledGoFiles),
		)
		a.addFinding(a.policy.repoRoot, "dupcode_analysis_coverage_error", message, token.Position{}, CallerIdentity{}, ProtectedSymbol{}, "")
		a.invalid = true
	}
	for index, file := range pkg.Syntax {
		position := pkg.Fset.PositionFor(file.Pos(), true)
		if position.Filename == "" {
			a.addFinding(a.policy.repoRoot, "dupcode_analysis_coverage_error", fmt.Sprintf("%s: syntax tree has no filename", pkg.PkgPath), token.Position{}, CallerIdentity{}, ProtectedSymbol{}, "")
			a.invalid = true
			continue
		}
		a.typedFiles[a.relativePath(position.Filename)] = true
		if index < len(pkg.CompiledGoFiles) && !sameFile(position.Filename, pkg.CompiledGoFiles[index]) {
			path := a.relativePath(pkg.CompiledGoFiles[index])
			a.addFinding(path, "dupcode_analysis_coverage_error", "compiled file does not match syntax tree", position, CallerIdentity{}, ProtectedSymbol{}, "")
			a.invalid = true
		}
	}
}

func sameFile(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func (a *canonicalAnalysis) relativePath(path string) string {
	rel, err := filepath.Rel(a.policy.repoRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func (a *canonicalAnalysis) reconcileCoverage(discovered []string) {
	for _, path := range discovered {
		if a.typedFiles[path] {
			continue
		}
		reason := "missing_from_packages"
		if a.ignoredFiles[path] {
			reason = "build_ignored"
		}
		a.addFinding(path, "dupcode_analysis_coverage_error", fmt.Sprintf("eligible file not analyzed (reason=%s): %s", reason, path), token.Position{}, CallerIdentity{}, ProtectedSymbol{}, "")
	}
}

func (a *canonicalAnalysis) syntaxFiles(pkg *packages.Package) []struct {
	filename string
	file     *ast.File
} {
	files := make([]struct {
		filename string
		file     *ast.File
	}, 0, len(pkg.Syntax))
	for _, file := range pkg.Syntax {
		filename := pkg.Fset.PositionFor(file.Pos(), true).Filename
		if filename != "" {
			files = append(files, struct {
				filename string
				file     *ast.File
			}{filename: filename, file: file})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].filename < files[j].filename })
	return files
}
