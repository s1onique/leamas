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
// (func@line:col) are NOT allowed as approvals. CallerKind, ReferenceClass,
// and Cardinality are now explicit per record (no implicit normalization
// at runtime). The exact effective values are frozen by the
// TestApprovalCensusNormalizationOracleHashStable oracle.
var ApprovedCallers = []ApprovedCaller{
	// Layer 1: raw dupcode operations allowed only inside exact adapter methods.
	// Each *DupcodeRunner method is the sole allowed caller of the matching raw op.
	{
		PackagePath:    "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:       "RunCheckRepo",
		Receiver:       "DupcodeRunner",
		CallerKind:     CallerKindMethod,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "CheckRepo", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath:    "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:       "RunCheckReport",
		Receiver:       "DupcodeRunner",
		CallerKind:     CallerKindMethod,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "CheckReport", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath:    "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:       "LoadBaseline",
		Receiver:       "DupcodeRunner",
		CallerKind:     CallerKindMethod,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "LoadBaseline", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath:    "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:       "VerifyBaseline",
		Receiver:       "DupcodeRunner",
		CallerKind:     CallerKindMethod,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "VerifyBaseline", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath:    "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:       "WriteBaseline",
		Receiver:       "DupcodeRunner",
		CallerKind:     CallerKindMethod,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "WriteBaseline", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath:    "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:       "CompareToBaseline",
		Receiver:       "DupcodeRunner",
		CallerKind:     CallerKindMethod,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
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
		PackagePath:    "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:       "runSharedDupcodeBaseline",
		Receiver:       "",
		CallerKind:     CallerKindPackageFunction,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "ValidateBaselineArtifact", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath:    "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:       "runSharedDupcodeBaseline",
		Receiver:       "",
		CallerKind:     CallerKindPackageFunction,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "CheckBaselineDriftFromReport", Kind: ProtectedPackageFunction,
		},
	},
}

// dupcodeInternalApprovedCallers records exact implementation-internal raw
// edges. They are validated bidirectionally like every external edge.
// CallerKind, ReferenceClass, and Cardinality are explicit per record.
var dupcodeInternalApprovedCallers = []ApprovedCaller{
	{
		PackagePath:    "github.com/s1onique/leamas/internal/factory/dupcode",
		Function:       "CheckReport",
		CallerKind:     CallerKindPackageFunction,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "CheckRepo", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath:    "github.com/s1onique/leamas/internal/factory/dupcode",
		Function:       "CheckBaselineDrift",
		CallerKind:     CallerKindPackageFunction,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "CheckReport", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath:    "github.com/s1onique/leamas/internal/factory/dupcode",
		Function:       "VerifyBaseline",
		CallerKind:     CallerKindPackageFunction,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "ValidateBaselineArtifact", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath:    "github.com/s1onique/leamas/internal/factory/dupcode",
		Function:       "CheckBaselineDrift",
		CallerKind:     CallerKindPackageFunction,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name: "CheckBaselineDriftFromReport", Kind: ProtectedPackageFunction,
		},
	},
}

func allApprovedCallers() []ApprovedCaller {
	capacity := len(ApprovedCallers) + len(dupcodeInternalApprovedCallers) + len(AdapterApprovedCallers) + len(GateApprovedCallers)
	out := make([]ApprovedCaller, 0, capacity)
	out = append(out, ApprovedCallers...)
	out = append(out, dupcodeInternalApprovedCallers...)
	out = append(out, AdapterApprovedCallers...)
	out = append(out, GateApprovedCallers...)
	return out
}

// IsApprovedCaller remains a strict declarative query for callers outside the
// full analysis pipeline. The canonical analysis additionally requires exact
// types.Object and reference-class identity.
func IsApprovedCaller(caller CallerIdentity, callee ProtectedSymbol) bool {
	if caller.Kind == "" {
		caller.Kind = CallerKindPackageFunction
		if caller.Receiver != "" {
			caller.Kind = CallerKindMethod
		}
	}
	for _, configured := range allApprovedCallers() {
		approval := normalizeApproval(configured)
		if approvalCallerIdentity(approval) == caller && approval.Callee == callee {
			return true
		}
	}
	return false
}
