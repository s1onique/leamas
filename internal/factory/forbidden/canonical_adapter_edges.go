// SPDX-License-Identifier: Apache-2.0

package forbidden

// AdapterApprovedCallers defines the exact post-authority caller edges for
// adapter-layer symbols. Each edge corresponds to a real named caller
// declaration in internal/factory/gate or internal/factory/protectedverifier
// that constructs the protected adapter or invokes an exact adapter
// operation.
//
// No wildcards. Function/Receiver must match exactly. Package-level wildcards
// (all functions in gate) and anonymous literals (func@line:col) are
// forbidden. The list reflects the production topology after the
// CORRECTION02G-AUTHORITY-CLI-AND-CLOSURE-TRUTH correction:
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
//	  → exact raw dupcode operation (within approved adapter method only)
//
// External gate-package calls (named post-authority binder only):
//
//	run method on DupcodeVerifyBinder       → NewDupcodeRunner, LoadBaseline,
//	                                         RunCheckReport, CompareToBaseline
//	run method on DupcodeBaselineBinder     → NewDupcodeRunner, VerifyBaseline
//	run method on DupcodeUpdateBaselineBinder → NewDupcodeRunner, RunCheckReport,
//	                                         WriteBaseline
//	dupCodeVerifier                         → NewDupcodeRunner, LoadBaseline,
//	                                         RunCheckReport, CompareToBaseline
//	dupcodeBaselineVerifier                 → NewDupcodeRunner, VerifyBaseline
//	factorizeVerifiersWithAnalyzer          → NewAnalyzerFromAdapter,
//	                                         ReadBaselineThresholds
//
// Internal same-package adapter calls:
//
//	analyzeThroughAdapter           → NewDupcodeRunner, RunCheckRepo
//	NewAnalyzerFromAdapter          → (return analyzeThroughAdapter only;
//	                                 no protected call site)
//	runSharedDupcodeVerify          → NewDupcodeRunner, LoadBaseline,
//	                                 CompareToBaseline
//	runSharedDupcodeBaseline        → dupcode.ValidateBaselineArtifact,
//	                                 dupcode.CheckBaselineDriftFromReport
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

	// dupcodeRunnerAdapter (gate-package adapter type) forwards each
	// dupcodeRunnerAdapter method to its corresponding DupcodeRunner
	// method. The adapter is the only authorized gate-side caller of
	// these adapter methods, used by DupcodeUpdateBaselineBinder.run
	// via the newProtectedRunner hook.
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "LoadBaseline",
		Receiver:    "dupcodeRunnerAdapter",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "LoadBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "RunCheckReport",
		Receiver:    "dupcodeRunnerAdapter",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "RunCheckReport", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "VerifyBaseline",
		Receiver:    "dupcodeRunnerAdapter",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "VerifyBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "WriteBaseline",
		Receiver:    "dupcodeRunnerAdapter",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "WriteBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "CompareToBaseline",
		Receiver:    "dupcodeRunnerAdapter",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "CompareToBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
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
	// factorizeVerifiersWithAnalyzer (gate → adapter metadata reads)
	// ────────────────────────────────────────────────────────────────────
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "factorizeVerifiersWithAnalyzer",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "NewAnalyzerFromAdapter", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/gate",
		Function:    "factorizeVerifiersWithAnalyzer",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "ReadBaselineThresholds", Kind: ProtectedPackageFunction,
		},
	},

	// ────────────────────────────────────────────────────────────────────
	// protectedverifier.analyzeThroughAdapter (named analyzer function)
	// Internal adapter call — same-package, approved explicitly.
	// ────────────────────────────────────────────────────────────────────
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "analyzeThroughAdapter",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "NewDupcodeRunner", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "analyzeThroughAdapter",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "RunCheckRepo", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},

	// ────────────────────────────────────────────────────────────────────
	// protectedverifier.runSharedDupcodeVerify (named package function)
	// Shared-context closure body extracted to a named function so the
	// policy scanner can resolve caller identity to a real declaration.
	// ────────────────────────────────────────────────────────────────────
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "runSharedDupcodeVerify",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "NewDupcodeRunner", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "runSharedDupcodeVerify",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "LoadBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "runSharedDupcodeVerify",
		Receiver:    "",
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
			Name:        "CompareToBaseline", Kind: ProtectedMethod, Receiver: "DupcodeRunner",
		},
	},

	// ────────────────────────────────────────────────────────────────────
	// protectedverifier.runSharedDupcodeBaseline (named package function)
	// Uses raw dupcode helpers for validation and drift; approved via
	// dupcode-internal approvals list (see canonical_edges.go).
	// ────────────────────────────────────────────────────────────────────
}
