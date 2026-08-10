// SPDX-License-Identifier: Apache-2.0

// closure_evidence_publication_runner_adapter.go owns the
// non-publishing runner entry point that produces the B2
// evidence inputs from authoritative runner sources.
//
// R6-B invariant: every field in V2ExecutionObservation
// comes from a runner source. No field is synthesised.
// The orchestrator's B2 inputs come from this struct; a
// caller cannot smuggle a fabricated CandidateInputs bundle
// past the B2 barrier.
//
// The previous synthetic adapter (SourceClean/SourceDetached/
// OutputOutsideAllWorktrees/Executable literals, hard-coded
// "PASS" classification, hard-coded InvocationCount == 1)
// has been removed. The current adapter transports the
// production B1 binary authority verbatim and derives the
// gate authority from the actual GateCapture the production
// executor captured inside the live-S window.
package closure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// V2ExecutionObservation is the authoritative execution record
// the B2 publication barrier requires. Every field is
// derived from sources the V2 runner already consults; no
// field is synthesised.
type V2ExecutionObservation struct {
	Manifest     V2Manifest
	Runtime      evidence.RuntimeAuthority
	Results      []evidence.CheckResult
	Gate         evidence.GateAuthority
	Binary       evidence.BinaryAuthority
	CallerBefore evidence.CallerStateSnapshot
	CallerAfter  evidence.CallerStateSnapshot
	Cleanup      evidence.CleanupAuthority
}

// mergePublicationErrorDiags enriches an inner-runner error
// with the AFTER-snapshot diagnostics before returning.
func mergePublicationErrorDiags(err error, afterSnap v2CallerStateSnapshot) error {
	v2err, ok := err.(*V2Error)
	if !ok {
		return err
	}
	post := append(V2Diagnostics{}, afterSnap.Diagnostics...)
	v2err.Diags = append(v2err.Diags, post...)
	return v2err
}

// RunClosureProtocolV2Execute is the non-publishing public
// entry point. It runs the production V2 runner with the
// immutable-subject topology (F < S), captures the B2
// evidence inputs from runner sources, and returns the typed
// observation. It MUST NOT call AtomicWriteV2Manifest.
//
// The R6-B integration routes the B1 binary through
// BuildExactSubjectBinary and the gate through the
// production GateCollector. No field is synthesised.
func RunClosureProtocolV2Execute(
	ctx context.Context,
	req V2Request,
	identity V2BinaryIdentity,
) (V2Manifest, V2ExecutionObservation, error) {
	return RunClosureProtocolV2ExecuteWithDeps(ctx, req, identity, RunClosureProtocolV2ExecuteDeps{})
}

