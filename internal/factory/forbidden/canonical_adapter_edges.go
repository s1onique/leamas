// SPDX-License-Identifier: Apache-2.0

package forbidden

const protectedVerifierPackagePath = "github.com/s1onique/leamas/internal/factory/protectedverifier"

// AdapterApprovedCallers is the exact observed adapter-edge inventory. Dynamic
// dupcodeRunner interface calls are intentionally absent: only source edges to
// protectedverifier declaration objects belong here.
var AdapterApprovedCallers = []ApprovedCaller{
	// Gate adapter construction and forwarding methods.
	adapterApproval(gatePackagePath, "newProtectedDupcodeRunner", "", "NewDupcodeRunner", ProtectedPackageFunction, ""),
	adapterApproval(gatePackagePath, "LoadBaseline", "protectedDupcodeRunnerAdapter", "LoadBaseline", ProtectedMethod, "DupcodeRunner"),
	adapterApproval(gatePackagePath, "RunCheckRepo", "protectedDupcodeRunnerAdapter", "RunCheckRepo", ProtectedMethod, "DupcodeRunner"),
	adapterApproval(gatePackagePath, "RunCheckReport", "protectedDupcodeRunnerAdapter", "RunCheckReport", ProtectedMethod, "DupcodeRunner"),
	adapterApproval(gatePackagePath, "VerifyBaseline", "protectedDupcodeRunnerAdapter", "VerifyBaseline", ProtectedMethod, "DupcodeRunner"),
	adapterApproval(gatePackagePath, "WriteBaseline", "protectedDupcodeRunnerAdapter", "WriteBaseline", ProtectedMethod, "DupcodeRunner"),
	adapterApproval(gatePackagePath, "CompareToBaseline", "protectedDupcodeRunnerAdapter", "CompareToBaseline", ProtectedMethod, "DupcodeRunner"),

	// Canonical direct verifier implementations.
	adapterApproval(gatePackagePath, "dupCodeVerifier", "", "NewDupcodeRunner", ProtectedPackageFunction, ""),
	adapterApproval(gatePackagePath, "dupCodeVerifier", "", "LoadBaseline", ProtectedMethod, "DupcodeRunner"),
	adapterApproval(gatePackagePath, "dupCodeVerifier", "", "RunCheckReport", ProtectedMethod, "DupcodeRunner"),
	adapterApproval(gatePackagePath, "dupCodeVerifier", "", "CompareToBaseline", ProtectedMethod, "DupcodeRunner"),
	adapterApproval(gatePackagePath, "dupcodeBaselineVerifier", "", "NewDupcodeRunner", ProtectedPackageFunction, ""),
	adapterApproval(gatePackagePath, "dupcodeBaselineVerifier", "", "VerifyBaseline", ProtectedMethod, "DupcodeRunner"),

	// Lazy factorize wrapper-to-adapter edges.
	adapterApproval(gatePackagePath, "newFactorizeDupcodeAnalyzer", "", "NewAnalyzerFromAdapter", ProtectedPackageFunction, ""),
	adapterApproval(gatePackagePath, "readFactorizeDupcodeThresholds", "", "ReadBaselineThresholds", ProtectedPackageFunction, ""),

	// Protectedverifier same-package implementation edges.
	adapterApproval(protectedVerifierPackagePath, "analyzeThroughAdapter", "", "NewDupcodeRunner", ProtectedPackageFunction, ""),
	adapterApproval(protectedVerifierPackagePath, "analyzeThroughAdapter", "", "RunCheckRepo", ProtectedMethod, "DupcodeRunner"),
	adapterApproval(protectedVerifierPackagePath, "runSharedDupcodeVerify", "", "NewDupcodeRunner", ProtectedPackageFunction, ""),
	adapterApproval(protectedVerifierPackagePath, "runSharedDupcodeVerify", "", "LoadBaseline", ProtectedMethod, "DupcodeRunner"),
	adapterApproval(protectedVerifierPackagePath, "runSharedDupcodeVerify", "", "CompareToBaseline", ProtectedMethod, "DupcodeRunner"),
}

func adapterApproval(
	callerPackage, function, receiver, calleeName string,
	calleeKind ProtectedSymbolKind,
	calleeReceiver string,
) ApprovedCaller {
	callerKind := CallerKindPackageFunction
	if receiver != "" {
		callerKind = CallerKindMethod
	}
	return ApprovedCaller{
		PackagePath: callerPackage,
		Function:    function,
		Receiver:    receiver,
		CallerKind:  callerKind,
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerAdapter,
			PackagePath: protectedVerifierPackagePath,
			Name:        calleeName,
			Kind:        calleeKind,
			Receiver:    calleeReceiver,
		},
		ReferenceClass: refDirectCall,
		Cardinality:    1,
	}
}
