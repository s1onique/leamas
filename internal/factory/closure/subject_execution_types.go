// SPDX-License-Identifier: Apache-2.0

package closure

// subject_execution_types.go owns the R6-A V2ExecuteRequest
// and V2ExecuteResult types. The struct definitions are
// large (a typed result carries every fact the live
// detached subject worktree can yield) and they belong
// here so the executor's flow stays linear and under the
// LLM-friendly line threshold.
//
// The types are referenced by:
//
//   - closure_protocol_v2_executor.go (production flow)
//   - subject_execution_observation.go (failure-path helper)
//   - subject_observation.go, subject_observation_inventory.go
//     (helper observation types)
//
// Splitting this from closure_protocol_v2_executor.go keeps
// the executor under the LLM-friendly 400-line threshold
// while preserving the single closure over the descriptor
// that ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"time"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// V2ExecuteRequest captures the inputs the executor needs to
// run checks against S^{tree}.
//
// R6-A adds TopologyFacts. The executor is a transport for
// the runtime-context topology authority; it MUST NOT
// recompute topology from the subject observations and it
// MUST NOT hard-code any relation. When the request omits
// the field the executor returns a zero-valued
// V2TopologyFacts and the result still transports the
// zero value.
//
// R6-B adds GateCollector and GateCaptureTemplate. The
// executor invokes the (optional) collector inside the
// live-S window and records the captured GateCapture via
// the V2ExecuteResult.GateObservationAvailable flag.
type V2ExecuteRequest struct {
	RepositoryRoot  string
	SubjectCommit   string
	SubjectTree     string
	EvidenceDir     string
	Checks          []PlanCheck
	CommandExecutor commandExecutor
	Now             func() time.Time
	// TopologyFacts is the runtime-context topology authority
	// that governed execution. The executor transports the
	// existing typed value into the result so downstream
	// consumers (R6-B/C) can read the facts the runner
	// already established. R6-A MUST NOT change topology
	// semantics.
	TopologyFacts V2TopologyFacts
	// GateCollector is the optional exactly-once gate
	// authority. Production passes the canonical
	// GateCollector; the executor invokes it inside the
	// live-S window (after the live identity / status /
	// refs / inventory observations, before the subject
	// worktree is removed) and transports the captured
	// GateCapture into V2ExecuteResult via the
	// GateObservationAvailable flag. A nil GateCollector
	// means the executor did not run a gate; the result
	// fields remain zero-valued.
	GateCollector *evidence.GateCollector
	// GateCaptureTemplate is the per-run request the
	// executor uses to invoke the collector. The executor
	// fills SubjectRoot from the live subject worktree
	// path before delegating to Capture so the runtime
	// surface of the gate matches the live-S worktree.
	GateCaptureTemplate evidence.GateCaptureRequest
}

// V2ExecuteResult captures the deterministic outputs of the
// subject-tree execution. CheckResults mirrors the v1 schema
// so the manifest can reuse the existing parser.
//
// R6-A adds the subject observation fields. The legacy
// ObservedTree/CheckResults/Evidence/CleanupError fields are
// preserved unchanged; new fields are additive so existing
// callers keep their wire contract.
//
// R6-B adds the gate capture fields. GateObservationAvailable
// is the explicit avail signal: a zero-valued GateCapture
// NEVER means the gate was observed; a successful capture
// reports Available=true with the populated GateCapture.
type V2ExecuteResult struct {
	// Legacy fields (unchanged).
	ObservedTree string
	CheckResults []CheckResult
	Evidence     []EvidenceRecord
	CleanupError string

	// R6-A: subject execution result fields. The executor
	// is the single authority for these facts.

	// SubjectWorktreePath is the actual path created by the
	// production worktree add. It is the canonical address
	// every other subject observation is relative to.
	SubjectWorktreePath string
	// SubjectHead is the live HEAD observed at the S
	// worktree via `git rev-parse HEAD`. It MUST equal the
	// request SubjectCommit when the observation is
	// available.
	SubjectHead string
	// SubjectTree is the live HEAD^{tree} observed at the S
	// worktree via `git rev-parse HEAD^{tree}`.
	SubjectTree string
	// SubjectDetached is the live detached state observed
	// via `git symbolic-ref -q HEAD`. The production
	// executor creates a --detach worktree, so the
	// authoritative value is true; the field is observed,
	// not hard-coded.
	SubjectDetached bool

	// StatusObservation is the live S-worktree status
	// captured via the canonical porcelain-v2 form. Empty
	// bytes with Available=true is a legitimate result and
	// is the canonical "clean worktree" signal.
	StatusObservation SubjectByteObservation
	// RefsObservation is the live S-worktree refs captured
	// via the canonical refs authority (snapshotCallerRefs).
	// Empty bytes with Available=true is a legitimate
	// result (no refs) and is NOT encoded as "unavailable".
	RefsObservation SubjectByteObservation

	// WorktreeInventoryBefore / AtSubject / After are the
	// three logically distinct inventory snapshots
	// required by Phase 7. AtSubject is the snapshot taken
	// after the S worktree was added; After is the snapshot
	// taken after production removed the S worktree.
	WorktreeInventoryBefore    SubjectWorktreeInventory
	WorktreeInventoryAtSubject SubjectWorktreeInventory
	WorktreeInventoryAfter     SubjectWorktreeInventory

	// SubjectRegistration is the canonical registration
	// from WorktreeInventoryAtSubject whose Path equals
	// SubjectWorktreePath. SubjectRegistrationAvailable
	// reports whether the lookup succeeded. The
	// (Path, Head) pair is the canonical identity Phase 8
	// and Phase 16 require.
	SubjectRegistration          SubjectWorktreeRegistration
	SubjectRegistrationAvailable bool

	// TopologyFacts transports the runtime-context topology
	// authority that governed execution. R6-A MUST NOT
	// recompute topology from the subject observations and
	// MUST NOT hard-code FAncestorOfS.
	TopologyFacts V2TopologyFacts

	// SubjectCleanupObserved is true when the executor
	// reached the cleanup path; false when cleanup was
	// never attempted (which the production executor only
	// does for input-validation failures, before any worktree
	// was created).
	SubjectCleanupObserved bool
	// SubjectCleanupError is the cleanup report's summary
	// when the cleanup was attempted. Empty when cleanup
	// succeeded.
	SubjectCleanupError string

	// R6-A: subject-observation diagnostics. The executor
	// captures every typed observation failure so downstream
	// consumers can audit the live-S window.
	SubjectObservationDiagnostics V2Diagnostics

	// R6-B: gate authority capture. The executor invokes
	// GateCollector (when supplied) inside the live-S
	// window and records the captured GateCapture here.
	// GateObservationAvailable is the explicit avail signal:
	// a zero-valued GateCapture NEVER means the gate was
	// observed; a successful capture reports Available=true
	// with the populated GateCapture. GateObservationError
	// captures the collector's error string when the gate
	// observation failed; GateObservationCause carries the
	// original typed error so downstream errors.Is checks
	// survive the string conversion (e.g.
	// evidence.ErrCollectorRequestMismatch).
	GateObservationAvailable bool
	GateCapture              evidence.GateCapture
	GateObservationError     string
	GateObservationCause     error
}
