// SPDX-License-Identifier: Apache-2.0

package closure

// r2cr4_dogfood_plan_sha_test.go covers the raw-blob
// authority path, the trailing-newline regression, the
// object-format policy, the inner-cause preservation, and
// the exact diagnostic cardinality / order required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2C-R4.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
)

// TestR2CRDogfoodPlanSHAIncludesTrailingNewline proves the
// manifest plan SHA-256 equals the SHA-256 of the raw blob
// bytes returned by `git cat-file blob <oid>` and that the
// trailing newline is preserved end-to-end. The trimmed
// byte SHA-256 must differ so a regression that silently
// strips the trailing newline is caught.
//
// The fixture explicitly appends a single '\n' to the
// canonical Plan Contract v1 document so the F:P blob ends
// with exactly one trailing newline.
func TestR2CRDogfoodPlanSHAIncludesTrailingNewline(t *testing.T) {
	dir := initRepo(t)
	planPath := "docs/closure-plans/NEWLINE.json"
	// Subject is the parent of freeze in v2 topology.
	subject := makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	if subjectTree == "" {
		t.Fatalf("subject tree must be non-empty")
	}
	frozenBytes := buildR2CRNewlineFixture(t, subject, subjectTree)
	if frozenBytes[len(frozenBytes)-1] != '\n' {
		t.Fatalf("fixture must end with '\\n'")
	}
	if bytes.Count(frozenBytes, []byte{'\n'}) != 1 {
		t.Fatalf("fixture must contain exactly one trailing newline, got %d newlines",
			bytes.Count(frozenBytes, []byte{'\n'}))
	}
	freeze := makeCommit(t, dir, "freeze: trailing newline fixture", map[string]string{
		planPath: string(frozenBytes),
	})

	blobOID := mustRunGit(t, dir, "rev-parse", freeze+":"+planPath)
	if len(blobOID) != 40 {
		t.Fatalf("frozen blob OID must be 40 chars, got %d", len(blobOID))
	}

	rawBytes, err := runR2CRGitRaw(context.Background(), dir, blobOID)
	if err != nil {
		t.Fatalf("runR2CRGitRaw(%s): %v", blobOID, err)
	}

	if len(rawBytes) == 0 || rawBytes[len(rawBytes)-1] != '\n' {
		t.Fatalf("raw blob must end with '\\n', got last byte 0x%02x", rawBytes[len(rawBytes)-1])
	}

	if !bytes.Equal(rawBytes, frozenBytes) {
		t.Fatalf("raw blob bytes disagree with in-memory fixture: raw=%d want=%d",
			len(rawBytes), len(frozenBytes))
	}

	trimmedBytes := bytes.TrimRight(rawBytes, " \t\r\n")
	if bytes.Equal(trimmedBytes, rawBytes) {
		t.Fatalf("trimming produced identical bytes; trailing whitespace test is vacuous")
	}

	rawSHA := sha256Hex(rawBytes)
	trimmedSHA := sha256Hex(trimmedBytes)
	if rawSHA == trimmedSHA {
		t.Fatalf("raw and trimmed SHA-256 must differ (raw=%s trimmed=%s)", rawSHA, trimmedSHA)
	}

	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               planPath,
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         manifestPath,
	}
	manifest, err := runClosureProtocolV2ForTest(t, context.Background(), req)
	if err != nil {
		t.Fatalf("RunClosureProtocolV2: %v", err)
	}
	if manifest.PlanBlob != blobOID {
		t.Fatalf("manifest.plan_blob: got=%s want=%s", manifest.PlanBlob, blobOID)
	}
	if manifest.PlanSHA256 != rawSHA {
		t.Fatalf("manifest.plan_sha256: got=%s want=%s", manifest.PlanSHA256, rawSHA)
	}
	if manifest.PlanSHA256 == trimmedSHA {
		t.Fatalf("manifest.plan_sha256 must not equal trimmed SHA-256 (both %s)", manifest.PlanSHA256)
	}
	wantSum := sha256.Sum256(rawBytes)
	if manifest.PlanSHA256 != hex.EncodeToString(wantSum[:]) {
		t.Fatalf("manifest.plan_sha256 disagrees with raw-byte SHA-256")
	}
}

