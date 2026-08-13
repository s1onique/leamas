// SPDX-License-Identifier: Apache-2.0

// freeze_history_discovery.go is the canonical committed-history
// freeze authority primitive for the simplified Factory closure
// entrypoint.
//
// Architectural contract (frozen for ACT-LEAMAS-FACTORY-FREEZE-
// REDISCOVERY-PORTABILITY-AND-REAL-DOGFOOD01):
//
//	F = committed Git commit reachable from S
//	P = docs/closure-plans/<ACT-ID>.json contained in F
//	F^ = pre-freeze baseline commit
//	P.baseline.commit_oid = F^
//	P.baseline.tree_oid   = tree(F^)
//	F is a strict ancestor of S
//
// `discoverFrozenPlanFromHistory` is the SINGLE authority source.
// It never reads `refs/factory/freeze/<ACT-ID>`; that ref is
// purely a local cache used by the façade in simple_entrypoint.go.
//
// The primitive is bounded:
//
//   - Cheap narrowing: `git rev-list <S> -- <planPath>` returns
//     only commits reachable from S that touched the canonical
//     plan path. Repository-size history is NOT walked.
//   - Structural validation: each candidate must satisfy F1..F7.
//   - Unique-authority rule: exactly one valid candidate or the
//     primitive fails closed with a typed reason code.
//
// F8 (commit-message prefilter) is NOT used as authority; it
// would only ever reduce the candidate set further, never
// increase it. The committed-history invariants are sufficient.

package closure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// HistoryDiscoveryReason enumerates the typed outcomes of
// discoverFrozenPlanFromHistory. Callers MUST branch on these
// codes rather than parsing message text.
type HistoryDiscoveryReason string

const (
	// HistoryDiscoveryDerived: a unique valid candidate F
	// was found in committed history. HistoryF is set.
	HistoryDiscoveryDerived HistoryDiscoveryReason = "history_derived"

	// HistoryDiscoveryNotFrozen: zero valid candidates.
	// The ACT has not been frozen (no commit in the ancestry
	// of S introduces the canonical plan).
	HistoryDiscoveryNotFrozen HistoryDiscoveryReason = "act_not_frozen"

	// HistoryDiscoveryAmbiguous: more than one valid
	// candidate. Ambiguous authority → fail closed.
	HistoryDiscoveryAmbiguous HistoryDiscoveryReason = "freeze_authority_ambiguous"
)

// HistoryDiscoveryOutcome is the typed result envelope.
type HistoryDiscoveryOutcome struct {
	Reason     HistoryDiscoveryReason
	HistoryF   string   // populated only on Derived
	Candidates []string // populated only on Ambiguous (>=2)
}

// DiscoverFrozenPlanFromHistory is the canonical committed-
// history freeze authority primitive.
//
// `subject` is the committed S. The function enumerates commits
// reachable from S that touched `docs/closure-plans/<ACT>.json`
// and applies the F1..F7 structural predicates:
//
//	F1  strict ancestry       — C != S, C ancestor-of S
//	F2  canonical plan exists — C:P resolves
//	F3  plan parses           — canonical Plan loader accepts
//	F4  ACT binding           — P.act_id == actID
//	F5  baseline commit       — P.baseline.commit_oid == C^
//	F6  baseline tree         — P.baseline.tree_oid == tree(C^)
//	F7  candidate introduced  — blob(C:P) != blob(C^:P) (or C^:P absent)
//
// Exactly one valid candidate ⇒ FrozenPlan{FreezeCommit: C, ...}
// Any other count ⇒ typed failure.
//
// This function never reads or writes `refs/factory/freeze/*`.
// That namespace is an optional cache owned by the façade.
func DiscoverFrozenPlanFromHistory(
	ctx context.Context,
	git gitClient,
	repoRoot, actID, subject string,
) (FrozenPlan, HistoryDiscoveryOutcome, error) {
	if git == nil {
		return FrozenPlan{}, HistoryDiscoveryOutcome{}, errors.New("history discovery: git client is required")
	}
	if strings.TrimSpace(repoRoot) == "" {
		return FrozenPlan{}, HistoryDiscoveryOutcome{}, errors.New("history discovery: repository_root is required")
	}
	if err := validateActID(actID); err != nil {
		return FrozenPlan{}, HistoryDiscoveryOutcome{}, err
	}
	if strings.TrimSpace(subject) == "" {
		return FrozenPlan{}, HistoryDiscoveryOutcome{}, errors.New("history discovery: subject is required")
	}
	if err := validateOID("subject", subject); err != nil {
		return FrozenPlan{}, HistoryDiscoveryOutcome{}, fmt.Errorf("history discovery: invalid subject: %w", err)
	}

	planPath := frozenPlanPath(actID)

	// F8 (cheap narrowing): path-filtered rev-list. Returns
	// commits reachable from S that touched the canonical plan
	// path, including S itself if S modified the plan.
	candidates, err := enumeratePlanHistoryCommits(ctx, git, repoRoot, subject, planPath)
	if err != nil {
		return FrozenPlan{}, HistoryDiscoveryOutcome{}, fmt.Errorf("history discovery: enumerate candidates: %w", err)
	}

	valid := make([]string, 0, 1)
	for _, c := range candidates {
		ok, err := validateCandidate(ctx, git, repoRoot, c, subject, planPath, actID)
		if err != nil {
			return FrozenPlan{}, HistoryDiscoveryOutcome{}, fmt.Errorf("history discovery: validate %s: %w", c, err)
		}
		if ok {
			valid = append(valid, c)
		}
	}

	switch len(valid) {
	case 0:
		return FrozenPlan{}, HistoryDiscoveryOutcome{Reason: HistoryDiscoveryNotFrozen}, nil
	case 1:
		return FrozenPlan{FreezeCommit: valid[0], PlanPath: planPath},
			HistoryDiscoveryOutcome{Reason: HistoryDiscoveryDerived, HistoryF: valid[0]}, nil
	default:
		return FrozenPlan{},
			HistoryDiscoveryOutcome{Reason: HistoryDiscoveryAmbiguous, Candidates: valid}, nil
	}
}

