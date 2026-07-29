// SPDX-License-Identifier: Apache-2.0

package verifierauthority

// executionObservation captures the provenance of an execution-context
// observation. The fields are unexported so only the authority package
// can construct a successfully observed context. External callers
// cannot manufacture a "local" classification by populating exported
// fields.
//
// The observation is considered "local" only when:
//   - the observation completed successfully;
//   - HEAD, status, repository-root, and (when relevant) workspace-root
//     Git observations all succeeded;
//   - no positive CI or GitHub Actions signal is present;
//   - the marker / SHA / workspace environment fields are either absent
//     or syntactically well-formed.
//
// A dirty worktree is allowed for baseline mutation; only failure to
// execute git status (HeadErr / StatusErr / RepositoryRootErr /
// WorkspaceRootErr) blocks the local classification.
type executionObservation struct {
	completed       bool
	local           bool
	headObserved    bool
	statusObserved  bool
	repoObserved    bool
	workspaceLookup bool
	workspaceOK     bool
}

// observationZero is the zero value for executionObservation. It
// indicates "no observation has occurred" and the environment is
// classified as EnvironmentUnknown.
var observationZero = executionObservation{}

// recordLocalObservation is the authority-package-only entry point
// that records a successful local observation. The caller must supply
// the result of each Git observation explicitly so the trust
// boundary lives at the package boundary, not at exported field
// population.
func recordLocalObservation(headOK, statusOK, repoOK, workspaceLookup, workspaceOK bool) executionObservation {
	return executionObservation{
		completed:       true,
		local:           headOK && statusOK && repoOK && (!workspaceLookup || workspaceOK),
		headObserved:    headOK,
		statusObserved:  statusOK,
		repoObserved:    repoOK,
		workspaceLookup: workspaceLookup,
		workspaceOK:     workspaceOK,
	}
}