// RunClosureProtocolV2ExecuteDeps captures the optional
// seams the R6-B failure matrix and umbrella tests use to
// inject fake B1 builds, fake GateCollector runners, and
// pre-set binary output roots.
type RunClosureProtocolV2ExecuteDeps struct {
	// BuildFn is the production B1 path. The default is
	// BuildExactSubjectBinary. Tests may inject a fake.
	BuildFn func(ctx context.Context, req ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error)
	// NewCollectorFn is the production GateCollector
	// constructor. The default is evidence.NewGateCollector.
	NewCollectorFn func(runner evidence.CommandRunner) *evidence.GateCollector
	// CommandRunner is the CommandRunner the GateCollector
	// uses. The default is nil (evidence.NewGateCollector
	// defaults to OsRunner so a real `factory gate` runs).
	CommandRunner evidence.CommandRunner
	// OutputRoot is the per-run external binary output
	// root. The default is a fresh temp directory the
	// integration creates and removes after the
	// authoritative observation is built.
	OutputRoot string
	// OutputName is the binary name written inside
	// OutputRoot. The default is "leamas".
	OutputName string
	// RunID is the closure execution identity. The
	// default is "closure-execute".
	RunID string
	// EvidenceDir is the per-run external gate scratch
	// directory. The default is a fresh temp directory.
	EvidenceDir string
	// GateACTOwnedPaths lists the paths the current ACT
	// owns. The gate classification uses the same path
	// set the ACT-owned pass/fail verdict would use.
	GateACTOwnedPaths []string
	// GateBaselineFindings is the typed bundle of
	// pre-existing findings the classifier uses to
	// distinguish unchanged baseline findings from new
	// findings. The integration threads this through
	// without alteration; the rig did not invent a new
	// baseline authority.
	GateBaselineFindings []evidence.GateFinding
	// ClassifierCalls is a typed counter the failure
	// matrix uses to verify the classifier was reached
	// exactly once. The integration increments it
	// around the ClassifyACTOwnedGate call; tests
	// assert the count for PASS / FAIL / UNAVAILABLE
	// rows.
	ClassifierCalls *int
	// GitClient is the production git client used to
	// resolve S and S^{tree} before B1. The default is
	// RealGit. Tests may inject a fake.
	GitClient gitClient
	// (OPERATIONAL CLEANUP NOTE: the external binary
	// OutputRoot is NOT cleaned up by this non-publishing
	// adapter. The R6-B frozen lifetime is:
	//
	//   B1 binary
	//   -> GateCollector
	//   -> B2 observation/candidate proof
	//   -> ONLY THEN external OutputRoot cleanup
	//
	// The cleanup is operational hygiene that belongs to
	// the caller / orchestrator (R6-C). Operators can read
	// obs.Binary.BinaryPath and remove the external binary
	// after the B2 candidate has been verified.)
}