// enumeratePlanHistoryCommits runs:
//
//	git rev-list <subject> -- <planPath>
//
// and returns the trimmed OIDs in rev-list order (most-recent
// first). The path filter ensures we only walk commits that
// touched the canonical plan file. The result may include S
// itself if S modified the plan; F1 rejects that case.
//
// The function uses -- to terminate option parsing so paths
// beginning with '-' are unambiguous.
func enumeratePlanHistoryCommits(
	ctx context.Context,
	git gitClient,
	repoRoot, subject, planPath string,
) ([]string, error) {
	res := git.Run(ctx, repoRoot, "rev-list", subject, "--", planPath)
	if res.Err != nil || res.ExitCode != 0 {
		detail := strings.TrimSpace(string(res.Stderr))
		if detail == "" && res.Err != nil {
			detail = res.Err.Error()
		}
		return nil, fmt.Errorf("git rev-list %s -- %s failed (exit %d): %s",
			subject, planPath, res.ExitCode, sanitizeDiagnostic(detail))
	}
	out := strings.TrimSpace(string(res.Stdout))
	if out == "" {
		return nil, nil
	}
	lines := strings.Split(out, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := validateOID("candidate", line); err != nil {
			return nil, fmt.Errorf("rev-list produced invalid OID %q: %w", line, err)
		}
		result = append(result, line)
	}
	return result, nil
}

// validateCandidate applies F1..F7 to a single candidate commit
// and returns true iff all predicates pass. A failed predicate
// causes the function to return (false, nil) — predicate failures
// are NOT errors, only structural OID/read failures are.
func validateCandidate(
	ctx context.Context,
	git gitClient,
	repoRoot, candidate, subject, planPath, actID string,
) (bool, error) {
	// F1a: F != S. (path-filtered rev-list may include S if S
	// touched the plan; F must be strictly earlier.)
	if candidate == subject {
		return false, nil
	}

	// F1b: F ancestor-of S. The path filter implies reachability
	// from S, but is-ancestor is the canonical primitive and the
	// explicit test guards against any future enumeration change.
	if !historyIsAncestor(ctx, git, repoRoot, candidate, subject) {
		return false, nil
	}

	// F2: C:P must exist as a blob in C's tree.
	blobAtC, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options",
		candidate+":"+planPath)
	if err != nil {
		// Path-filtered rev-list should already guarantee the
		// file existed at this commit, but be defensive: if
		// rev-parse disagrees, reject.
		return false, nil
	}
	if err := validateOID("blob(C:plan)", blobAtC); err != nil {
		return false, nil
	}

	// Read the canonical plan bytes from F's tree via
	// git cat-file. Never trust the worktree copy.
	planBytes, err := readBlobBytesViaGit(ctx, git, repoRoot, blobAtC)
	if err != nil {
		return false, nil
	}

	// F3: structural parsing of the Plan Contract v1 wire shape.
	// We use DecodeBytes (structural) rather than DecodeAndValidateFull
	// (semantic) because BeginAct emits canonical plans with empty
	// checks (a known production shape); the F4 act_id binding and
	// F5/F6 baseline OID bindings below provide the semantic checks
	// that matter for authority. The closure runner's
	// parsePlanBytes/ValidatePlan pair is the runtime authority on
	// the close path; F3 here only proves the bytes are structurally
	// a valid Plan document.
	root, err := plancontract.DecodeBytes(planBytes)
	if err != nil {
		return false, nil
	}
	rootMap, ok := root.(map[string]any)
	if !ok {
		return false, nil
	}
	plan, err := decodeTypedPlanForDiscovery(rootMap)
	if err != nil {
		return false, nil
	}

	// F4: ACT binding.
	if plan.ActID != actID {
		return false, nil
	}

	// Resolve C^ (parent of C). For a non-merge single-parent
	// commit, C^ is the single parent. Use rev-list --parents
	// so we get an explicit list (rejects root/merge commits).
	parentOID, err := historySingleParent(ctx, git, repoRoot, candidate)
	if err != nil {
		return false, nil
	}

	// F5: baseline.commit_oid == C^.
	if plan.Baseline.CommitOID != parentOID {
		return false, nil
	}

	// F6: baseline.tree_oid == tree(C^).
	parentTree, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options",
		parentOID+"^{tree}")
	if err != nil {
		return false, nil
	}
	if plan.Baseline.TreeOID != parentTree {
		return false, nil
	}

	// F7: C introduced or modified the plan blob relative to
	// C^. Either C^:P is absent (file introduced at C) OR
	// blob(C^:P) != blob(C:P) (file modified at C). The
	// path-filtered rev-list already implies one of these holds,
	// but we verify explicitly.
	blobAtParent, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options",
		parentOID+":"+planPath)
	if err == nil {
		// Parent has the file: it must differ.
		if blobAtParent == blobAtC {
			return false, nil
		}
	} else {
		// Parent does not have the file. Distinguish a real
		// "missing" (rev-parse exits 128 / not a valid object)
		// from a structural failure. A missing path at parent
		// is the introduction case and is valid; only an
		// invalid rev-parse OID is a structural error.
		// runGitValue already returns an error for both, so we
		// treat any rev-parse failure here as "parent absent".
		// (ValidateOID would have caught a malformed parent OID
		// upstream.)
		_ = err
	}

	return true, nil
}

