// SPDX-License-Identifier: Apache-2.0

package forbidden

const gatePackagePath = "github.com/s1onique/leamas/internal/factory/gate"

// GateProtectedSymbols prevents the lazy factorize setup wrappers from becoming
// general pre-authority capability laundering points.
var GateProtectedSymbols = []ProtectedSymbol{
	{
		Layer:       AuthorityLayerGate,
		PackagePath: gatePackagePath,
		Name:        "newFactorizeDupcodeAnalyzer",
		Kind:        ProtectedPackageFunction,
	},
	{
		Layer:       AuthorityLayerGate,
		PackagePath: gatePackagePath,
		Name:        "readFactorizeDupcodeThresholds",
		Kind:        ProtectedPackageFunction,
	},
}

// GateApprovedCallers gives each setup wrapper one exact data-only caller.
var GateApprovedCallers = []ApprovedCaller{
	{
		PackagePath:    gatePackagePath,
		Function:       "productionFactorizeDupcodeDeps",
		CallerKind:     CallerKindPackageFunction,
		Callee:         GateProtectedSymbols[0],
		ReferenceClass: refFunctionValue,
		Cardinality:    1,
	},
	{
		PackagePath:    gatePackagePath,
		Function:       "productionFactorizeDupcodeDeps",
		CallerKind:     CallerKindPackageFunction,
		Callee:         GateProtectedSymbols[1],
		ReferenceClass: refFunctionValue,
		Cardinality:    1,
	},
}
