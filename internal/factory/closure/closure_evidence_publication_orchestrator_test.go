// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// b3sha256Hex is a tiny helper used by the test fixtures.
func b3sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// makeAuthoritativeObservation constructs a fully populated
// V2ExecutionObservation whose every field is a known-good
// value the B2 barrier accepts.
func makeAuthoritativeObservation(t *testing.T) V2ExecutionObservation {
	t.Helper()
	planBytes := []byte(`{"contract_version":1,` +
		`"act_id":"ACT-PARITY-B2R4-01",` +
		`"baseline":{"commit_oid":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","tree_oid":"ffffffffffffffffffffffffffffffffffffffff"},` +
		`"execution":{"mode":"serial_fail_fast"},` +
		`"checks":[{"id":"c1","mode":"run","argv":["go","test"],"working_directory":".","timeout_seconds":60,"environment":{"K":"V"}}],` +
		`"policy":{"require_clean_before":true,"require_clean_after":true,"forbid_tracked_full_digests":true,"require_diff_check":true}` +
		`}`)
	planSHA := b3sha256Hex(planBytes)
	subjectCommit := strings.Repeat("a", 40)
	subjectTree := strings.Repeat("b", 40)
	subjectExecutionRoot := "/tmp/leamas-subject-1234"
	statusHash := strings.Repeat("1", 64)
	refsHash := strings.Repeat("2", 64)
	worktreeHash := strings.Repeat("3", 64)
	snap := evidence.CallerStateSnapshot{
		Available:             true,
		Head:                  subjectCommit,
		Tree:                  subjectTree,
		StatusHash:            statusHash,
		RefsHash:              refsHash,
		WorktreeInventoryHash: worktreeHash,
	}
	return V2ExecutionObservation{
		Runtime: evidence.RuntimeAuthority{
			RepositoryRoot:       "/repo",
			FreezeCommit:         strings.Repeat("c", 40),
			FreezeTree:           strings.Repeat("d", 40),
			SubjectCommit:        subjectCommit,
			SubjectTree:          subjectTree,
			SubjectExecutionRoot: subjectExecutionRoot,
			ExecutionTree:        subjectTree,
			PlanPath:             "docs/closure-plans/x.json",
			PlanBlob:             "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			PlanSHA256:           planSHA,
			PlanBytes:            planBytes,
			FAncestorOfSVerified: true,
		},
		Results: []evidence.CheckResult{{CheckID: "c1", Mode: "run", Outcome: "pass", ExitCode: 0}},
		Gate: evidence.GateAuthority{
			ObservedStatus:       statusHash,
			Classification:       "PASS",
			InvocationCount:      1,
			RepositoryRoot:       "/repo",
			SubjectRoot:          subjectExecutionRoot,
			SubjectExecutionRoot: subjectExecutionRoot,
		},
		Binary: evidence.BinaryAuthority{
			BinaryPath:                "/bin/leamas",
			BinarySHA256:              strings.Repeat("a", 64),
			BinaryCommit:              subjectCommit,
			BinaryModified:            false,
			SourceCommit:              subjectCommit,
			SourceTree:                subjectTree,
			SourceClean:               true,
			SourceDetached:            true,
			OutputOutsideAllWorktrees: true,
			Executable:                true,
		},
		CallerBefore: snap,
		CallerAfter:  snap,
	}
}

// TestOrchestratorInputsDerivedFromExecution is the B3-R2
// regression proof: a caller who fabricates the B2 candidate
// bundle cannot influence the published evidence because the
// orchestrator derives the bundle from the runner's
// authoritative V2ExecutionObservation.
func TestOrchestratorInputsDerivedFromExecution(t *testing.T) {
	obs := makeAuthoritativeObservation(t)
	// Deliberately contradictory: a hand-built inputs that
	// claim the run failed. The B2 candidate builder on this
	// bundle would produce an INCOMPLETE candidate. But the
	// orchestrator's bundle is derived from `obs` (which
	// records PASS), so the published evidence records
	// PASS, not FAIL.
	fabricated := evidence.CandidateInputs{
		Runtime:      obs.Runtime,
		Results:      []evidence.CheckResult{{CheckID: "c1", Mode: "run", Outcome: "fail"}},
		Gate:         obs.Gate,
		Binary:       obs.Binary,
		CallerBefore: obs.CallerBefore,
		CallerAfter:  obs.CallerAfter,
	}
	fabCandidate := evidence.BuildClosureEvidenceCandidate(fabricated)
	if got := evidence.DeriveClosureEvidenceCompleteness(fabCandidate); got == evidence.EvidenceComplete {
		t.Fatalf("fabricated inputs must NOT cross the B2 barrier; got %s", got)
	}
	// Orchestrator-derived bundle uses the authoritative obs,
	// not the fabricated one.
	inputs := deriveInputsFromObservation(obs)
	candidate := evidence.BuildClosureEvidenceCandidate(inputs)
	if got := evidence.DeriveClosureEvidenceCompleteness(candidate); got != evidence.EvidenceComplete {
		t.Fatalf("observation-derived candidate must be COMPLETE; got %s", got)
	}
}

// TestOrchestratorPublishEvidenceSmoke is the end-to-end smoke
// for the orchestrator. It uses a fake runner that returns
// the authoritative observation; the publish path produces a
// pair-visible state because the destination parent is the
// t.TempDir (real fsync may not be exercised in this test).
func TestOrchestratorPublishEvidenceSmoke(t *testing.T) {
	obs := makeAuthoritativeObservation(t)
	wt := t.TempDir()
	outside := t.TempDir()
	dest := outside + "/evidence.json"
	o := &EvidencePublicationOrchestrator{
		Runner: func(ctx context.Context, req V2Request, id V2BinaryIdentity) (V2Manifest, V2ExecutionObservation, error) {
			return V2Manifest{}, obs, nil
		},
		RepositoryRoot:      wt,
		EvidenceDestination: dest,
		Worktrees:           []CanonicalWorktree{{Path: wt}},
	}
	_, res, err := o.PublishEvidence(context.Background(), V2Request{}, V2BinaryIdentity{})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if res.Err != nil {
		t.Fatalf("res.Err = %v", res.Err)
	}
	if res.State != EvidencePublicationPairDurable && res.State != EvidencePublicationPairVisibleDurabilityUnconfirmed {
		t.Fatalf("state = %s, want pair_durable or pair_visible_durability_unconfirmed", res.State)
	}
}