// RunClosureProtocolV2ExecuteWithDeps is the seam over
// RunClosureProtocolV2Execute. The seam exists so the
// R6-B failure matrix and umbrellas can inject fake
// B1 builds, fake gate runners, and pre-set binary output
// roots without rewriting the production flow.
func RunClosureProtocolV2ExecuteWithDeps(
	ctx context.Context,
	req V2Request,
	identity V2BinaryIdentity,
	seamDeps RunClosureProtocolV2ExecuteDeps,
) (V2Manifest, V2ExecutionObservation, error) {
	var zero V2Manifest
	if strings.TrimSpace(req.RepositoryRoot) == "" {
		return zero, V2ExecutionObservation{}, errors.New("execute: repository root is required")
	}
	if strings.TrimSpace(req.SubjectCommit) == "" || strings.TrimSpace(req.FreezeCommit) == "" || strings.TrimSpace(req.PlanPath) == "" {
		return zero, V2ExecutionObservation{}, errors.New("execute: subject, freeze, and plan_path are required")
	}
	buildFn := seamDeps.BuildFn
	if buildFn == nil {
		buildFn = BuildExactSubjectBinary
	}
	newCollector := seamDeps.NewCollectorFn
	if newCollector == nil {
		newCollector = evidence.NewGateCollector
	}
	gitClient := seamDeps.GitClient
	if gitClient == nil {
		gitClient = RealGit{}
	}
	outputRoot := seamDeps.OutputRoot
	if outputRoot == "" {
		var err error
		outputRoot, err = defaultExternalBinaryOutputRoot()
		if err != nil {
			return zero, V2ExecutionObservation{}, fmt.Errorf("execute: create external binary output root: %w", err)
		}
	}
	outputName := seamDeps.OutputName
	if outputName == "" {
		outputName = "leamas"
	}
	runID := seamDeps.RunID
	if runID == "" {
		runID = "closure-execute"
	}
	evidenceDir := seamDeps.EvidenceDir
	if evidenceDir == "" {
		var err error
		evidenceDir, err = defaultGateEvidenceDir()
		if err != nil {
			return zero, V2ExecutionObservation{}, fmt.Errorf("execute: create gate evidence dir: %w", err)
		}
	}

	// Phase 0: Resolve S and S^{tree} to literal OIDs BEFORE
	// calling B1. The R6-B contract mandates that B1 receive
	// literal commit and tree OIDs, not revision expressions.
	// A revision expression would force B1 to re-resolve the
	// tree and produce potentially different SHA-256 outputs.
	subjectCommitOID, err := runGitValue(ctx, gitClient, req.RepositoryRoot, "rev-parse", "--verify", "--end-of-options", req.SubjectCommit+"^{commit}")
	if err != nil {
		return zero, V2ExecutionObservation{}, fmt.Errorf("execute: resolve subject commit: %w", err)
	}
	subjectTreeOID, err := runGitValue(ctx, gitClient, req.RepositoryRoot, "rev-parse", "--verify", "--end-of-options", subjectCommitOID+"^{tree}")
	if err != nil {
		return zero, V2ExecutionObservation{}, fmt.Errorf("execute: resolve subject tree: %w", err)
	}

	// Phase 1: Build the exact-S binary via the B1
	// production authority. The B1 internal cleanup runs
	// as part of the production flow; the external
	// OutputRoot leamas binary remains alive for the gate
	// and the B2 observation/candidate proof.
	subjectTreeRes, err := buildFn(ctx, ExactSubjectBinaryRequest{
		RepositoryRoot: req.RepositoryRoot,
		SubjectCommit:  subjectCommitOID,
		SubjectTree:    subjectTreeOID,
		OutputRoot:     outputRoot,
		OutputName:     outputName,
	})
	if err != nil {
		return zero, V2ExecutionObservation{}, fmt.Errorf("execute: build exact subject binary: %w", err)
	}
	// CORRECTION06 fail-closed check: the B1 result must
	// correspond to the resolved literal S / S^{tree}
	// authority. A returned result that disagrees with the
	// request (wrong BinaryCommit, empty source identity,
	// wrong source tree, modified binary, non-detached
	// source, etc.) is an owning R6-B binary-authority
	// failure family.
	if vErr := validateExactSubjectBinaryResult(
		subjectCommitOID, subjectTreeOID, subjectTreeRes,
	); vErr != nil {
		return zero, V2ExecutionObservation{}, vErr
	}
	binaryAuthority := binaryAuthorityFromBuild(subjectTreeRes)

	// Phase 2: Construct the production GateCollector.
	collector := newCollector(seamDeps.CommandRunner)

	// Phase 3: Default deps; override only the gate
	// collector. The collector is wired into the
	// executor via V2ExecuteRequest.GateCollector; the
	// executor invokes it inside the live-S window.
	deps := DefaultV2RunnerDeps()
	deps.Git = gitClient
	deps.BinaryIdentity = identity
	deps.GateCollector = collector
	deps.GateCaptureTemplate = evidence.GateCaptureRequest{
		RepositoryRoot: req.RepositoryRoot,
		EvidenceDir:    evidenceDir,
		RunID:          runID,
		// argv[0] is the binary; the Collector takes
		// argv[0] separately via Run(ctx, argv[0],
		// argv[1:], dir, env). SubjectRoot is bound
		// from the live worktree path by the executor;
		// the template MUST leave it empty.
		SubjectRoot: "",
		MakeExecutable: []string{
			subjectTreeRes.BinaryPath,
			"factory", "gate", "--lane=fast",
		},
	}
	// Phase 4: Capture before snapshot.
	callerBeforeSnap := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseBefore)
	if !callerBeforeSnap.Available {
		return zero, V2ExecutionObservation{}, &V2Error{Diags: callerBeforeSnap.Diagnostics}
	}
	// Phase 5: Run the inner executor. The executor
	// captures the gate inside the live-S window via
	// captureGate; the captured GateCapture is stored
	// in V2ExecuteResult.GateCapture.
	execResult, err := runClosureProtocolV2InnerForExecute(ctx, req, deps, callerBeforeSnap.State, collector)
	if err != nil {
		callerAfterSnap := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseAfter)
		return zero, V2ExecutionObservation{}, mergePublicationErrorDiags(err, callerAfterSnap)
	}
	// Phase 6: Capture after snapshot.
	callerAfterSnap := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseAfter)
	if !callerAfterSnap.Available {
		return execResult.Manifest, V2ExecutionObservation{}, &V2Error{Diags: callerAfterSnap.Diagnostics}
	}
	if drift := callerBeforeSnap.State.Diff(callerAfterSnap.State); len(drift) > 0 {
		return execResult.Manifest, V2ExecutionObservation{}, &V2Error{Diags: drift}
	}
	// Phase 7: Reload the F:P bytes for the B2 candidate
	// builder. The loader's BlobOID is the authoritative
	// source; the integration MUST NOT recompute it.
	frozen, ferr := deps.Loader.LoadFrozenPlan(ctx, req.RepositoryRoot, req.FreezeCommit, req.PlanPath)
	if ferr != nil {
		return execResult.Manifest, V2ExecutionObservation{}, fmt.Errorf("execute: reload F:P for B2: %w", ferr)
	}
	planBytes := append([]byte(nil), frozen.Bytes...)

	// Phase 8: Build the B2 gate authority from the
	// captured GateCapture. The classification is the
	// ACT-owned verdict built from the actual observed
	// status / findings / lane flags.
	if !execResult.Result.GateObservationAvailable {
		// CORRECTION06: gate observation failure is its own
		// failure family. A spawn/start failure or any
		// other capture-time error must NOT be folded into
		// generic B2 incompleteness; it must surface as
		// the typed gate-observation failure so the R6-B
		// failure matrix can own the row. The original
		// typed error (e.g. evidence.ErrCollectorRequestMismatch)
		// is preserved as the wrapped cause so downstream
		// errors.Is checks still work.
		return execResult.Manifest, V2ExecutionObservation{}, &V2Error{
			Diags: V2Diagnostics{{
				Code:         V2CodeR6BGateObservationFailed,
				Message:      "gate capture did not produce a valid observation: " + execResult.Result.GateObservationError,
				PropertyName: "gate_observation",
				Detail:       execResult.Result.GateObservationError,
			}},
			Cause: execResult.Result.GateObservationCause,
		}
	}
	if collector.Calls() != 1 {
		return execResult.Manifest, V2ExecutionObservation{}, fmt.Errorf("execute: gate invocation count %d != 1", collector.Calls())
	}
	// R6-B-CORRECTION04: the classifier is the SINGLE
	// authority for the gate verdict. The integration
	// constructs ClassificationInputs from the actual
	// GateCapture and lets ClassifyACTOwnedGate produce
	// PASS / FAIL / UNAVAILABLE. The early transport-
	// rejection returns are removed so the classifier
	// is reached for every row.
	gate := execResult.Result.GateCapture
	_ = gate
	// R6-B-CORRECTION04: wire the actual baseline-findings
	// authority and increment the classifier-call counter
	// so the failure matrix can prove the classifier was
	// reached exactly once.
	if seamDeps.ClassifierCalls != nil {
		*seamDeps.ClassifierCalls++
	}
	classification, _ := classifyCapturedGate(
		execResult.Result.GateCapture,
		seamDeps.GateACTOwnedPaths,
		seamDeps.GateBaselineFindings,
		nil,
	)
	// CORRECTION06: surface classifier FAIL and
	// UNAVAILABLE as the owning typed R6-B integration
	// failure family. The authoritative capture is still
	// in obs.Gate so the B2 barrier can see the
	// classification the integration would have published.
	if classification == evidence.ACTOwnedFail {
		return execResult.Manifest, V2ExecutionObservation{}, NewV2ErrorWith(V2CodeR6BGateClassificationFailed,
			"gate classifier returned FAIL; ACT-owned finding intersected or new finding introduced",
			"gate_classification", string(classification))
	}
	if classification == evidence.ACTOwnedUnavailable {
		return execResult.Manifest, V2ExecutionObservation{}, NewV2ErrorWith(V2CodeR6BGateClassificationUnavailable,
			"gate classifier returned UNAVAILABLE; lane is missing, timed out, or truncated",
			"gate_classification", string(classification))
	}
	gateAuthority := evidence.GateAuthority{
		ObservedStatus:       execResult.Result.GateCapture.ExecGateObservedStatus,
		Classification:       string(classification),
		InvocationCount:      collector.Calls(),
		RepositoryRoot:       req.RepositoryRoot,
		SubjectRoot:          execResult.Result.SubjectWorktreePath,
		SubjectExecutionRoot: execResult.Result.SubjectWorktreePath,
		TimedOut:             execResult.Result.GateCapture.TimedOut,
		StdoutTruncated:      execResult.Result.GateCapture.StdoutTruncated,
		StderrTruncated:      execResult.Result.GateCapture.StderrTruncated,
	}
	if execResult.Result.GateObservationError != "" {
		gateAuthority.Error = execResult.Result.GateObservationError
	}

	// Phase 9: Map the V2 manifest's CheckResults into
	// the B2 Result slice. The V2 manifest records
	// {ID, Mode, Outcome, ExitCode, ...} per check; the
	// B2 candidate builder needs the same fields
	// translated into the evidence CheckResult shape.
	// The map is a direct transport; no field is
	// synthesised.
	b2Results := make([]evidence.CheckResult, 0, len(execResult.Manifest.CheckResults))
	for _, r := range execResult.Manifest.CheckResults {
		b2Results = append(b2Results, evidence.CheckResult{
			CheckID:  r.ID,
			Mode:     r.Mode,
			Outcome:  r.Outcome,
			ExitCode: derefIntPtr(r.ExitCode),
		})
	}
	// CleanupAuthority is the B1 INTERNAL build-worktree
	// cleanup outcome (per the frozen contract). The
	// external binary OutputRoot is operational hygiene
	// that runs AFTER the authoritative observation has
	// been assembled; its failure is reported separately
	// in the OperationalCleanupError field, NOT folded
	// into the canonical B2 CleanupAuthority.
	cleanup := evidence.CleanupAuthority{
		SubjectCleanupError: execResult.Result.SubjectCleanupError,
		BinaryCleanupError:  subjectTreeRes.CleanupError,
	}
	// CORRECTION06: the subject-cleanup authority lives in
	// the R6-A executor result. A failure there is the
	// owning subject-cleanup failure family. The check
	// runs AFTER the gate has executed and the B2 candidate
	// would otherwise be published.
	if vErr := validateSubjectCleanupOutcome(execResult.Result); vErr != nil {
		return execResult.Manifest, V2ExecutionObservation{}, vErr
	}
	b2Before := evidence.CallerStateSnapshot{
		Available:             true,
		Head:                  callerBeforeSnap.State.HEADCommit,
		Tree:                  callerBeforeSnap.State.HEADTree,
		StatusHash:            sha256OfBytes([]byte(callerBeforeSnap.State.StatusPorcelain)),
		RefsHash:              sha256OfBytes([]byte(callerBeforeSnap.State.RefsBytes)),
		WorktreeInventoryHash: sha256OfWorktreeInventory(callerBeforeSnap.State.WorktreeRegistrations),
	}
	b2After := evidence.CallerStateSnapshot{
		Available:             true,
		Head:                  callerAfterSnap.State.HEADCommit,
		Tree:                  callerAfterSnap.State.HEADTree,
		StatusHash:            sha256OfBytes([]byte(callerAfterSnap.State.StatusPorcelain)),
		RefsHash:              sha256OfBytes([]byte(callerAfterSnap.State.RefsBytes)),
		WorktreeInventoryHash: sha256OfWorktreeInventory(callerAfterSnap.State.WorktreeRegistrations),
	}
	obs := V2ExecutionObservation{
		Manifest: execResult.Manifest,
		Runtime: evidence.RuntimeAuthority{
			RepositoryRoot:       req.RepositoryRoot,
			FreezeCommit:         req.FreezeCommit,
			FreezeTree:           execResult.Manifest.FreezeTree,
			SubjectCommit:        execResult.Manifest.SubjectCommit,
			SubjectTree:          execResult.Manifest.SubjectTree,
			SubjectExecutionRoot: execResult.Result.SubjectWorktreePath,
			ExecutionTree:        execResult.Manifest.ExecutionTree,
			PlanPath:             frozen.Path,
			PlanBlob:             frozen.BlobOID,
			PlanSHA256:           frozen.SHA256,
			PlanBytes:            planBytes,
			FAncestorOfSVerified: execResult.Result.TopologyFacts.Classify() == V2RelationFreezeBeforeSubject,
		},
		Results:      b2Results,
		Gate:         gateAuthority,
		Binary:       binaryAuthority,
		CallerBefore: b2Before,
		CallerAfter:  b2After,
		Cleanup:      cleanup,
	}
	// R6-B-CORRECTION02: the external binary OutputRoot
	// is intentionally NOT cleaned up by this non-publishing
	// adapter. The R6-B frozen lifetime is:
	//
	//   B1 binary
	//   -> GateCollector
	//   -> B2 observation/candidate proof
	//   -> ONLY THEN external OutputRoot cleanup
	//
	// Cleaning up inside the adapter would delete the
	// exact binary referenced by obs.Binary.BinaryPath
	// before the caller has verified the B2 candidate,
	// breaking the B2 correlation. The cleanup is
	// operational hygiene that belongs to the caller /
	// orchestrator (R6-C). The BinaryPath handed back
	// points to the live binary file until the cleanup
	// owner removes it.
	return execResult.Manifest, obs, nil
}

