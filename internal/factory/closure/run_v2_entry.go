// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"time"
)

// RunV2Options contains options for the v2 closure transaction kernel.
type RunV2Options struct {
	PlanPath      string // Path to the closure plan JSON file
	Subject       string // Subject commit OID
	JSONOutput    bool   // Output JSON format
	RepoDirectory string // Repository directory (defaults to ".")
}

// verifiedTransactionVerifier verifies a verified-candidate transaction
// in place of running checks. It receives the exact expected
// transaction (the deterministic reconstruction of C, E, T) and the
// invocation's sole qualified evidence snapshot.
type verifiedTransactionVerifier func(ctx context.Context, git gitClient, repoRoot, evidenceDir string,
	expected v2ExpectedTransaction, evidence v2EvidenceSnapshot) (*TransactionResult, error)

// v2Dependencies contains the dependencies for v2 execution.
type v2Dependencies struct {
	Git                   gitClient
	Commands              commandExecutor
	Runner                runnerIdentityProvider
	RunningBinarySHA256   func() (string, error)
	VerifyExisting        verifiedTransactionVerifier
	Now                   func() time.Time
	RunChecks             v2CheckRunner
	EvaluatePatchPolicy   v2PatchPolicyEvaluator
	EvaluateClosurePolicy v2ClosurePolicyEvaluator
	FinalizeNew           v2Finalizer
	PublishEvidence       v2EvidencePublisher
}

// RunClosureV2 executes the v2 closure transaction kernel.
func RunClosureV2(ctx context.Context, options RunV2Options) (*TransactionResult, error) {
	return runClosureV2WithDependencies(ctx, options, v2Dependencies{
		Git:                   RealGit{},
		Commands:              boundedCommandExecutor{},
		Runner:                currentRunnerIdentity{},
		RunningBinarySHA256:   identifyRunningBinary,
		VerifyExisting:        verifyExistingTransactionExact,
		Now:                   time.Now,
		RunChecks:             v2ExecuteChecks,
		EvaluatePatchPolicy:   evaluateRequiredPatchHygieneV2,
		EvaluateClosurePolicy: evaluateRequiredClosurePolicyV2,
		FinalizeNew:           defaultV2FinalizeNew,
		PublishEvidence:       publishV2Evidence,
	})
}
