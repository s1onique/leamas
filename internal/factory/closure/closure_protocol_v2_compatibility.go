// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_compatibility.go provides the closed
// compatibility authority for Plan Contract version × Closure
// Protocol version combinations. Only the supported matrix
// is accepted; every other combination emits a typed
// diagnostic and the typed V2Error.
//
// Splitting this from closure_protocol_v2.go keeps the file
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import "fmt"

// PlanContractVersion is the integer Plan Contract version
// extracted from the plan bytes. The supported values are
// closed; future plan contracts will be added here once the
// schema is finalised.
type PlanContractVersion int

const (
	PlanContractV1 PlanContractVersion = 1
)

// IsSupported reports whether the version is a known Plan
// Contract version.
func (v PlanContractVersion) IsSupported() bool {
	return v == PlanContractV1
}

// V2VersionCombination captures the two version axes a v2
// request must declare. Both fields are required before the
// runner accepts the request.
type V2VersionCombination struct {
	PlanContract    PlanContractVersion
	ClosureProtocol ClosureProtocolVersion
}

// IsSupported reports whether the combination is in the
// closed support matrix.
//
//	v1 + v1 -> supported (legacy ACT)
//	v1 + v2 -> supported (target ACT)
//	any other combination -> unsupported
func (c V2VersionCombination) IsSupported() bool {
	if !c.PlanContract.IsSupported() {
		return false
	}
	if !c.ClosureProtocol.IsSupported() {
		return false
	}
	switch c.PlanContract {
	case PlanContractV1:
		return c.ClosureProtocol == ClosureProtocolV1 ||
			c.ClosureProtocol == ClosureProtocolV2
	}
	return false
}

// ValidateV2VersionCombination returns nil when the supplied
// axes form a supported combination and a typed V2Error
// otherwise. The function never silently coerces or accepts a
// free-form int.
//
// Codes:
//   - unsupported_plan_contract_version
//   - unsupported_closure_protocol_version
//   - unsupported_plan_protocol_combination
func ValidateV2VersionCombination(plan PlanContractVersion, closure ClosureProtocolVersion) error {
	combo := V2VersionCombination{PlanContract: plan, ClosureProtocol: closure}
	if !plan.IsSupported() {
		return NewV2ErrorWith(V2CodeUnsupportedPlanContractVersion,
			fmt.Sprintf("plan contract version %d is not supported", int(plan)),
			"plan_contract_version", "")
	}
	if !closure.IsSupported() {
		return NewV2ErrorWith(V2CodeUnsupportedClosureProtocolVersion,
			fmt.Sprintf("closure protocol version %q is not supported", string(closure)),
			"closure_protocol_version", "")
	}
	if !combo.IsSupported() {
		return NewV2ErrorWith(V2CodeUnsupportedPlanProtocolComb,
			fmt.Sprintf("plan contract v%d + closure protocol %q is not a supported combination",
				int(plan), string(closure)),
			"plan_protocol_combination", "")
	}
	return nil
}
