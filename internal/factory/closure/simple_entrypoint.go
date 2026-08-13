// SPDX-License-Identifier: Apache-2.0

// simple_entrypoint.go is the thin façade for the simplified
// closure entrypoint. The façade accepts ACT identity, a
// committed subject, a lane, and (optionally) publication
// delegation. Everything else — freeze derivation, plan
// loading, exact binary build, gate capture, evidence
// publication — is owned by existing closure authorities
// inside this package.
//
// ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01 contract:
// the agent supplies ONLY ACT/S/lane/publish. Leamas owns
// every other authority.
//
// Phase 6D — corrected freeze authority:
//
//	1. SUBJECT_TREE production binding:
//	     resolveSubjectTree → git rev-parse --verify S^{tree}
//
//	2. FREEZE AUTHORITY production binding:
//	     BeginAct generates canonical Plan bytes (no self-F
//	     reference), commits them once via a TEMP INDEX so the
//	     caller's live index is never mutated, and stores F in
//	     refs/factory/freeze/<ACT-ID> for canonical discovery.
//
//	3. FROZEN_PLAN_DISCOVERY production binding:
//	     discoverFrozenPlanForAct reads F from
//	     refs/factory/freeze/<ACT-ID> — F identifies the plan,
//	     the plan does NOT identify F (no circular authority).
//
//	4. BOUNDED_PUSH production binding:
//	     boundedPush → resolve LOCAL → fetch → resolve REMOTE
//	     → merge-base --is-ancestor REMOTE LOCAL (FF proof)
//	     → ordinary push → fresh read-back. NO --force.
//
// Authority model:
//
//	  F identifies P
//	  P does NOT identify F
//
//	  refs/factory/freeze/<ACT-ID> → F
//	  git show F:<plan-rel>          → canonical P bytes
//
//	  close() reads F via the sideband ref; the working-tree
//	  plan file is informational only.

package closure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SimpleCloseRequest is the canonical agent-facing input. The
// agent supplies ONLY act identity, committed subject, lane,
// and (optionally) publication delegation.
type SimpleCloseRequest struct {
	ActID   string
	Subject string
	Lane    string
	Publish bool
}

// SimpleCloseLifecycleVersion pins the closure lifecycle
// topology the simplified entrypoint expects. 1 = v1 topology
// (F strict ancestor of S, freeze-before-subject) — the only
// topology compatible with the two-command begin/close UX.
// RunClosureV2 derives F as the sideband ref's value; in v1
// topology that ref points at the freeze commit of the canonical
// plan (F != S, F ancestor-of S).
//
// Lifecycle statement (authoritative, repeated everywhere
// touched by this façade):
//
//	SimpleCloseLifecycleVersion = 1
//	F != S
//	F is a strict ancestor of S
const SimpleCloseLifecycleVersion = 1

// SimpleCloseResult is the canonical machine-readable envelope.
// The subject tree and closure tree are populated from
// authoritative, distinct sources — never from one another.
// FreezeCommit binds the envelope to the freeze ref observed
// by the underlying closure transaction; in v1 topology F is
// a strict ancestor of S (F != S).
type SimpleCloseResult struct {
	ActID           string
	FreezeCommit    string // F observed by the closure transaction
	SubjectCommit   string
	SubjectTree     string // derived from S via rev-parse S^{tree}
	ClosureCommit   string
	ClosureTree     string // the closure commit's tree
	Verdict         string
	State           string
	RerunRequired   bool
	Published       bool
	PublicationHead string
	ReasonCode      string
}

// validateActID returns nil iff the supplied act identifier is
// safe to use as both a ref-name component
// (refs/factory/freeze/<ACT>) AND a filesystem path component
// (docs/closure-plans/<ACT>.json). The grammar is the existing
// canonical closure-package ACT-ID pattern (see plan_patterns.go).
func validateActID(actID string) error {
	if !actIDPattern.MatchString(actID) {
		return fmt.Errorf("act_id_invalid: %q does not match canonical ACT-ID grammar", actID)
	}
	if strings.Contains(actID, "..") {
		return fmt.Errorf("act_id_invalid: %q contains traversal sequence", actID)
	}
	return nil
}

