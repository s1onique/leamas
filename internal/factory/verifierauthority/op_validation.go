// SPDX-License-Identifier: Apache-2.0

package verifierauthority

import (
	"fmt"
	"strings"
)

// ReasonCodeOperationUnknown is reported when an operation value is not
// one of the two accepted canonical strings.
const ReasonCodeOperationUnknown = "verifier_operation_unknown"

// ValidateOperation reports whether operation is one of the two
// canonical operation strings. Empty strings, whitespace, mixed case,
// and unknown values are rejected. Callers must invoke this before any
// observer call, runner construction, or authority validation.
func ValidateOperation(operation VerifierOperation) error {
	switch strings.TrimSpace(string(operation)) {
	case string(OperationVerify), string(OperationUpdateBaseline):
		// exact-match accept: reject any whitespace or case variant
		// that did not exactly equal the canonical constant.
		if string(operation) != string(OperationVerify) && string(operation) != string(OperationUpdateBaseline) {
			return &AuthorityError{
				Operation:  operation,
				ReasonCode: ReasonCodeOperationUnknown,
				Message: fmt.Sprintf(
					"unknown verifier operation %q (only %q and %q accepted)",
					operation, OperationVerify, OperationUpdateBaseline,
				),
			}
		}
		return nil
	default:
		return &AuthorityError{
			Operation:  operation,
			ReasonCode: ReasonCodeOperationUnknown,
			Message: fmt.Sprintf(
				"unknown verifier operation %q (only %q and %q accepted)",
				operation, OperationVerify, OperationUpdateBaseline,
			),
		}
	}
}