// buildR2CRNewlineFixture returns a contract-valid Plan
// Contract v1 document that ends with exactly one trailing
// newline so the F:P blob satisfies the trailing-newline
// regression requirement. The supplied subject / tree OIDs
// are bound into the baseline so the runner accepts the
// fixture.
func buildR2CRNewlineFixture(t *testing.T, subject, subjectTree string) []byte {
	t.Helper()
	doc := map[string]any{
		"contract_version": 1,
		"act_id":           "ACT-R2CR-NEWLINE",
		"baseline":         map[string]string{"commit_oid": subject, "tree_oid": subjectTree},
		"execution":        map[string]string{"mode": "serial_fail_fast"},
		"checks": []map[string]any{{
			"id":                "noop",
			"mode":              "run",
			"argv":              []string{"true"},
			"working_directory": ".",
			"timeout_seconds":   60,
			"environment":       map[string]string{},
		}},
		"artifacts": []any{},
		"policy": map[string]bool{
			"require_clean_before":        true,
			"require_clean_after":         true,
			"forbid_tracked_full_digests": true,
			"require_diff_check":          true,
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return append(raw, '\n')
}

// TestR2CRDogfoodPlanSHACoverLeadingAndTrailingWhitespace
// proves the byte-authority contract also covers leading
// whitespace and trailing spaces. The test constructs the
// S < F topology explicitly: subject first, freeze as the
// child. The leading/trailing whitespace must round-trip
// through the byte-authority path unchanged.
func TestR2CRDogfoodPlanSHACoverLeadingAndTrailingWhitespace(t *testing.T) {
	dir := initRepo(t)
	planPath := "docs/closure-plans/WS.json"
	subject := makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	frozenBytes := buildR2CRWhitespaceFixture(t, subject, subjectTree)
	freeze := makeCommit(t, dir, "freeze: whitespace fixture", map[string]string{
		planPath: string(frozenBytes),
	})
	blobOID := mustRunGit(t, dir, "rev-parse", freeze+":"+planPath)
	rawBytes, err := runR2CRGitRaw(context.Background(), dir, blobOID)
	if err != nil {
		t.Fatalf("runR2CRGitRaw: %v", err)
	}
	if !bytes.Equal(rawBytes, frozenBytes) {
		t.Fatalf("raw bytes disagree with fixture: %d vs %d", len(rawBytes), len(frozenBytes))
	}
	wantSHA := sha256Hex(rawBytes)
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               planPath,
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         manifestPath,
	}
	manifest, err := runClosureProtocolV2ForTest(t, context.Background(), req)
	if err != nil {
		t.Fatalf("RunClosureProtocolV2: %v", err)
	}
	if manifest.PlanSHA256 != wantSHA {
		t.Fatalf("manifest.plan_sha256: got=%s want=%s", manifest.PlanSHA256, wantSHA)
	}
	if rawBytes[0] != ' ' {
		t.Fatalf("fixture must start with space, got 0x%02x", rawBytes[0])
	}
	if rawBytes[len(rawBytes)-1] != ' ' {
		t.Fatalf("fixture must end with space, got 0x%02x", rawBytes[len(rawBytes)-1])
	}
}

// buildR2CRWhitespaceFixture returns JSON bytes that begin
// and end with a single space so the SHA-256 differs from
// the unmangled document. The supplied subject / tree OIDs
// are bound into the baseline so the runner accepts the
// fixture.
func buildR2CRWhitespaceFixture(t *testing.T, subject, subjectTree string) []byte {
	t.Helper()
	doc := map[string]any{
		"contract_version": 1,
		"act_id":           "ACT-R2CR-WS",
		"baseline":         map[string]string{"commit_oid": subject, "tree_oid": subjectTree},
		"execution":        map[string]string{"mode": "serial_fail_fast"},
		"checks": []map[string]any{{
			"id":                "noop",
			"mode":              "run",
			"argv":              []string{"true"},
			"working_directory": ".",
			"timeout_seconds":   60,
			"environment":       map[string]string{},
		}},
		"artifacts": []any{},
		"policy": map[string]bool{
			"require_clean_before":        true,
			"require_clean_after":         true,
			"forbid_tracked_full_digests": true,
			"require_diff_check":          true,
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return append(append([]byte(" "), raw...), ' ')
}

// TestR2CRNonV2InnerErrorPreservesCause_NoDrift proves the
// outer runner preserves the original error via errors.Is
// when the inner runner returns a plain (non-*V2Error)
// sentinel and there is no drift.
func TestR2CRNonV2InnerErrorPreservesCause_NoDrift(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel")
	runR2CRInnerCauseMatrix(t, sentinel, "")
}

// TestR2CRNonV2InnerErrorPreservesCause_HeadDrift covers the
// HEAD drift scenario.
func TestR2CRNonV2InnerErrorPreservesCause_HeadDrift(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel for head drift")
	runR2CRInnerCauseMatrix(t, sentinel, "head_commit")
}

// TestR2CRNonV2InnerErrorPreservesCause_TreeDrift covers the
// HEAD-tree drift scenario.
func TestR2CRNonV2InnerErrorPreservesCause_TreeDrift(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel for tree drift")
	runR2CRInnerCauseMatrix(t, sentinel, "head_tree")
}

// TestR2CRNonV2InnerErrorPreservesCause_StatusDrift covers
// the status drift scenario.
func TestR2CRNonV2InnerErrorPreservesCause_StatusDrift(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel for status drift")
	runR2CRInnerCauseMatrix(t, sentinel, "status")
}

// TestR2CRNonV2InnerErrorPreservesCause_WorktreeLeak covers
// the worktree-leak scenario.
func TestR2CRNonV2InnerErrorPreservesCause_WorktreeLeak(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel for worktree leak")
	runR2CRInnerCauseMatrix(t, sentinel, "worktree_leaked")
}

// TestR2CRNonV2InnerErrorPreservesCause_AfterUnavailable
// covers the after-snapshot unavailability scenario.
func TestR2CRNonV2InnerErrorPreservesCause_AfterUnavailable(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel for after unavailability")
	runR2CRInnerCauseAfterUnavailableMatrix(t, sentinel)
}

// runR2CRInnerCauseMatrix exercises the runner with a
// non-*V2Error sentinel and mutates the after-snapshot per
// `driftKind`. It asserts:
//   - the wrapped error is a *V2Error,
//   - errors.Is(result, sentinel) is true (Cause preserved),
//   - the diagnostic list begins with the inner fallback
//     diagnostic (V2CodeExecutionFailed),
//   - the drift diagnostic (if any) follows in canonical
//     order,
//   - no manifest bytes were published.
func runR2CRInnerCauseMatrix(t *testing.T, sentinel error, driftKind string) {
	t.Helper()
	req := v2FailClosedRunnerRequest(t)
	before := snapshotCallerStateForFixture(t, req.RepositoryRoot)
	var afterFn V2RunnerSnapshotFunc
	if driftKind == "" {
		afterFn = func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
			return before
		}
	} else {
		afterFn = driftSnapshotAfterFn(before, driftKind)
	}
	deps := nonV2CauseDeps(t, req, sentinel, afterFn)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("runner must surface a V2Error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("errors.Is must reach sentinel: v2err.Cause=%v", v2err.Cause)
	}
	if len(v2err.Diags) == 0 {
		t.Fatalf("expected at least one diagnostic, got none")
	}
	if v2err.Diags[0].Code != V2CodeExecutionFailed {
		t.Fatalf("first diagnostic must be execution_failed, got %s", v2err.Diags[0].Code)
	}
	if !containsDiagsMessage(v2err.Diags[0].Message, sentinel.Error()) {
		t.Fatalf("inner diagnostic message must embed sentinel text: %q", v2err.Diags[0].Message)
	}
	if driftKind != "" {
		driftCode := r2cr4DriftCode(driftKind)
		idx := -1
		for i, d := range v2err.Diags {
			if d.Code == driftCode {
				idx = i
				break
			}
		}
		if idx < 1 {
			t.Fatalf("drift code %s must follow inner diagnostic, idx=%d diags=%+v",
				driftCode, idx, v2err.Diags)
		}
	}
	if _, statErr := osStatImpl(req.ManifestOutput); statErr == nil {
		t.Fatalf("manifest must not be published on inner failure")
	} else if !osIsNotExistImpl(statErr) {
		t.Fatalf("unexpected manifest stat error: %v", statErr)
	}
}

// runR2CRInnerCauseAfterUnavailableMatrix exercises the
// runner with a sentinel error and an unavailable
// after-snapshot. The inner diagnostic must remain FIRST.
func runR2CRInnerCauseAfterUnavailableMatrix(t *testing.T, sentinel error) {
	t.Helper()
	req := v2FailClosedRunnerRequest(t)
	before := snapshotCallerStateForFixture(t, req.RepositoryRoot)
	afterFn := func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
		if phase == V2SnapshotPhaseBefore {
			return before
		}
		return v2CallerStateSnapshot{
			Available: false,
			Diagnostics: V2Diagnostics{{
				Code:         V2CodeCallerStateUnavailable,
				Message:      "synthetic after-snapshot unavailable",
				PropertyName: "caller_state",
			}},
		}
	}
	deps := nonV2CauseDeps(t, req, sentinel, afterFn)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("runner must surface a V2Error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("errors.Is must reach sentinel: v2err.Cause=%v", v2err.Cause)
	}
	if len(v2err.Diags) == 0 {
		t.Fatalf("expected at least one diagnostic")
	}
	if v2err.Diags[0].Code != V2CodeExecutionFailed {
		t.Fatalf("first diagnostic must be execution_failed, got %s", v2err.Diags[0].Code)
	}
	hasUnavailable := false
	for _, d := range v2err.Diags {
		if d.Code == V2CodeCallerStateUnavailable {
			hasUnavailable = true
		}
	}
	if !hasUnavailable {
		t.Fatalf("after-availability diagnostic missing: %+v", v2err.Diags)
	}
}

// nonV2CauseDeps returns a V2RunnerDeps configured to fail
// the inner execution with the supplied plain sentinel error
// and to use the supplied snapshot function.
func nonV2CauseDeps(t *testing.T, req V2Request, sentinel error, afterFn V2RunnerSnapshotFunc) V2RunnerDeps {
	t.Helper()
	deps := DefaultV2RunnerDeps()
	deps.BinaryIdentity = newV2TestBinaryIdentity(t)
	deps.SnapshotFn = afterFn
	deps.Executor = &r2cr4SentinelExecutor{sentinel: sentinel}
	return deps
}

// r2cr4SentinelExecutor returns the supplied plain error so
// the outer runner must exercise the inner-cause preservation
// path.
type r2cr4SentinelExecutor struct {
	sentinel error
}

func (e *r2cr4SentinelExecutor) ExecuteSubjectChecks(ctx context.Context, req V2ExecuteRequest) (V2ExecuteResult, error) {
	return V2ExecuteResult{}, e.sentinel
}

// r2cr4DriftCode maps a drift-kind label to the matching
// V2DiagnosticCode.
func r2cr4DriftCode(kind string) V2DiagnosticCode {
	switch kind {
	case "head_commit":
		return V2CodeCallerHeadChanged
	case "head_tree":
		return V2CodeCallerTreeChanged
	case "status":
		return V2CodeCallerWorktreeDirtyAfter
	case "worktree_leaked":
		return V2CodeWorktreeRegistrationLeaked
	default:
		return ""
	}
}

// containsDiagsMessage reports whether needle appears in hay.
func containsDiagsMessage(hay, needle string) bool {
	if needle == "" {
		return true
	}
	if hay == "" {
		return false
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestR2CRMultiDriftPinsCanonicalOrder proves that a
// multi-drift fixture produces diagnostics in the canonical
// Diff order: head_commit, head_tree, status, worktree.
func TestR2CRMultiDriftPinsCanonicalOrder(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	before := snapshotCallerStateForFixture(t, req.RepositoryRoot)
	afterFn := func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
		if phase == V2SnapshotPhaseBefore {
			return before
		}
		after := before
		mutateR2CR2SnapshotState(&after.State, "head_commit")
		mutateR2CR2SnapshotState(&after.State, "head_tree")
		mutateR2CR2SnapshotState(&after.State, "status")
		mutateR2CR2SnapshotState(&after.State, "worktree_leaked")
		return after
	}
	deps := r2cr2InnerFailureDeps(t, req,
		V2CodeExecutionFailed, "multi-drift inner failure",
		afterFn, &countingCandidateObserver{})
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("runner must surface a V2Error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T", err)
	}
	if len(v2err.Diags) != 5 {
		t.Fatalf("expected exactly 5 diagnostics (inner + 4 drifts), got %d: %+v",
			len(v2err.Diags), v2err.Diags)
	}
	wantOrder := []V2DiagnosticCode{
		V2CodeExecutionFailed,
		V2CodeCallerHeadChanged,
		V2CodeCallerTreeChanged,
		V2CodeCallerWorktreeDirtyAfter,
		V2CodeWorktreeRegistrationLeaked,
	}
	for i, want := range wantOrder {
		if v2err.Diags[i].Code != want {
			t.Fatalf("diag[%d].Code: got=%s want=%s", i, v2err.Diags[i].Code, want)
		}
	}
}

// TestR2CRTypedInnerErrorExactCardinality proves a typed
// *V2Error inner failure surfaces exactly two diagnostics.
func TestR2CRTypedInnerErrorExactCardinality(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	before := snapshotCallerStateForFixture(t, req.RepositoryRoot)
	afterFn := driftSnapshotAfterFn(before, "head_commit")
	deps := r2cr2InnerFailureDeps(t, req,
		V2CodeExecutionFailed, "typed inner failure for exact cardinality",
		afterFn, &countingCandidateObserver{})
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("runner must surface a V2Error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T", err)
	}
	if len(v2err.Diags) != 2 {
		t.Fatalf("expected exactly 2 diagnostics, got %d: %+v",
			len(v2err.Diags), v2err.Diags)
	}
	if v2err.Diags[0].Code != V2CodeExecutionFailed {
		t.Fatalf("diag[0].Code: got=%s want=%s",
			v2err.Diags[0].Code, V2CodeExecutionFailed)
	}
	if !containsDiagsMessage(v2err.Diags[0].Message, "typed inner failure for exact cardinality") {
		t.Fatalf("diag[0].Message must embed inner message: %q", v2err.Diags[0].Message)
	}
	if v2err.Diags[1].Code != V2CodeCallerHeadChanged {
		t.Fatalf("diag[1].Code: got=%s want=%s",
			v2err.Diags[1].Code, V2CodeCallerHeadChanged)
	}
}

