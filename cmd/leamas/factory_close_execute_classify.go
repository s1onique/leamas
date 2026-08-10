// SPDX-License-Identifier: Apache-2.0

// factory_close_execute_classify.go owns the CLI error
// classification. The exit-taxonomy is:
//
//	0 PASS
//	2 request / topology / policy rejection
//	3 authoritative verification failure
//	4 observer / execution / publication unavailable
//
// The mapping is driven by the typed V2 diagnostic code on
// the inner runner error; non-typed errors collapse to 4.

package main

import (
	"github.com/s1onique/leamas/internal/factory/closure"
)

// executeResult is the internal outcome shape used before
// rendering JSON or text.
type executeResult struct {
	ok       bool
	exitCode int
	errCode  string
	manifest *closure.V2Manifest
	diag     closure.V2Diagnostics
}

// classifyExecuteError maps a runner error to the typed exit
// code and machine-readable error_code required by the
// published contract.
func classifyExecuteError(err error) executeResult {
	if err == nil {
		return executeResult{ok: true}
	}
	v2err, ok := err.(*closure.V2Error)
	if !ok {
		return executeResult{
			exitCode: 4,
			errCode:  "execution_unavailable",
			diag: closure.V2Diagnostics{{
				Code:         closure.V2CodeExecutionFailed,
				Message:      err.Error(),
				PropertyName: "v2_runner",
			}},
		}
	}
	exit := 3
	errCode := "v2_failure"
	if len(v2err.Diags) > 0 {
		code := v2err.Diags[0].Code
		switch {
		case isRequestRejection(code):
			exit = 2
			errCode = string(code)
		case isObserverUnavailable(code):
			exit = 4
			errCode = string(code)
		default:
			exit = 3
			errCode = string(code)
		}
	}
	return executeResult{
		exitCode: exit,
		errCode:  errCode,
		diag:     v2err.Diags,
	}
}

// isRequestRejection returns true when the diagnostic code
// belongs to the request / topology / policy class.
func isRequestRejection(code closure.V2DiagnosticCode) bool {
	switch code {
	case closure.V2CodeUnsupportedClosureProtocolVersion,
		closure.V2CodeUnsupportedPlanContractVersion,
		closure.V2CodeUnsupportedPlanProtocolComb,
		closure.V2CodeSubjectCommitNotFound,
		closure.V2CodeFreezeCommitNotFound,
		closure.V2CodeSubjectEqualsFreeze,
		closure.V2CodeSubjectNotAncestorOfFreeze,
		closure.V2CodeFreezeAncestorOfSubject,
		closure.V2CodeSubjectFreezeUnrelated,
		closure.V2CodeFrozenPlanPathMissing,
		closure.V2CodeFrozenPlanNotBlob,
		closure.V2CodeFrozenPlanInvalid,
		closure.V2CodeInvalidPlanPath,
		closure.V2CodeWorkingPlanMismatch,
		closure.V2CodeEvidencePathNotDetached,
		closure.V2CodeManifestPathNotDetached,
		closure.V2CodeWorkingPlanPathInvalid,
		closure.V2CodeCallerWorktreeDirty,
		closure.V2CodeRequestIncomplete,
		closure.V2CodeBinaryIdentityInvalid,
		closure.V2CodeObjectFormatUnavailable,
		closure.V2CodeUnsupportedObjectFormat:
		return true
	}
	return false
}

// isObserverUnavailable returns true when the diagnostic
// code belongs to the observer / execution / publication
// unavailable class.
func isObserverUnavailable(code closure.V2DiagnosticCode) bool {
	switch code {
	case closure.V2CodeCallerStateUnavailable,
		closure.V2CodeWorktreeInventoryUnavailable,
		closure.V2CodeGitTimeout,
		closure.V2CodeGitCancelled,
		closure.V2CodeGitOutputOverflow,
		closure.V2CodeGitSpawnFailed,
		closure.V2CodeGitNotRepository,
		closure.V2CodeGitPermissionDenied,
		closure.V2CodeGitMalformedRevision,
		closure.V2CodeGitOperationFailed,
		closure.V2CodeExecutionFailed,
		closure.V2CodeCleanupFailed,
		closure.V2CodeManifestWriteFailed,
		closure.V2CodeExecutionTreeMismatch,
		closure.V2CodeCallerHeadChanged,
		closure.V2CodeCallerTreeChanged,
		closure.V2CodeCallerWorktreeDirtyAfter,
		closure.V2CodeWorktreeRegistrationLeaked,
		closure.V2CodeCallerRefsChanged,
		closure.V2CodeManifestIdentityInvalid,
		closure.V2CodeCheckResultMappingInvalid:
		return true
	}
	return false
}
