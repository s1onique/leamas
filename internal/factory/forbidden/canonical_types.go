// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"go/types"
	"strings"
)

// AuthorityLayer identifies one protected capability boundary.
type AuthorityLayer string

const (
	AuthorityLayerRaw     AuthorityLayer = "raw_dupcode"
	AuthorityLayerAdapter AuthorityLayer = "protected_adapter"
	AuthorityLayerGate    AuthorityLayer = "factorize_gate"
)

// ProtectedSymbolKind is the configured declaration class.
type ProtectedSymbolKind string

const (
	ProtectedPackageFunction ProtectedSymbolKind = "package_function"
	ProtectedMethod          ProtectedSymbolKind = "method"
	ProtectedPackageVariable ProtectedSymbolKind = "package_variable"
)

// ReferenceClass describes how source refers to a protected declaration.
type ReferenceClass string

type referenceClass = ReferenceClass

const (
	refDirectCall       ReferenceClass = "DIRECT_CALL"
	refFunctionValue    ReferenceClass = "FUNCTION_VALUE"
	refMethodValue      ReferenceClass = "METHOD_VALUE"
	refMethodExpression ReferenceClass = "METHOD_EXPRESSION"
	refPackageVariable  ReferenceClass = "PACKAGE_VARIABLE_REFERENCE"
	refDotImport        ReferenceClass = "DOT_IMPORT"
	refDeclaration      ReferenceClass = "DECLARATION"
)

const (
	CallerKindPackageFunction     = "package_function"
	CallerKindMethod              = "method"
	CallerKindVariableInitializer = "variable_initializer"
	CallerKindPackageInit         = "package_init"
	CallerKindFunctionLiteral     = "function_literal"
)

// ProtectedSymbol is the declarative identity of one protected object.
type ProtectedSymbol struct {
	Layer       AuthorityLayer
	PackagePath string
	Name        string
	Kind        ProtectedSymbolKind
	Receiver    string
}

// ApprovedCaller declares one exact caller/callee/reference edge.
type ApprovedCaller struct {
	PackagePath    string
	Function       string
	Receiver       string
	CallerKind     string
	Callee         ProtectedSymbol
	ReferenceClass ReferenceClass
	Cardinality    int
}

// CallerIdentity identifies a stable enclosing declaration.
type CallerIdentity struct {
	PackagePath string
	Function    string
	Receiver    string
	Kind        string
}

var DupcodeProtectedPrefixes = []string{
	"github.com/s1onique/leamas/internal/factory/dupcode",
}

// ProtectedSymbols is the raw-layer declaration inventory.
var ProtectedSymbols = []ProtectedSymbol{
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckRepo", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckReport", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "LoadBaseline", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "VerifyBaseline", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "WriteBaseline", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CompareToBaseline", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "ValidateBaselineArtifact", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckBaselineDriftFromReport", Kind: ProtectedPackageFunction},
}

var AdapterProtectedPrefixes = []string{
	"github.com/s1onique/leamas/internal/factory/protectedverifier",
}

var GateProtectedPrefixes = []string{
	"github.com/s1onique/leamas/internal/factory/gate",
}

func allProtectedSymbols() []ProtectedSymbol {
	out := make([]ProtectedSymbol, 0, len(ProtectedSymbols)+len(AdapterProtectedSymbols)+len(GateProtectedSymbols))
	out = append(out, ProtectedSymbols...)
	out = append(out, AdapterProtectedSymbols...)
	out = append(out, GateProtectedSymbols...)
	return out
}

// ProtectedSymbolsMap returns configured package/name identities.
func ProtectedSymbolsMap() map[string]bool {
	result := make(map[string]bool)
	for _, symbol := range allProtectedSymbols() {
		result[symbol.PackagePath+"."+symbol.Name] = true
	}
	return result
}

// FindProtectedSymbol finds an exact configured package/name identity.
func FindProtectedSymbol(pkgPath, name string) *ProtectedSymbol {
	for _, symbol := range allProtectedSymbols() {
		if symbol.PackagePath == pkgPath && symbol.Name == name {
			copy := symbol
			return &copy
		}
	}
	return nil
}

func isProtectedPackage(path string) bool {
	return pathMatchesPrefixes(path, DupcodeProtectedPrefixes)
}

func isAdapterProtectedPackage(path string) bool {
	return pathMatchesPrefixes(path, AdapterProtectedPrefixes)
}

func isGateProtectedPackage(path string) bool {
	return pathMatchesPrefixes(path, GateProtectedPrefixes)
}

func pathMatchesPrefixes(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
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
	typeOf := recv.Type()
	if pointer, ok := typeOf.(*types.Pointer); ok {
		typeOf = pointer.Elem()
	}
	if named, ok := typeOf.(*types.Named); ok {
		return named.Obj().Name()
	}
	return ""
}
