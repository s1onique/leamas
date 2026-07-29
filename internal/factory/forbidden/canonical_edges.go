// SPDX-License-Identifier: Apache-2.0

package forbidden

// AdapterProtectedSymbols defines the protectedverifier adapter-layer capabilities.
// These are the only exported methods/constructors through which raw dupcode
// operations may be invoked.
//
// Authority chain:
//
//	raw dupcode operations (AuthorityLayerRaw)
//	  → exact adapter implementation (AuthorityLayerAdapter, here)
//	  → exact dispatcher-owned authority path
//
// Anything else that touches raw dupcode or constructs an adapter is a bypass.
var AdapterProtectedSymbols = []ProtectedSymbol{
	// Constructor
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "NewDupcodeRunner", Kind: ProtectedPackageFunction},
	// Methods on *DupcodeRunner
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "RunCheckRepo", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "RunCheckReport", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "LoadBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "VerifyBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "WriteBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "CompareToBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	// Global analyzer escape - explicitly protected (not approved in production).
	// Analyzer() and SetAnalyzer() are package-level functions.
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "Analyzer", Kind: ProtectedPackageFunction},
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "SetAnalyzer", Kind: ProtectedPackageFunction},
	// DefaultAnalyzer is a package-level variable holding a function value.
	// Resolved as *types.Var (not *types.Func).
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "DefaultAnalyzer", Kind: ProtectedPackageVariable},
}

// ApprovedCallers defines exact approved caller-to-callee edges for BOTH layers.
//
// Raw edges: adapter method on *DupcodeRunner → raw dupcode function.
// Adapter edges: dispatcher owner → protectedverifier adapter symbol.
//
// No wildcards. Function/Receiver must match exactly. Anonymous literals
// (func@line:col) are NOT allowed as approvals.
var ApprovedCallers = []ApprovedCaller{
	// Layer 1: raw dupcode operations allowed only inside exact adapter methods.
	// Each *DupcodeRunner method is the sole allowed caller of the matching raw op.
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "RunCheckRepo",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "CheckRepo", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "RunCheckReport",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "CheckReport", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "LoadBaseline",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "LoadBaseline", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "VerifyBaseline",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "VerifyBaseline", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "WriteBaseline",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "WriteBaseline", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "CompareToBaseline",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "CompareToBaseline", Kind: ProtectedPackageFunction,
		},
	},
}

// AdapterApprovedCallers defines approved dispatcher-owner edges for adapter-layer symbols.
// These are the ONLY legitimate callers of protectedverifier adapter methods.
// Package-level wildcards are forbidden.
var AdapterApprovedCallers = []ApprovedCaller{
	// Dispatcher.Dispatch is the canonical authority owner.
	// Concrete function/edge must be set after dispatcher source audit.
}

// IsApprovedCaller checks if a caller-callee edge is approved for EITHER layer.
// Function and Receiver must match exactly. Empty Function is invalid.
//
// No wildcards. Every approved edge must correspond to an exact named caller
// declaration in ApprovedCallers or AdapterApprovedCallers. Internal
// implementation-package edges (e.g., helper functions inside dupcode) must
// be listed explicitly when needed.
func IsApprovedCaller(caller CallerIdentity, callee ProtectedSymbol) bool {
	approved := ApprovedCallers
	if callee.Layer == AuthorityLayerAdapter {
		approved = AdapterApprovedCallers
	}
	for _, ac := range approved {
		if ac.PackagePath != caller.PackagePath {
			continue
		}
		if ac.Callee.PackagePath != callee.PackagePath ||
			ac.Callee.Name != callee.Name ||
			ac.Callee.Kind != callee.Kind ||
			ac.Callee.Receiver != callee.Receiver ||
			ac.Callee.Layer != callee.Layer {
			continue
		}
		if ac.Function == "" || ac.Function != caller.Function {
			continue
		}
		if ac.Receiver != caller.Receiver {
			continue
		}
		return true
	}
	return false
}