// historyIsAncestor returns true iff `ancestor` is an ancestor
// of `descendant` per `git merge-base --is-ancestor`.
func historyIsAncestor(ctx context.Context, git gitClient, repoRoot, ancestor, descendant string) bool {
	res := git.Run(ctx, repoRoot, "merge-base", "--is-ancestor", ancestor, descendant)
	return res.Err == nil && res.ExitCode == 0
}

// historySingleParent returns the single parent OID of `commit`,
// or an error if `commit` is a root commit or a merge commit.
// Backed by `git rev-list --parents -n 1` (the same primitive
// run_v2_helpers.verifySingleParent uses internally).
func historySingleParent(ctx context.Context, git gitClient, repoRoot, commit string) (string, error) {
	res := git.Run(ctx, repoRoot, "rev-list", "--parents", "-n", "1", commit)
	if res.Err != nil || res.ExitCode != 0 {
		detail := strings.TrimSpace(string(res.Stderr))
		if detail == "" && res.Err != nil {
			detail = res.Err.Error()
		}
		return "", fmt.Errorf("git rev-list --parents -n 1 %s failed (exit %d): %s",
			commit, res.ExitCode, sanitizeDiagnostic(detail))
	}
	out := strings.TrimSpace(string(res.Stdout))
	if out == "" {
		return "", fmt.Errorf("rev-list --parents %s returned empty", commit)
	}
	fields := strings.Fields(out)
	// Expected: [<commit> <parent>]
	if len(fields) != 2 {
		return "", fmt.Errorf("rev-list --parents %s returned %d fields; want exactly 2 (commit + single parent)",
			commit, len(fields))
	}
	if fields[0] != commit {
		return "", fmt.Errorf("rev-list --parents returned %q, expected %q", fields[0], commit)
	}
	if err := validateOID("parent OID", fields[1]); err != nil {
		return "", fmt.Errorf("rev-list --parents returned invalid parent OID: %w", err)
	}
	return fields[1], nil
}

// plancontract.DecodeBytes is the structural decoder for Plan
// Contract v1. It returns a map[string]any representation of the
// canonical wire shape without applying semantic validation.
// Re-exposed here so the freeze history primitive does not depend
// on the closure runner's typed decode (which would require pulling
// in plancontract directly).
//
// plancontract's DecodeBytes is the single structural decoder. We
// use the closure-package alias (which is the same function via the
// public alias in plancontract.go) for clarity.

// decodeTypedPlanForDiscovery is the structural-only mirror of
// closure.decodeTypedPlan. It re-uses the public plan contract
// canonical model to produce a typed Plan without enforcing the
// non-empty-checks rule. The freeze history primitive does not
// require that rule because BeginAct emits minimal plans; the
// closure runner enforces it at close time via validateV2Plan.
func decodeTypedPlanForDiscovery(root map[string]any) (Plan, error) {
	plan := Plan{}
	if v, ok := root["contract_version"].(float64); ok {
		plan.ContractVersion = int(v)
	}
	if v, ok := root["act_id"].(string); ok {
		plan.ActID = v
	}
	if baseAny, ok := root["baseline"].(map[string]any); ok {
		if v, ok := baseAny["commit_oid"].(string); ok {
			plan.Baseline.CommitOID = v
		}
		if v, ok := baseAny["tree_oid"].(string); ok {
			plan.Baseline.TreeOID = v
		}
	}
	return plan, nil
}
