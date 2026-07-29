// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"go/types"
	"strings"
)

// AuthorityLayer distinguishes between raw protected operations and adapter capabilities.
type AuthorityLayer string

const (
	// AuthorityLayerRaw protects raw dupcode package-level operations.
	AuthorityLayerRaw AuthorityLayer = "raw_dupcode"
	// AuthorityLayerAdapter protects protectedverifier adapter capabilities.
	AuthorityLayerAdapter AuthorityLayer = "protected_adapter"
)

// ProtectedSymbolKind is the kind of a protected symbol.
type ProtectedSymbolKind string

const (
	ProtectedPackageFunction ProtectedSymbolKind = "package_function"
	ProtectedMethod          ProtectedSymbolKind = "method"
	// ProtectedPackageVariable is a package-level variable holding a function value.
	ProtectedPackageVariable ProtectedSymbolKind = "package_variable"
)

// ProtectedSymbol represents an exact protected declaration.
type ProtectedSymbol struct {
	Layer       AuthorityLayer
	PackagePath string
	Name        string
	Kind        ProtectedSymbolKind
	// Receiver is required for ProtectedMethod.
	Receiver string
}

// ApprovedCaller defines an exact approved caller edge.
type ApprovedCaller struct {
	PackagePath string
	Function    string
	Receiver    string
	Callee      ProtectedSymbol
}

// DupcodeProtectedPrefixes defines protected package prefixes.
var DupcodeProtectedPrefixes = []string{
	"github.com/s1onique/leamas/internal/factory/dupcode",
}

// ProtectedSymbols defines exact raw-layer protected symbols.
// Source of truth: internal/factory/dupcode
var ProtectedSymbols = []ProtectedSymbol{
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckRepo", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckReport", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "LoadBaseline", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "VerifyBaseline", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "WriteBaseline", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CompareToBaseline", Kind: ProtectedPackageFunction},
}

// CallerIdentity identifies an enclosing caller.
type CallerIdentity struct {
	PackagePath string
	Function    string
	Receiver    string
	Kind        string
}

// ProtectedSymbolsMap returns a map for fast lookup.
func ProtectedSymbolsMap() map[string]bool {
	result := make(map[string]bool)
	for _, sym := range ProtectedSymbols {
		key := sym.PackagePath + "." + sym.Name
		result[key] = true
	}
	return result
}

// FindProtectedSymbol finds a protected symbol by package path and name.
func FindProtectedSymbol(pkgPath, name string) *ProtectedSymbol {
	for _, sym := range ProtectedSymbols {
		if sym.PackagePath == pkgPath && sym.Name == name {
			return &sym
		}
	}
	return nil
}

func isProtectedPackage(path string) bool {
	for _, prefix := range DupcodeProtectedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func recvTypeNameFromSig(recv *types.Var) string {
	if recv == nil {
		return ""
	}
	if named, ok := recv.Type().(*types.Named); ok {
		return named.Obj().Name()
	}
	if ptr, ok := recv.Type().(*types.Pointer); ok {
		if named, ok := ptr.Elem().(*types.Named); ok {
			return named.Obj().Name()
		}
	}
	return ""
}
