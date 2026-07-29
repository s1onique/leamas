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
//
// analyzeThroughAdapter is intentionally NOT listed — it is an internal
// package helper that the constructor NewAnalyzerFromAdapter returns. Its
// capture as a function value within NewAnalyzerFromAdapter is allowed
// because the only caller (NewAnalyzerFromAdapter) is the protected
// constructor itself.
var AdapterProtectedSymbols = []ProtectedSymbol{
	// Constructor
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "NewDupcodeRunner", Kind: ProtectedPackageFunction},
	// Adapter-package analyzer wrapper (raw analyzer bound at construction).
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "NewAnalyzerFromAdapter", Kind: ProtectedPackageFunction},
	// Methods on *DupcodeRunner
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "RunCheckRepo", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "RunCheckReport", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "LoadBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "VerifyBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "WriteBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "CompareToBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner"},
	// Protected metadata reader (does not invoke raw LoadBaseline; reads JSON directly).
	// Declared as ProtectedPackageFunction so any external caller is flagged.
	{Layer: AuthorityLayerAdapter, PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Name: "ReadBaselineThresholds", Kind: ProtectedPackageFunction},
}

// ApprovedCallers defines exact approved caller-to-callee edges for the raw
// layer. Each *DupcodeRunner method is the sole allowed caller of the matching
// raw op.
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
	// Layer 2: runSharedDupcodeBaseline (named package function) may
	// invoke raw dupcode validation/drift helpers for the shared-context
	// dupcode-baseline verifier. These calls are NOT adapter methods —
	// they are direct dupcode-package calls from a named adapter-package
	// function, approved individually.
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "runSharedDupcodeBaseline",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "ValidateBaselineArtifact", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "runSharedDupcodeBaseline",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "CheckBaselineDriftFromReport", Kind: ProtectedPackageFunction,
		},
	},
}

// dupcodeInternalApprovedCallers defines approved callers inside the dupcode
// package itself for its own intra-package helpers. These are legitimate
// internal implementation edges. The dupcode package is fully protected, so
// ANY caller — internal or external — is matched against this list.
var dupcodeInternalApprovedCallers = []ApprovedCaller{
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
		Function:    "CheckReport",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerRaw,
			PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name:        "CheckRepo",
			Kind:        ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
		Function:    "CheckBaselineDrift",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerRaw,
			PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name:        "CheckReport",
			Kind:        ProtectedPackageFunction,
		},
	},
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
	if callee.Layer == AuthorityLayerRaw && caller.PackagePath == callee.PackagePath {
		// Internal dupcode-package calls require explicit intra-package approval.
		approved = append(approved, dupcodeInternalApprovedCallers...)
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
