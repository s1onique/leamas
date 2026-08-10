// SPDX-License-Identifier: Apache-2.0

// Package evidence - plancontract_consumer.go is the B2-R2
// entry point for the canonical Plan Contract v1 decoder.
//
// B2-R2 motivation: the previous B2-R1 implementation
// mirrored the production parser inside the evidence package
// to avoid a test-only import cycle. Two decoders is two
// authorities for the same wire contract; the doctrine
// rejects that. The plancontract package is the leaf both
// consumers import. The closure runner also imports
// plancontract so the production decoder and the evidence
// decoder cannot diverge.
//
// The wrapper here adapts the leaf's minimal DecodeResult to
// the evidence package's canonical PlanCheckSpec list. The
// conversion is mechanical so the leaf can stay focused on
// the wire contract.
package evidence

import (
	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// productionDecodeClosurePlan is the B2-R2 authority entry
// point. It routes the supplied bytes through the canonical
// Plan Contract v1 decoder (plancontract.DecodeAndValidate)
// and projects the result into the evidence package's
// PlanCheckSpec list. The decoder enforces the MaxPlanBytes
// cap, rejects malformed JSON, rejects duplicate keys,
// rejects trailing values, and rejects unknown modes.
//
// The function is the only path used by the production
// decoder predicates. Tests that need to drive the decoder
// route through this helper so the production code path is
// the only path exercised.
func productionDecodeClosurePlan(bytes []byte) ([]PlanCheckSpec, error) {
	result, err := plancontract.DecodeAndValidate(bytes)
	if err != nil {
		return nil, err
	}
	out := make([]PlanCheckSpec, 0, len(result.Checks))
	for _, c := range result.Checks {
		out = append(out, PlanCheckSpec{
			ID:   c.ID,
			Mode: c.Mode,
		})
	}
	return out, nil
}

// deriveExpectedChecksFromPlanBytes is the candidate builder
// derivation helper. It decodes the supplied F:P bytes via
// the production Plan Contract decoder and returns the
// expected check set. When the bytes are empty or fail to
// decode the function returns an empty slice; the
// runtimeExpectedChecksDerivedFromPlanBytes predicate will
// reject the candidate in that case.
func deriveExpectedChecksFromPlanBytes(planBytes []byte) []PlanCheckSpec {
	if len(planBytes) == 0 {
		return nil
	}
	out, err := productionDecodeClosurePlan(planBytes)
	if err != nil {
		return nil
	}
	return out
}
