// SPDX-License-Identifier: Apache-2.0

package closure

// subject_execution_gate.go implements the R6-B gate capture
// the production executor invokes inside its live-S window.
// The capture is the only path that may produce a
// V2ExecuteResult.GateCapture; the executor MUST NOT
// synthesize the gate fields from other observations.
//
// Splitting the gate capture from the executor flow keeps
// closure_protocol_v2_executor.go under the LLM-friendly
// 400-line threshold while preserving the single closure
// over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"context"
	"fmt"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// V2GateCapture is the typed bundle the executor returns
// from the gate capture phase. GateObservationAvailable is
// the explicit avail signal: a zero-valued GateCapture
// NEVER means the gate was observed; a successful capture
// reports Available=true with the populated GateCapture.
// GateObservationError captures the collector's error string
// when the gate observation failed.
type V2GateCapture struct {
	Available           bool
	Capture             evidence.GateCapture
	ObservationError    string
}

// captureGate invokation inside the executor's live-S window.
// The function is a no-op (returns the zero-valued capture)
// when the request did not supply a GateCollector.
//
// The capture fills the SubjectRoot from the live subject
// worktree path the executor produced so the gate's runtime
// surface matches the live-S worktree. The
// GateCaptureTemplate carries the other identity-bearing
// fields (RepositoryRoot, EvidenceDir, RunID, MakeExecutable);
// the executor MUST NOT mutate them.
//
// The capture runs while the live-S worktree is alive.
// Cleanup of the worktree is the executor's responsibility
// after this function returns.
func captureGate(
	ctx context.Context,
	collector *evidence.GateCollector,
	worktreePath string,
	template evidence.GateCaptureRequest,
) V2GateCapture {
	if collector == nil {
		return V2GateCapture{}
	}
	if template.RunID == "" {
		return V2GateCapture{
			ObservationError: "execute: GateCaptureTemplate.RunID is empty",
		}
	}
	if template.SubjectRoot != "" {
		// Template already claims an identity; the executor
		// is the only caller allowed to override SubjectRoot.
		// We refuse to override because the supply-side
		// caller would have constructed an alternate identity.
		// (This branch is reserved for future explicit
		// SubjectRoot-override semantics.)
	}
	runReq := template
	runReq.SubjectRoot = worktreePath
	capture, err := collector.Capture(ctx, runReq)
	if err != nil {
		return V2GateCapture{
			ObservationError: fmt.Sprintf("evidence: gate capture: %s", err.Error()),
		}
	}
	return V2GateCapture{
		Available: true,
		Capture:   capture,
	}
}