// FrozenPlan is the in-memory derived result of BeginAct
// (which creates F) and discoverFrozenPlanForAct (which
// reads F). The plan schema itself does NOT carry F; the
// F-to-plan binding lives in refs/factory/freeze/<ACT-ID>.
type FrozenPlan struct {
	FreezeCommit string
	PlanPath     string
}

// SimplePublicationResult is the bounded publisher's typed return.
type SimplePublicationResult struct {
	ReasonCode      string
	PublicationHead string
}

// SimpleCloseDeps captures the seams the façade exposes for
// tests and production wiring. The defaults match production.
type SimpleCloseDeps struct {
	FrozenPlanLoader    func(ctx context.Context, git gitClient, repoRoot, actID string) (FrozenPlan, error)
	SubjectTreeResolver func(ctx context.Context, git gitClient, repoRoot, subject string) (string, error)
	TransactionRunner   func(ctx context.Context, req RunV2Options) (*TransactionResult, error)
	Publisher           func(ctx context.Context, git gitClient, repoRoot, remote, localRef string) (SimplePublicationResult, error)
	Git                 gitClient
	RepositoryRoot      string
	EvidenceDir         string
	Remote              string
	Now                 func() time.Time
}

var supportedLanes = map[string]bool{"fast": true}

// freezeRefName returns the sideband ref name that holds F for
// the given ACT identifier.
func freezeRefName(actID string) string {
	return "refs/factory/freeze/" + actID
}

// frozenPlanPath returns the canonical frozen-plan path
// for the given ACT identifier. The path is repo-relative.
// Named to avoid colliding with the existing canonicalPlanPath
// helper in run_validation.go (which takes a different signature).
func frozenPlanPath(actID string) string {
	return "docs/closure-plans/" + actID + ".json"
}

