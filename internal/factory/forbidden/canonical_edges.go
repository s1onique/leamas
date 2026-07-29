// SPDX-License-Identifier: Apache-2.0

package forbidden

// ApprovedCallers defines exact approved caller-to-callee edges based on the
// real production declarations in internal/factory/protectedverifier/adapter.go.
//
// Source of truth:
//
//	(*DupcodeRunner).RunCheckRepo     → dupcode.CheckRepo        (adapter.go:32-33)
//	(*DupcodeRunner).RunCheckReport   → dupcode.CheckReport      (adapter.go:25-26)
//	(*DupcodeRunner).LoadBaseline     → dupcode.LoadBaseline     (adapter.go:39-40)
//	(*DupcodeRunner).VerifyBaseline   → dupcode.VerifyBaseline   (adapter.go:46-47)
//	(*DupcodeRunner).WriteBaseline    → dupcode.WriteBaseline    (adapter.go:53-54)
//	(*DupcodeRunner).CompareToBaseline → dupcode.CompareToBaseline (adapter.go:60-61)
//	<var-init:DefaultAnalyzer>        → dupcode.CheckRepo        (adapter.go:68)
//
// No wildcards. Function/Receiver must match exactly.
var ApprovedCallers = []ApprovedCaller{
	// *DupcodeRunner methods (6 edges)
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "RunCheckRepo",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name:        "CheckRepo", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "RunCheckReport",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name:        "CheckReport", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "LoadBaseline",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name:        "LoadBaseline", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "VerifyBaseline",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name:        "VerifyBaseline", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "WriteBaseline",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name:        "WriteBaseline", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "CompareToBaseline",
		Receiver:    "DupcodeRunner",
		Callee: ProtectedSymbol{
			PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name:        "CompareToBaseline", Kind: ProtectedPackageFunction,
		},
	},
	// DefaultAnalyzer var-init in protectedverifier (function-value capture)
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "<var-init:DefaultAnalyzer>",
		Receiver:    "",
		Callee: ProtectedSymbol{
			PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name:        "CheckRepo", Kind: ProtectedPackageFunction,
		},
	},
	// Function literals inside *DupcodeVerifierFactory.SharedDupCodeVerifier
	// shared_context.go:140 - calls dupcode.LoadBaseline (line 153) and dupcode.CompareToBaseline (line 187)
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "func@140:9",
		Receiver:    "",
		Callee: ProtectedSymbol{
			PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name:        "LoadBaseline", Kind: ProtectedPackageFunction,
		},
	},
	{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "func@140:9",
		Receiver:    "",
		Callee: ProtectedSymbol{
			PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
			Name:        "CompareToBaseline", Kind: ProtectedPackageFunction,
		},
	},
}

// IsApprovedCaller checks if a caller-callee edge is approved.
// Function and Receiver must match exactly. Empty Function in ApprovedCallers
// is treated as invalid (no wildcards).
func IsApprovedCaller(caller CallerIdentity, callee ProtectedSymbol) bool {
	for _, ac := range ApprovedCallers {
		if ac.PackagePath != caller.PackagePath {
			continue
		}
		if ac.Callee.PackagePath != callee.PackagePath ||
			ac.Callee.Name != callee.Name ||
			ac.Callee.Kind != callee.Kind ||
			ac.Callee.Receiver != callee.Receiver {
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