// TestR2CRObjectFormatEnforcedBeforeOIDValidation proves
// the verifier rejects a sha256 resolver with a typed
// unsupported_object_format diagnostic before any OID
// validation runs.
func TestR2CRObjectFormatEnforcedBeforeOIDValidation(t *testing.T) {
	resolver := &r2cr4StubResolver{formatResult: "sha256"}
	err := EnforceSHA1ObjectFormat(resolver)
	if err == nil {
		t.Fatalf("expected unsupported_object_format error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeUnsupportedObjectFormat) {
		t.Fatalf("expected unsupported_object_format, got %v", v2err.Diags.Codes())
	}
}

// TestR2CRObjectFormatEmptyRejected proves an empty format
// string is rejected with object_format_unavailable.
func TestR2CRObjectFormatEmptyRejected(t *testing.T) {
	resolver := &r2cr4StubResolver{formatResult: ""}
	err := EnforceSHA1ObjectFormat(resolver)
	if err == nil {
		t.Fatalf("expected object_format_unavailable error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T", err)
	}
	if !v2err.Diags.HasCode(V2CodeObjectFormatUnavailable) {
		t.Fatalf("expected object_format_unavailable, got %v", v2err.Diags.Codes())
	}
}

// TestR2CRObjectFormatResolverErrorRejected proves a
// resolver error surfaces as object_format_unavailable.
func TestR2CRObjectFormatResolverErrorRejected(t *testing.T) {
	resolver := &r2cr4StubResolver{formatErr: errors.New("synthetic observation failure")}
	err := EnforceSHA1ObjectFormat(resolver)
	if err == nil {
		t.Fatalf("expected object_format_unavailable error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T", err)
	}
	if !v2err.Diags.HasCode(V2CodeObjectFormatUnavailable) {
		t.Fatalf("expected object_format_unavailable, got %v", v2err.Diags.Codes())
	}
}

// TestR2CRObjectFormatSha1Accepted proves a sha1 resolver
// passes EnforceSHA1ObjectFormat without diagnostics.
func TestR2CRObjectFormatSha1Accepted(t *testing.T) {
	resolver := &r2cr4StubResolver{formatResult: "sha1"}
	if err := EnforceSHA1ObjectFormat(resolver); err != nil {
		t.Fatalf("sha1 must be accepted, got %v", err)
	}
}

// TestR2CRObjectFormatOIDLengthNotUsedAsDetector proves the
// resolver's reported format overrides OID length detection.
func TestR2CRObjectFormatOIDLengthNotUsedAsDetector(t *testing.T) {
	resolver := &r2cr4StubResolver{
		formatResult: "sha1",
		cat:          func(oid string) ([]byte, error) { return []byte("tree " + oid + "\n"), nil },
	}
	if err := EnforceSHA1ObjectFormat(resolver); err != nil {
		t.Fatalf("sha1 must be accepted regardless of OID length, got %v", err)
	}
}

// r2cr4StubResolver is a GitObjectResolver whose ObjectFormat
// and CatFile outcomes are configured by the test.
type r2cr4StubResolver struct {
	cat          func(oid string) ([]byte, error)
	formatResult string
	formatErr    error
}

func (r *r2cr4StubResolver) CatFile(oid string) ([]byte, error) {
	if r.cat != nil {
		return r.cat(oid)
	}
	return nil, errors.New("cat not configured")
}

func (r *r2cr4StubResolver) ObjectFormat() (string, error) {
	return r.formatResult, r.formatErr
}