// SimpleClose is the canonical façade.
func SimpleClose(ctx context.Context, req SimpleCloseRequest, deps SimpleCloseDeps) (SimpleCloseResult, error) {
	if deps.FrozenPlanLoader == nil {
		deps.FrozenPlanLoader = discoverFrozenPlanForAct
	}
	if deps.SubjectTreeResolver == nil {
		deps.SubjectTreeResolver = resolveSubjectTree
	}
	if deps.TransactionRunner == nil {
		deps.TransactionRunner = RunClosureV2
	}
	if deps.Publisher == nil {
		deps.Publisher = boundedPush
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Remote == "" {
		deps.Remote = "origin"
	}

	result := SimpleCloseResult{
		ActID:         req.ActID,
		SubjectCommit: req.Subject,
	}

	if reason, ok := validateSimpleRequest(req); !ok {
		result.State = "rerun_required"
		result.Verdict = "FAIL"
		result.RerunRequired = true
		result.ReasonCode = reason
		return result, fmt.Errorf("simplified close: %s", reason)
	}

	subjectTree, err := deps.SubjectTreeResolver(ctx, deps.Git, deps.RepositoryRoot, req.Subject)
	if err != nil {
		result.State = "rerun_required"
		result.Verdict = "FAIL"
		result.RerunRequired = true
		result.ReasonCode = "subject_tree_unavailable"
		return result, fmt.Errorf("simplified close: subject_tree_unavailable: %w", err)
	}
	result.SubjectTree = subjectTree

	frozen, err := deps.FrozenPlanLoader(ctx, deps.Git, deps.RepositoryRoot, req.ActID)
	if err != nil {
		result.State = "rerun_required"
		result.Verdict = "FAIL"
		result.RerunRequired = true
		result.ReasonCode = "act_freeze_failed"
		return result, fmt.Errorf("simplified close: act_freeze_failed: %w", err)
	}

	txResult, runErr := deps.TransactionRunner(ctx, RunV2Options{
		PlanPath:      frozen.PlanPath,
		Subject:       req.Subject,
		RepoDirectory: deps.RepositoryRoot,
	})

	if txResult != nil {
		// Phase 6F: bind the envelope to the freeze ref observed
		// by the underlying closure transaction AND verify it
		// equals the sideband F we discovered. In v1 topology
		// (F < S), RunClosureV2 derives F = S^{commit}; the
		// sideband ref must agree with that derivation, or the
		// envelope fails closed with freeze_authority_mismatch.
		if txResult.FreezeCommit != "" {
			result.FreezeCommit = txResult.FreezeCommit
		}
		if txResult.SubjectCommit != "" {
			result.SubjectCommit = txResult.SubjectCommit
		}
		if txResult.ClosureCommit != "" {
			result.ClosureCommit = txResult.ClosureCommit
		}
		if txResult.ClosureTree != "" {
			result.ClosureTree = txResult.ClosureTree
		}
		if txResult.Verdict != "" {
			result.Verdict = txResult.Verdict
		}
		// Phase 6G: freeze-authority invariant. The discovered
		// sideband F MUST be observed by the transaction.
		//   - Mismatch → freeze_authority_mismatch (agent re-authored)
		//   - Empty txResult.FreezeCommit → freeze_authority_unavailable
		//     (transaction failed to derive F; do not paper over)
		//   - Match → proceed normally
		if frozen.FreezeCommit != "" && result.FreezeCommit == "" {
			result.State = "rerun_required"
			result.Verdict = "FAIL"
			result.RerunRequired = true
			result.ReasonCode = "freeze_authority_unavailable"
			return result, fmt.Errorf("simplified close: freeze_authority_unavailable: sideband=%s but transaction observed empty F; cannot paper over missing F", frozen.FreezeCommit)
		}
		if frozen.FreezeCommit != "" && result.FreezeCommit != "" && frozen.FreezeCommit != result.FreezeCommit {
			result.State = "rerun_required"
			result.Verdict = "FAIL"
			result.RerunRequired = true
			result.ReasonCode = "freeze_authority_mismatch"
			return result, fmt.Errorf("simplified close: freeze_authority_mismatch: sideband=%s tx=%s; agent must not re-author the plan mid-flight", frozen.FreezeCommit, result.FreezeCommit)
		}
	}
	if runErr != nil {
		result.State = "rerun_required"
		result.RerunRequired = true
		if result.Verdict == "" {
			result.Verdict = "FAIL"
		}
		result.ReasonCode = "transaction_failed"
		return result, fmt.Errorf("simplified close: transaction_failed: %w", runErr)
	}
	if txResult == nil {
		result.State = "rerun_required"
		result.Verdict = "FAIL"
		result.RerunRequired = true
		result.ReasonCode = "transaction_failed"
		return result, errors.New("simplified close: transaction_failed: nil result")
	}

	if txResult.TransactionState == v2StateVerified {
		result.State = "fixed_point"
		result.RerunRequired = false
		return result, nil
	}

	if txResult.Verdict != "PASS" {
		result.State = "rerun_required"
		result.RerunRequired = true
		result.ReasonCode = "verdict_not_pass"
		return result, fmt.Errorf("simplified close: verdict_not_pass: %s", txResult.Verdict)
	}

	result.State = "fixed_point"
	result.RerunRequired = false

	if req.Publish {
		localRef := "refs/heads/main"
		pub, pubErr := deps.Publisher(ctx, deps.Git, deps.RepositoryRoot, deps.Remote, localRef)
		if pubErr != nil {
			result.Published = false
			result.State = "publication_blocked"
			result.RerunRequired = false
			result.ReasonCode = pub.ReasonCode
			if pub.ReasonCode == "" {
				result.ReasonCode = "publication_failed"
			}
			return result, fmt.Errorf("simplified close: %s: %w", result.ReasonCode, pubErr)
		}
		result.Published = true
		result.PublicationHead = pub.PublicationHead
	}

	return result, nil
}

// validateSimpleRequest enforces the input surface rules.
// Phase 6G: the ACT-ID is validated against the canonical
// ref/path grammar (validateActID). The same validator is
// shared with BeginAct so neither path can construct an
// invalid ref name or filesystem path.
func validateSimpleRequest(req SimpleCloseRequest) (string, bool) {
	if strings.TrimSpace(req.ActID) == "" {
		return "missing_act_id", false
	}
	if err := validateActID(req.ActID); err != nil {
		return "act_id_invalid", false
	}
	if strings.TrimSpace(req.Subject) == "" {
		return "missing_subject", false
	}
	if strings.TrimSpace(req.Lane) == "" {
		return "missing_lane", false
	}
	if !supportedLanes[req.Lane] {
		return "unsupported_lane", false
	}
	return "", true
}

// discoverFrozenPlanForAct reads F from the canonical sideband
// ref refs/factory/freeze/<ACT-ID>. The freeze commit
// identifies the plan; the plan schema carries no self-F
// reference (no circular authority dependency).
//
// Production flow:
//   - resolve refs/factory/freeze/<ACT-ID> via git rev-parse.
//   - if the ref is absent, return typed act_not_frozen; the
//     agent must run `leamas factory begin <ACT-ID>` first.
func discoverFrozenPlanForAct(ctx context.Context, git gitClient, repoRoot, actID string) (FrozenPlan, error) {
	if git == nil {
		return FrozenPlan{}, errors.New("git client is required")
	}
	if strings.TrimSpace(repoRoot) == "" {
		return FrozenPlan{}, errors.New("repository_root is required")
	}
	if strings.TrimSpace(actID) == "" {
		return FrozenPlan{}, errors.New("act_id is required")
	}
	refName := freezeRefName(actID)
	out, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options", refName)
	if err != nil {
		return FrozenPlan{}, fmt.Errorf("act_not_frozen: ref %s missing; agent must run `leamas factory begin %s` first to establish F: %w", refName, actID, err)
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return FrozenPlan{}, fmt.Errorf("act_not_frozen: ref %s empty", refName)
	}
	return FrozenPlan{FreezeCommit: out, PlanPath: frozenPlanPath(actID)}, nil
}

// BeginAct creates the freeze commit F and stores it in the
// canonical sideband ref. F commits the canonical Plan bytes
// once via a TEMP INDEX so the caller's live index is never
// mutated. The plan schema carries NO self-F reference
// (canonical authority model: F identifies P, P does NOT
// identify F).
//
// Production sequence (bounded; no --force semantics):
//
//  1. validate clean worktree (caller staging must be empty)
//  2. resolve current HEAD^{commit}
//  3. build canonical Plan bytes (no F-pending, no phase)
//  4. hash-object -w --stdin → blobOID
//  5. create TEMP INDEX FILE; read-tree HEAD into it
//  6. update-index --add --cacheinfo 100644,blobOID,planRel
//     into the temp index (NOT the caller's index)
//  7. write-tree (against temp index) → treeOID
//  8. commit-tree treeOID -p HEAD^{commit} -m msg → F
//  9. update-ref refs/factory/freeze/<ACT-ID> F
//  10. update-ref HEAD F (rewind; agent commits S on top)
//  11. write plan file to worktree (informational; may
//     legitimately diverge from F's plan bytes; F is
//     authoritative)
func BeginAct(ctx context.Context, deps SimpleCloseDeps, actID string) (FrozenPlan, error) {
	if deps.TransactionRunner == nil {
		deps.TransactionRunner = RunClosureV2
	}
	if deps.Publisher == nil {
		deps.Publisher = boundedPush
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Remote == "" {
		deps.Remote = "origin"
	}
	if deps.Git == nil {
		return FrozenPlan{}, errors.New("begin: git client is required")
	}
	if strings.TrimSpace(actID) == "" {
		return FrozenPlan{}, errors.New("begin: act_id is required")
	}
	if err := validateActID(actID); err != nil {
		return FrozenPlan{}, err
	}
	if strings.TrimSpace(deps.RepositoryRoot) == "" {
		return FrozenPlan{}, errors.New("begin: repository_root is required")
	}

	// 1. Require clean worktree.
	statusRes := deps.Git.Run(ctx, deps.RepositoryRoot, "status", "--porcelain=v1", "--untracked-files=all")
	if statusRes.Err != nil || statusRes.ExitCode != 0 {
		return FrozenPlan{}, fmt.Errorf("begin: status --porcelain failed: %s", simpleBoundedGitFailureDetail(statusRes))
	}
	if strings.TrimSpace(string(statusRes.Stdout)) != "" {
		return FrozenPlan{}, fmt.Errorf("begin: caller worktree is dirty; commit or stash changes before running `leamas factory begin`. dirty output:\n%s", string(statusRes.Stdout))
	}

	// 2. Resolve current HEAD^{commit}. F will be parented there.
	headCommit, err := runGitValue(ctx, deps.Git, deps.RepositoryRoot,
		"rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if err != nil {
		return FrozenPlan{}, fmt.Errorf("begin: rev-parse HEAD^{commit}: %w", err)
	}
	headCommit = strings.TrimSpace(headCommit)
	if headCommit == "" {
		return FrozenPlan{}, errors.New("begin: empty HEAD^{commit}")
	}

	// 2.5. EARLY ref-existence check (before any blob/tree/commit).
	//    If the freeze ref already exists, return idempotently
	//    WITHOUT manufacturing F2. Validate the existing F
	//    parent matches the originally observed headCommit; otherwise
	//    refuse (authority_changed).
	refName0 := freezeRefName(actID)
	curRes0 := deps.Git.Run(ctx, deps.RepositoryRoot, "rev-parse", "--verify", refName0)
	if curRes0.ExitCode == 0 {
		existingF0 := strings.TrimSpace(string(curRes0.Stdout))
		if existingF0 == "" {
			return FrozenPlan{}, fmt.Errorf("begin: act_already_frozen: %s ref is empty", refName0)
		}
		// Validate existing F is a valid freeze for this ACT:
		// it must be parented at the originally observed headCommit
		// (F < S ancestor contract).
		existingParent, _ := runGitValue(ctx, deps.Git, deps.RepositoryRoot, "rev-parse", "--verify", "--end-of-options", existingF0+"^")
		if headCommit == existingF0 || existingParent == headCommit {
			// idempotent: ref already established at the right F;
			// no second commit-tree, no second update-ref.
			return FrozenPlan{FreezeCommit: existingF0, PlanPath: frozenPlanPath(actID)}, nil
		}
		return FrozenPlan{}, fmt.Errorf("begin: act_already_frozen: %s = %s, but its parent %s != observed headCommit %s; authority_changed", refName0, existingF0, existingParent, headCommit)
	}
	// ref absent — proceed to construct F.

	// 3. Build canonical Plan bytes (no F-pending, no phase,
	//    no self-F reference). Uses the existing canonical
	//    Plan struct from this package; baseline.commit_oid
	//    records HEAD^{commit} so the closure machinery can
	//    later derive S^{tree} from F.
	serialMode := ExecutionModeSerialFailFast
	plan := Plan{
		ContractVersion: 1,
		ActID:           actID,
		Baseline:        Baseline{CommitOID: headCommit},
		Execution:       PlanExecution{Mode: &serialMode},
		Checks:          []PlanCheck{},
		Artifacts:       []PlanArtifact{},
		Policy:          PlanPolicy{},
	}
	planBytes, err := json.Marshal(&plan)
	if err != nil {
		return FrozenPlan{}, fmt.Errorf("begin: marshal canonical plan: %w", err)
	}

	// 4. Blob the plan bytes via git hash-object -w --stdin.
	blobRes := deps.Git.RunWithStdin(ctx, deps.RepositoryRoot,
		string(planBytes), "hash-object", "-w", "--stdin")
	if blobRes.Err != nil || blobRes.ExitCode != 0 {
		return FrozenPlan{}, fmt.Errorf("begin: hash-object -w --stdin: %s", simpleBoundedGitFailureDetail(blobRes))
	}
	blobOID := strings.TrimSpace(string(blobRes.Stdout))
	if blobOID == "" {
		return FrozenPlan{}, errors.New("begin: empty blob OID")
	}

	// 5/6/7. Use a TEMP INDEX so the caller's live index is
	// never mutated. read-tree populates the temp index from
	// HEAD; update-index --add stages the plan blob; write-tree
	// produces the tree against the temp index only.
	planRel := frozenPlanPath(actID)
	tmpIndex, err := os.CreateTemp("", "begin-idx-*.idx")
	if err != nil {
		return FrozenPlan{}, fmt.Errorf("begin: create temp index: %w", err)
	}
	tmpIndexPath := tmpIndex.Name()
	tmpIndex.Close()
	defer os.Remove(tmpIndexPath)
	env := []string{"GIT_INDEX_FILE=" + tmpIndexPath}

	rtRes := deps.Git.RunWithEnv(ctx, deps.RepositoryRoot, env,
		"read-tree", "HEAD")
	if rtRes.Err != nil || rtRes.ExitCode != 0 {
		return FrozenPlan{}, fmt.Errorf("begin: read-tree HEAD: %s", simpleBoundedGitFailureDetail(rtRes))
	}
	uiRes := deps.Git.RunWithEnv(ctx, deps.RepositoryRoot, env,
		"update-index", "--add", "--cacheinfo", "100644,"+blobOID+","+planRel)
	if uiRes.Err != nil || uiRes.ExitCode != 0 {
		return FrozenPlan{}, fmt.Errorf("begin: update-index --add --cacheinfo: %s", simpleBoundedGitFailureDetail(uiRes))
	}
	wtRes := deps.Git.RunWithEnv(ctx, deps.RepositoryRoot, env, "write-tree")
	if wtRes.Err != nil || wtRes.ExitCode != 0 {
		return FrozenPlan{}, fmt.Errorf("begin: write-tree: %s", simpleBoundedGitFailureDetail(wtRes))
	}
	treeOID := strings.TrimSpace(string(wtRes.Stdout))
	if treeOID == "" {
		return FrozenPlan{}, errors.New("begin: empty tree OID")
	}

	// 8. commit-tree → F (parented at the caller's HEAD).
	commitMsg := "factory: freeze ACT " + actID
	commitRes := deps.Git.Run(ctx, deps.RepositoryRoot,
		"commit-tree", treeOID, "-p", headCommit, "-m", commitMsg)
	if commitRes.Err != nil || commitRes.ExitCode != 0 {
		return FrozenPlan{}, fmt.Errorf("begin: commit-tree: %s", simpleBoundedGitFailureDetail(commitRes))
	}
	fOID := strings.TrimSpace(string(commitRes.Stdout))
	if fOID == "" {
		return FrozenPlan{}, errors.New("begin: empty freeze OID")
	}

	// 10. Atomically publish the freeze ref + advance HEAD.
	//    This is the single authority transition: the sideband
	//    freeze ref MUST be CAS-created (old=all-zeros → ensure
	//    absent) and HEAD MUST be CAS-moved (old=headCommit →
	//    ensure another writer did not change it). Both happen
	//    in one ref transaction (`git update-ref --stdin`) so
	//    either both become authoritative or neither does.
	// Production invariant:
	//
	//	worktree plan path == F's plan path == frozenPlanPath(ACT)
	//	=== docs/closure-plans/<ACT>.json
	//
	// The planRel string already begins with "docs/closure-plans/",
	// so a single repository-relative join is the canonical answer.
	// A previous shape duplicated the directory component (joined
	// planRel onto a pre-built docs/closure-plans path) and produced
	// docs/closure-plans/docs/closure-plans/<ACT>.json — that bug is
	// fixed by joining once.
	planAbs := filepath.Join(deps.RepositoryRoot, planRel)
	if err := os.MkdirAll(filepath.Dir(planAbs), 0o755); err != nil {
		return FrozenPlan{}, fmt.Errorf("begin: mkdir plan dir: %w", err)
	}

	// 11. Atomic publish: BOTH the freeze ref and HEAD are
	//     moved together in ONE ref transaction
	//     (`git update-ref --stdin`). Git acquires the locks
	//     for all queued refs atomically; either both become
	//     authoritative or neither does. This is the canonical
	//     Factory authority transition.
	txn := "start\n" +
		"update " + refName0 + " " + fOID + " " + strings.Repeat("0", 40) + "\n" +
		"update HEAD " + fOID + " " + headCommit + "\n" +
		"prepare\n" +
		"commit\n"
	txnRes := deps.Git.RunWithStdin(ctx, deps.RepositoryRoot, txn, "update-ref", "--stdin")
	if txnRes.Err != nil || txnRes.ExitCode != 0 {
		// Transaction failed → no authority established.
		return FrozenPlan{}, fmt.Errorf("begin: ref transaction failed (race on freeze-ref or HEAD): %s", simpleBoundedGitFailureDetail(txnRes))
	}

	// 12. Post-commit sync (best-effort; F is already authoritative).
	//     If any of these fail, return FrozenPlan{FreezeCommit: fOID}
	//     + a typed error so the caller knows F exists and the
	//     transition is NOT retryable as a fresh begin.
	rtLive := deps.Git.Run(ctx, deps.RepositoryRoot, "read-tree", "HEAD")
	if rtLive.Err != nil || rtLive.ExitCode != 0 {
		return FrozenPlan{FreezeCommit: fOID, PlanPath: planRel},
			fmt.Errorf("begin: post_commit_sync_failed: read-tree HEAD live index sync: %s", simpleBoundedGitFailureDetail(rtLive))
	}

	// 13. Materialize the exact canonical plan bytes in the
	//     worktree at the SINGLE canonical repository-relative
	//     path. The worktree copy is informational; F is
	//     authoritative. If this fails, F still exists and the
	//     transition is NOT retryable.
	if err := os.WriteFile(planAbs, planBytes, 0o644); err != nil {
		return FrozenPlan{FreezeCommit: fOID, PlanPath: planRel},
			fmt.Errorf("begin: post_commit_sync_failed: write plan file: %w", err)
	}

	return FrozenPlan{FreezeCommit: fOID, PlanPath: planRel}, nil
}

// resolveSubjectTree resolves S^{tree} from the canonical
// committed subject via git rev-parse.
func resolveSubjectTree(ctx context.Context, git gitClient, repoRoot, subject string) (string, error) {
	if git == nil {
		return "", errors.New("git client is required")
	}
	if strings.TrimSpace(repoRoot) == "" {
		return "", errors.New("repository_root is required")
	}
	if strings.TrimSpace(subject) == "" {
		return "", errors.New("subject is required")
	}
	out, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options", subject+"^{tree}")
	if err != nil {
		return "", fmt.Errorf("subject_tree_unavailable: rev-parse %s^{tree}: %w", subject, err)
	}
	if strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("subject_tree_unavailable: empty rev-parse output for %s^{tree}", subject)
	}
	return out, nil
}

// boundedPush is the production FF-only push primitive.
// REMOTE_OID ancestor-of LOCAL_OID via merge-base --is-ancestor.
func boundedPush(ctx context.Context, git gitClient, repoRoot, remote, localRef string) (SimplePublicationResult, error) {
	if git == nil {
		return SimplePublicationResult{ReasonCode: "publication_invalid_args"}, errors.New("boundedPush: git client is required")
	}
	if strings.TrimSpace(repoRoot) == "" {
		return SimplePublicationResult{ReasonCode: "publication_invalid_args"}, errors.New("boundedPush: repository_root is required")
	}
	if strings.TrimSpace(remote) == "" {
		return SimplePublicationResult{ReasonCode: "publication_invalid_args"}, errors.New("boundedPush: remote is required")
	}
	if strings.TrimSpace(localRef) == "" {
		return SimplePublicationResult{ReasonCode: "publication_invalid_args"}, errors.New("boundedPush: localRef is required")
	}

	branch := strings.TrimPrefix(localRef, "refs/heads/")

	localOID, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if err != nil {
		return SimplePublicationResult{ReasonCode: "fresh_authority_unavailable"}, fmt.Errorf("boundedPush: rev-parse HEAD^{commit}: %w", err)
	}
	localOID = strings.TrimSpace(localOID)
	if localOID == "" {
		return SimplePublicationResult{ReasonCode: "fresh_authority_unavailable"}, errors.New("boundedPush: empty HEAD^{commit}")
	}

	fetchRes := git.Run(ctx, repoRoot, "fetch", remote)
	if fetchRes.Err != nil || fetchRes.ExitCode != 0 {
		return SimplePublicationResult{ReasonCode: "fresh_authority_unavailable"}, fmt.Errorf("boundedPush: fetch %s: %s", remote, simpleBoundedGitFailureDetail(fetchRes))
	}

	lsRes := git.Run(ctx, repoRoot, "ls-remote", "--heads", remote, branch)
	if lsRes.Err != nil || lsRes.ExitCode != 0 {
		return SimplePublicationResult{ReasonCode: "fresh_authority_unavailable"}, fmt.Errorf("boundedPush: ls-remote %s %s: %s", remote, branch, simpleBoundedGitFailureDetail(lsRes))
	}
	remoteOID := parseLsRemoteHead(string(lsRes.Stdout), branch)
	if remoteOID == "" {
		remoteOID = strings.Repeat("0", 40)
	}

	if remoteOID != strings.Repeat("0", 40) {
		anc := git.Run(ctx, repoRoot, "merge-base", "--is-ancestor", remoteOID, localOID)
		if anc.Err != nil || anc.ExitCode != 0 {
			return SimplePublicationResult{ReasonCode: "non_fast_forward"}, fmt.Errorf("boundedPush: remote %s is not an ancestor of local %s; FF required", remoteOID, localOID)
		}
	}

	pushRes := git.Run(ctx, repoRoot, "push", remote, "HEAD:"+localRef)
	if pushRes.Err != nil || pushRes.ExitCode != 0 {
		return SimplePublicationResult{ReasonCode: "publication_failed"}, fmt.Errorf("boundedPush: push %s HEAD:%s: %s", remote, localRef, simpleBoundedGitFailureDetail(pushRes))
	}

	reRes := git.Run(ctx, repoRoot, "ls-remote", "--heads", remote, branch)
	if reRes.Err != nil || reRes.ExitCode != 0 {
		return SimplePublicationResult{ReasonCode: "remote_moved"}, fmt.Errorf("boundedPush: post-push ls-remote failed: %s", simpleBoundedGitFailureDetail(reRes))
	}
	remoteAfter := parseLsRemoteHead(string(reRes.Stdout), branch)
	if remoteAfter != localOID {
		return SimplePublicationResult{ReasonCode: "publication_verification_failed"}, fmt.Errorf("boundedPush: post-push read-back %s != local %s", remoteAfter, localOID)
	}

	return SimplePublicationResult{
		ReasonCode:      "published",
		PublicationHead: localOID,
	}, nil
}

// parseLsRemoteHead extracts the OID for the requested branch.
func parseLsRemoteHead(output, branch string) string {
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[1] == "refs/heads/"+branch {
			return fields[0]
		}
	}
	return ""
}

// simpleBoundedGitFailureDetail returns a sanitized stderr
// detail for a bounded git command result.
func simpleBoundedGitFailureDetail(res gitCommandResult) string {
	detail := strings.TrimSpace(string(res.Stderr))
	if detail != "" {
		return detail
	}
	if res.Err != nil {
		return res.Err.Error()
	}
	return "git command failed"
}
