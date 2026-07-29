// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/forbidden/reporoot"
	"golang.org/x/tools/go/packages"
)

// DupcodeBypassPolicy is the canonical dupcode bypass policy.
type DupcodeBypassPolicy struct {
	repoRoot   string
	modulePath string
	resolver   *reporoot.RootResolver
}

// NewDupcodeBypassPolicy creates a new policy.
func NewDupcodeBypassPolicy(repoRoot, modulePath string) (*DupcodeBypassPolicy, error) {
	resolver := reporoot.New()
	canonicalRoot, err := resolver.Resolve(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve canonical root: %w", err)
	}
	return &DupcodeBypassPolicy{
		repoRoot:   canonicalRoot,
		modulePath: modulePath,
		resolver:   resolver,
	}, nil
}

// CanonicalCheckDupcodeBypass is the single exported entry point.
func CanonicalCheckDupcodeBypass(root string, modulePath string) []checks.Finding {
	policy, err := NewDupcodeBypassPolicy(root, modulePath)
	if err != nil {
		return []checks.Finding{
			{Path: root, Kind: "canonical_root_error", Message: err.Error(), Severity: checks.SeverityError},
		}
	}

	discovered, walkErr := policy.DiscoverProductionFilesRepoWide()
	if walkErr != nil {
		return []checks.Finding{
			{Path: policy.repoRoot, Kind: "dupcode_discovery_error", Message: walkErr.Error(), Severity: checks.SeverityError},
		}
	}

	cfg := &packages.Config{
		Dir:   policy.repoRoot,
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedModule,
		Tests: false,
		Fset:  token.NewFileSet(),
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return []checks.Finding{
			{Path: policy.repoRoot, Kind: "dupcode_package_load_error", Message: err.Error(), Severity: checks.SeverityError},
		}
	}

	var findings []checks.Finding
	typedFiles := make(map[string]bool)
	ignoredFiles := make(map[string]bool)

	for _, pkg := range pkgs {
		if pkg.PkgPath == "" || strings.HasSuffix(pkg.PkgPath, "_test") {
			continue
		}

		for _, ignored := range pkg.IgnoredFiles {
			rel, err := filepath.Rel(policy.repoRoot, ignored)
			if err != nil {
				continue
			}
			ignoredFiles[filepath.ToSlash(rel)] = true
		}

		packageInvalid := false
		if len(pkg.Errors) > 0 {
			for _, e := range pkg.Errors {
				findings = append(findings, checks.Finding{
					Path: policy.repoRoot, Kind: "dupcode_package_metadata_error",
					Message:  fmt.Sprintf("%s: %v", pkg.PkgPath, e),
					Severity: checks.SeverityError,
				})
			}
			packageInvalid = true
		}
		if len(pkg.TypeErrors) > 0 {
			for _, e := range pkg.TypeErrors {
				findings = append(findings, checks.Finding{
					Path: policy.repoRoot, Kind: "dupcode_type_error",
					Message:  fmt.Sprintf("%s: %v", pkg.PkgPath, e),
					Severity: checks.SeverityError,
				})
			}
			packageInvalid = true
		}
		if pkg.IllTyped {
			findings = append(findings, checks.Finding{
				Path: policy.repoRoot, Kind: "dupcode_type_error",
				Message:  fmt.Sprintf("%s: ill-typed package", pkg.PkgPath),
				Severity: checks.SeverityError,
			})
			packageInvalid = true
		}
		if pkg.Types == nil || pkg.TypesInfo == nil || pkg.Fset == nil {
			findings = append(findings, checks.Finding{
				Path: policy.repoRoot, Kind: "dupcode_type_info_error",
				Message:  fmt.Sprintf("%s: missing type info or file set", pkg.PkgPath),
				Severity: checks.SeverityError,
			})
			packageInvalid = true
		}

		if packageInvalid {
			continue
		}

		// Map syntax trees to filenames via Fset
		syntaxFiles := make(map[string]*ast.File)
		for _, file := range pkg.Syntax {
			pos := pkg.Fset.PositionFor(file.Pos(), true)
			if pos.Filename == "" {
				continue
			}
			syntaxFiles[filepath.ToSlash(pos.Filename)] = file
		}

		// Verify each CompiledGoFile has a syntax tree
		for _, compiled := range pkg.CompiledGoFiles {
			rel, err := filepath.Rel(policy.repoRoot, compiled)
			if err != nil {
				continue
			}
			relSlash := filepath.ToSlash(rel)
			if _, ok := syntaxFiles[filepath.ToSlash(compiled)]; !ok {
				if _, ok := syntaxFiles[relSlash]; !ok {
					findings = append(findings, checks.Finding{
						Path: relSlash, Kind: "dupcode_analysis_coverage_error",
						Message:  fmt.Sprintf("compiled file missing syntax: %s", relSlash),
						Severity: checks.SeverityError,
					})
				}
			}
		}

		for filename, file := range syntaxFiles {
			rel, err := filepath.Rel(policy.repoRoot, filename)
			if err != nil {
				continue
			}
			relSlash := filepath.ToSlash(rel)
			typedFiles[relSlash] = true

			fileFindings := policy.analyzeFile(pkg, filename, file)
			findings = append(findings, fileFindings...)
		}
	}

	// Reconcile discovery with typed files
	for _, d := range discovered {
		if !typedFiles[d] {
			reason := "missing_from_packages"
			if ignoredFiles[d] {
				reason = "build_ignored"
			}
			findings = append(findings, checks.Finding{
				Path: d, Kind: "dupcode_analysis_coverage_error",
				Message:  fmt.Sprintf("eligible file not analyzed (reason=%s): %s", reason, d),
				Severity: checks.SeverityError,
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		return findings[i].Message < findings[j].Message
	})

	return findings
}