// executorResultBundle is the inner-runner return bundle
// the R6-B adapter wraps. The struct is package-private
// because it is an internal sequencing convenience.
type executorResultBundle struct {
	Manifest V2Manifest
	Result   V2ExecuteResult
}

// runClosureProtocolV2InnerForExecute runs the inner
// executor with the gate collector wired in. The
// inner executor invokes the collector inside the
// live-S window and stores the captured GateCapture
// in V2ExecuteResult.GateCapture.
func runClosureProtocolV2InnerForExecute(
	ctx context.Context,
	req V2Request,
	deps V2RunnerDeps,
	callerBefore v2CallerState,
	collector *evidence.GateCollector,
) (executorResultBundle, error) {
	// Phase 3 (CORRECTION01): detached output locations.
	if err := EnforceDetachedV2Outputs(req); err != nil {
		return executorResultBundle{}, err
	}
	clean, cleanErr := workingTreeClean(ctx, deps.Git, req.RepositoryRoot)
	if cleanErr != nil {
		return executorResultBundle{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("inspect caller worktree: %s", cleanErr.Error()),
			"caller_worktree", cleanErr.Error())
	}
	if !clean {
		return executorResultBundle{}, NewV2ErrorWith(V2CodeCallerWorktreeDirty,
			"caller worktree must be clean before v2 run",
			"caller_worktree", "")
	}
	callerHead, err := runGitValue(ctx, deps.Git, req.RepositoryRoot, "rev-parse", "HEAD^{commit}")
	if err != nil {
		return executorResultBundle{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("read caller HEAD: %s", err.Error()),
			"caller_head", err.Error())
	}
	facts, err := deps.Topology.ResolveTopology(ctx, req.RepositoryRoot, req.SubjectCommit, req.FreezeCommit)
	if err != nil {
		return executorResultBundle{}, err
	}
	if !facts.SubjectResolved || !facts.FreezeResolved {
		return executorResultBundle{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("topology resolution incomplete: subject=%v freeze=%v relation=%s",
				facts.SubjectResolved, facts.FreezeResolved, string(facts.Classify())),
			"topology", string(facts.Classify()))
	}
	subjectCommit := facts.SubjectCommitValue()
	freezeCommit := facts.FreezeCommitValue()
	subjectTree, err := runGitValue(ctx, deps.Git, req.RepositoryRoot, "rev-parse", "--verify", "--end-of-options", subjectCommit+"^{tree}")
	if err != nil {
		return executorResultBundle{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("resolve subject tree: %s", err.Error()),
			"subject_tree", err.Error())
	}
	freezeTree, err := runGitValue(ctx, deps.Git, req.RepositoryRoot, "rev-parse", "--verify", "--end-of-options", freezeCommit+"^{tree}")
	if err != nil {
		return executorResultBundle{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("resolve freeze tree: %s", err.Error()),
			"freeze_tree", err.Error())
	}
	outcome := dispatchClosureTopology(req.ClosureProtocolVersion, executionTopologyFreezeBeforeSubject, facts)
	if !outcome.Accepted {
		return executorResultBundle{}, &V2Error{Diags: V2Diagnostics{{
			Code:         outcome.Code,
			Message:      outcome.Message,
			PropertyName: "topology",
			Detail:       string(outcome.Relation),
		}}}
	}
	frozen, err := deps.Loader.LoadFrozenPlan(ctx, req.RepositoryRoot, freezeCommit, req.PlanPath)
	if err != nil {
		return executorResultBundle{}, err
	}
	if req.OptionalWorkingPlanAssertion != "" {
		if err := enforceWorkingPlanAssertion(req.OptionalWorkingPlanAssertion, frozen); err != nil {
			return executorResultBundle{}, err
		}
	}
	plan, _, err := parsePlanBytes(frozen.Bytes)
	if err != nil {
		return executorResultBundle{}, NewV2ErrorWith(V2CodeFrozenPlanNotBlob,
			fmt.Sprintf("parse frozen plan: %s", err.Error()),
			"plan_bytes", err.Error())
	}
	if !PlanContractVersion(plan.ContractVersion).IsSupported() {
		return executorResultBundle{}, NewV2ErrorWith(V2CodeUnsupportedPlanContractVersion,
			fmt.Sprintf("frozen plan contract version %d is not supported", plan.ContractVersion),
			"plan_contract_version", "")
	}
	if err := ValidateV2VersionCombination(PlanContractVersion(plan.ContractVersion), req.ClosureProtocolVersion); err != nil {
		return executorResultBundle{}, err
	}
	if PlanContractVersion(plan.ContractVersion) != PlanContractVersion(req.PlanContractVersion) {
		return executorResultBundle{}, NewV2ErrorWith(V2CodeUnsupportedPlanProtocolComb,
			fmt.Sprintf("request plan contract version %d does not match frozen plan version %d",
				req.PlanContractVersion, plan.ContractVersion),
			"plan_contract_version", fmt.Sprintf("request=%d frozen=%d", req.PlanContractVersion, plan.ContractVersion))
	}
	if _, err := ValidateFrozenPlanV2(frozen.Bytes); err != nil {
		return executorResultBundle{}, err
	}
	if err := os.MkdirAll(req.EvidenceDirectory, 0o700); err != nil {
		return executorResultBundle{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("mkdir evidence dir: %s", err.Error()),
			"evidence_directory", err.Error())
	}
	execResult, err := deps.Executor.ExecuteSubjectChecks(ctx, V2ExecuteRequest{
		RepositoryRoot:      req.RepositoryRoot,
		SubjectCommit:       subjectCommit,
		SubjectTree:         subjectTree,
		EvidenceDir:         req.EvidenceDirectory,
		Checks:              plan.Checks,
		CommandExecutor:     deps.Commands,
		Now:                 deps.Now,
		TopologyFacts:       facts,
		GateCollector:       collector,
		GateCaptureTemplate: deps.GateCaptureTemplate,
	})
	if err != nil {
		return executorResultBundle{}, err
	}
	if execResult.ObservedTree != subjectTree {
		return executorResultBundle{}, NewV2ErrorWith(V2CodeExecutionTreeMismatch,
			fmt.Sprintf("executor observed tree %s does not match subject tree %s",
				execResult.ObservedTree, subjectTree),
			"execution_tree", execResult.ObservedTree)
	}
	manifest, err := NewV2Manifest(V2ManifestBuild{
		ClosureProtocolVersion: req.ClosureProtocolVersion,
		PlanContractVersion:    PlanContractVersion(plan.ContractVersion),
		SubjectCommit:          subjectCommit,
		SubjectTree:            subjectTree,
		FreezeCommit:           freezeCommit,
		FreezeTree:             freezeTree,
		PlanPath:               frozen.Path,
		PlanBlob:               frozen.BlobOID,
		PlanSHA256:             frozen.SHA256,
		PlanBytes:              frozen.Bytes,
		ExecutionTree:          subjectTree,
		CallerHead:             callerHead,
		BinaryIdentity:         deps.BinaryIdentity,
		PlanChecks:             plan.Checks,
		ExecutionResults:       execResult.CheckResults,
		Evidence:               execResult.Evidence,
	})
	if err != nil {
		return executorResultBundle{}, err
	}
	_ = callerBefore
	return executorResultBundle{Manifest: manifest, Result: execResult}, nil
}

