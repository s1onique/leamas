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

	// ────────────────────────────────────────────────────────────────────
	// gate.FactorizeVerifiersWithDupcodeContext wires the production dupcode
	// analyzer into the shared analysis context. The analyzer reference
	// (dupcode.CheckRepo) is captured here and passed through to the
	// binder-local DupcodeAnalysisProvider. The actual scan happens
	// later through the named post-authority binder.
	// ────────────────────────────────────────────────────────────────────
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "FactorizeVerifiersWithDupcodeContext",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "CheckRepo", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "factorizeVerifiersWithDupcodeAnalyzer",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "LoadBaseline", Kind: ProtectedPackageFunction,
		},
	},
}

// AdapterApprovedCallers defines the exact post-authority caller edges for
// adapter-layer symbols. Each edge corresponds to a real named caller
// declaration in internal/factory/gate that constructs the protected adapter
// or invokes an exact adapter operation.
//
// No wildcards. Function/Receiver must match exactly. Package-level wildcards
// (all functions in gate) and anonymous literals (func@line:col) are
// forbidden. The list reflects the production topology after the
// CORRECTION02G refactor:
//
//	CLI data → DispatchDupcodeVerifyTyped
//	CLI data → DispatchDupcodeBaselineVerifyTyped
//	CLI data → DispatchDupcodeUpdateBaselineTyped
//	  → named post-authority binder (DupcodeVerifyBinder / DupcodeBaselineBinder /
//	    DupcodeUpdateBaselineBinder) or named bound runner
//	    (dupCodeVerifier / dupcodeBaselineVerifier)
//	  → protectedverifier.NewDupcodeRunner
//	  → exact adapter method (LoadBaseline / RunCheckReport / VerifyBaseline /
//	    WriteBaseline / CompareToBaseline)
var AdapterApprovedCallers = []ApprovedCaller{
	// ────────────────────────────────────────────────────────────────────
	// DupcodeVerifyBinder.run (named post-authority binder, verify lane)
	// ────────────────────────────────────────────────────────────────────
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "run",
		Receiver:    "DupcodeVerifyBinder",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "NewDupcodeRunner", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "run",
		Receiver:    "DupcodeVerifyBinder",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "LoadBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "run",
		Receiver:    "DupcodeVerifyBinder",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "RunCheckReport", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "run",
		Receiver:    "DupcodeVerifyBinder",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "CompareToBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},

	// ────────────────────────────────────────────────────────────────────
	// DupcodeBaselineBinder.run (named post-authority binder, baseline lane)
	// ────────────────────────────────────────────────────────────────────
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "run",
		Receiver:    "DupcodeBaselineBinder",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "NewDupcodeRunner", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "run",
		Receiver:    "DupcodeBaselineBinder",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "VerifyBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},

	// ────────────────────────────────────────────────────────────────────
	// DupcodeUpdateBaselineBinder.run (named post-authority binder, update lane)
	// ────────────────────────────────────────────────────────────────────
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "run",
		Receiver:    "DupcodeUpdateBaselineBinder",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "NewDupcodeRunner", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "run",
		Receiver:    "DupcodeUpdateBaselineBinder",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "RunCheckReport", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "run",
		Receiver:    "DupcodeUpdateBaselineBinder",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "WriteBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},

	// ────────────────────────────────────────────────────────────────────
	// dupCodeVerifier (named bound runner, registry verify lane)
	// ────────────────────────────────────────────────────────────────────
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "dupCodeVerifier",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "NewDupcodeRunner", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "dupCodeVerifier",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "LoadBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "dupCodeVerifier",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "RunCheckReport", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "dupCodeVerifier",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "CompareToBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},

	// ────────────────────────────────────────────────────────────────────
	// dupcodeBaselineVerifier (named bound runner, registry baseline lane)
	// ────────────────────────────────────────────────────────────────────
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "dupcodeBaselineVerifier",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "NewDupcodeRunner", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "dupcodeBaselineVerifier",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "VerifyBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},

	// ────────────────────────────────────────────────────────────────────
	// SharedDupCodeVerifier closure (named post-authority bound runner inside
	// protectedverifier.DupcodeVerifierFactory). All raw dupcode operations
	// from this closure flow through the DupcodeRunner adapter.
	// ────────────────────────────────────────────────────────────────────
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "SharedDupCodeVerifier",
		Receiver:    "DupcodeVerifierFactory",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "NewDupcodeRunner", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "SharedDupCodeVerifier",
		Receiver:    "DupcodeVerifierFactory",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "LoadBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "SharedDupCodeVerifier",
		Receiver:    "DupcodeVerifierFactory",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "CompareToBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
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
