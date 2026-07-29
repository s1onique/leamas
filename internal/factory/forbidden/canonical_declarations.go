// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/checks"
	"golang.org/x/tools/go/packages"
)

// resolveProtectedDeclarations validates every configured protected symbol
// declared in the given package. It returns a slice of policy findings for
// mismatched, missing, duplicated, or scope-incorrect declarations.
//
// Validation rules (fail-closed with explicit codes):
//
//	authority_policy_symbol_missing    declaration not found
//	authority_policy_kind_mismatch    declaration kind does not match configured
//	authority_policy_receiver_mismatch  method receiver does not match
//	authority_policy_scope_mismatch   variable not in package scope
//	authority_policy_duplicate_symbol same logical symbol declared twice
func (p *DupcodeBypassPolicy) resolveProtectedDeclarations(pkg *packages.Package, file *ast.File) []checks.Finding {
	if pkg.Types == nil || pkg.TypesInfo == nil {
		return nil
	}
	var findings []checks.Finding
	relPath, _ := filepath.Rel(p.repoRoot, pkg.Fset.Position(file.Pos()).Filename)
	if relPath == "" {
		relPath = file.Name.Name
	}

	seen := make(map[string]bool)

	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		obj := pkg.TypesInfo.Defs[fd.Name]
		if obj == nil {
			continue
		}
		fn, ok := obj.(*types.Func)
		if !ok {
			continue
		}
		fnPkg := fn.Pkg()
		if fnPkg == nil {
			continue
		}
		pkgPath := fnPkg.Path()
		name := fn.Name()

		// Only check symbols configured in the policy.
		var configured *ProtectedSymbol
		for i := range ProtectedSymbols {
			s := &ProtectedSymbols[i]
			if s.PackagePath == pkgPath && s.Name == name {
				configured = s
				break
			}
		}
		var configuredAdapter *ProtectedSymbol
		if configured == nil {
			for i := range AdapterProtectedSymbols {
				s := &AdapterProtectedSymbols[i]
				if s.PackagePath == pkgPath && s.Name == name {
					configuredAdapter = s
					break
				}
			}
		}
		if configured == nil && configuredAdapter == nil {
			continue
		}
		cfg := configured
		if cfg == nil {
			cfg = configuredAdapter
		}

		// Layer mismatch: a function declared in protectedverifier can only be
		// an adapter-layer symbol; one declared in dupcode can only be raw.
		expectedLayer := AuthorityLayerRaw
		if configuredAdapter != nil {
			expectedLayer = AuthorityLayerAdapter
		}
		if cfg.Layer != expectedLayer {
			findings = append(findings, checks.Finding{
				Path: relPath, Kind: "authority_policy_kind_mismatch",
				Message: fmt.Sprintf("line %d: configured layer %s does not match package %s",
					pkg.Fset.Position(fd.Pos()).Line, cfg.Layer, pkgPath),
				Severity: checks.SeverityError,
			})
			continue
		}

		// Kind / receiver validation against the actual declaration.
		sig, _ := fn.Type().(*types.Signature)
		hasRecv := sig != nil && sig.Recv() != nil

		if configuredAdapter != nil && configuredAdapter.Kind == ProtectedPackageFunction {
			if hasRecv {
				findings = append(findings, checks.Finding{
					Path: relPath, Kind: "authority_policy_kind_mismatch",
					Message: fmt.Sprintf("line %d: %s.%s declared as method but configured as package_function",
						pkg.Fset.Position(fd.Pos()).Line, pkgPath, name),
					Severity: checks.SeverityError,
				})
				continue
			}
		}
		if configuredAdapter != nil && configuredAdapter.Kind == ProtectedMethod {
			if !hasRecv {
				findings = append(findings, checks.Finding{
					Path: relPath, Kind: "authority_policy_kind_mismatch",
					Message: fmt.Sprintf("line %d: %s.%s declared as package function but configured as method",
						pkg.Fset.Position(fd.Pos()).Line, pkgPath, name),
					Severity: checks.SeverityError,
				})
				continue
			}
			actualRecv := recvTypeNameFromSig(sig.Recv())
			if actualRecv != configuredAdapter.Receiver {
				findings = append(findings, checks.Finding{
					Path: relPath, Kind: "authority_policy_receiver_mismatch",
					Message: fmt.Sprintf("line %d: %s.%s receiver %q does not match configured %q",
						pkg.Fset.Position(fd.Pos()).Line, pkgPath, name,
						actualRecv, configuredAdapter.Receiver),
					Severity: checks.SeverityError,
				})
				continue
			}
		}

		key := string(expectedLayer) + "|" + pkgPath + "|" + name
		if seen[key] {
			findings = append(findings, checks.Finding{
				Path: relPath, Kind: "authority_policy_duplicate_symbol",
				Message:  fmt.Sprintf("%s.%s declared more than once in package", pkgPath, name),
				Severity: checks.SeverityError,
			})
			continue
		}
		seen[key] = true
	}

	// Validate configured package-level variables (e.g. DefaultAnalyzer).
	for _, sym := range AdapterProtectedSymbols {
		if sym.Kind != ProtectedPackageVariable {
			continue
		}
		scope := pkg.Types.Scope()
		if scope == nil {
			continue
		}
		obj := scope.Lookup(sym.Name)
		if obj == nil {
			findings = append(findings, checks.Finding{
				Path: relPath, Kind: "authority_policy_symbol_missing",
				Message: fmt.Sprintf("configured protected variable %s.%s not found in package scope",
					sym.PackagePath, sym.Name),
				Severity: checks.SeverityError,
			})
			continue
		}
		v, ok := obj.(*types.Var)
		if !ok {
			findings = append(findings, checks.Finding{
				Path: relPath, Kind: "authority_policy_kind_mismatch",
				Message: fmt.Sprintf("%s.%s configured as package_variable but is not a *types.Var",
					sym.PackagePath, sym.Name),
				Severity: checks.SeverityError,
			})
			continue
		}
		if v.Parent() != scope {
			findings = append(findings, checks.Finding{
				Path: relPath, Kind: "authority_policy_scope_mismatch",
				Message: fmt.Sprintf("%s.%s configured as package_variable but lives in non-package scope",
					sym.PackagePath, sym.Name),
				Severity: checks.SeverityError,
			})
		}
	}

	return findings
}