// binaryAuthorityFromBuild maps the B1 build observability
// into the canonical B2 BinaryAuthority. The mapping is a
// direct transport; no field is synthesised.
func binaryAuthorityFromBuild(b ExactSubjectBinaryResult) evidence.BinaryAuthority {
	return evidence.BinaryAuthority{
		BinaryPath:                b.BinaryPath,
		BinarySHA256:              b.BinarySHA256,
		BinaryCommit:              b.BinaryCommit,
		BinaryModified:            b.BinaryModified,
		SourceCommit:              b.SourceCommit,
		SourceTree:                b.SourceTree,
		SourceClean:               b.SourceClean,
		SourceDetached:            b.SourceDetached,
		OutputOutsideAllWorktrees: b.OutputOutsideAllWorktrees,
		Executable:                b.Executable,
	}
}

// derefIntPtr returns the dereferenced int or 0 for a nil.
func derefIntPtr(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// sha256OfBytes returns the lowercase hex SHA-256 of the
// supplied bytes. An empty byte stream is a valid observation
// whose authority hash is the SHA-256 of zero bytes — never
// an empty string.
func sha256OfBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// sha256OfWorktreeInventory hashes the canonical (Path, HEAD)
// representation of the runner's worktree registrations.
// The (Path, HEAD) pair is the R6-A canonical identity each
// registration records; using it directly preserves the
// authority the R6-A executor observed. Two registrations
// with the same path but different HEAD OIDs produce
// different hashes. Entries are sorted by (Path, HEAD) so
// different insertion orders produce the same hash.
func sha256OfWorktreeInventory(invs v2WorktreeRegistrationSet) string {
	if len(invs) == 0 {
		// Empty observation is a valid empty registration
		// set; the canonical authority hash is the SHA-256
		// of the versioned empty-payload marker.
		sum := sha256.Sum256([]byte("worktree-inventory-v1\n"))
		return hex.EncodeToString(sum[:])
	}
	entries := make([]string, 0, len(invs))
	for _, e := range invs {
		entries = append(entries, e.Path+"\x00"+e.Hash)
	}
	sort.Strings(entries)
	var buf strings.Builder
	buf.WriteString("worktree-inventory-v2\n")
	for _, e := range entries {
		buf.WriteString(e)
		buf.WriteByte(0)
	}
	sum := sha256.Sum256([]byte(buf.String()))
	return hex.EncodeToString(sum[:])
}
