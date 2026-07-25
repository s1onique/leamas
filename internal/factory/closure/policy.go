package closure

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

var fullDigestMarker = []byte("LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION:")

// evaluateTrackedDigestPolicy enforces the historical-ACT-scope
// tracked-digest policy on the plan.baseline..S range. See
// ProvenanceTopology (run_v2_authority.go) and the explicit policy
// range decision in run_v2_policy.go for why this range — not F..S
// — is the right one for "no new full digests in this ACT".
func evaluateTrackedDigestPolicy(ctx context.Context, git gitClient, root, planBaselineCommit, subjectCommit string) (ClosurePolicyResult, []byte) {
	result := ClosurePolicyResult{TrackedFullDigestStatus: CheckStatusPass}
	rangeSpec := planBaselineCommit + ".." + subjectCommit
	changed := git.Run(ctx, root, "diff", "--name-only", "-z", "--diff-filter=AM", rangeSpec, "--")
	if changed.Err != nil || changed.ExitCode != 0 {
		result.TrackedFullDigestStatus = CheckStatusFail
		result.DiagnosticCount = 1
		return result, []byte("tracked digest policy: unable to enumerate changed files\n")
	}
	var diagnostics strings.Builder
	for _, rawPath := range bytes.Split(changed.Stdout, []byte{0}) {
		if len(rawPath) == 0 {
			continue
		}
		path := string(rawPath)
		if strings.ContainsAny(path, "\r\n") {
			result.TrackedFullDigestStatus = CheckStatusFail
			result.DiagnosticCount++
			diagnostics.WriteString("tracked digest policy: changed path has unsupported control characters\n")
			continue
		}
		blob := git.Run(ctx, root, "cat-file", "blob", subjectCommit+":"+path)
		if bytes.HasPrefix(blob.Stdout, fullDigestMarker) {
			result.TrackedFullDigestStatus = CheckStatusFail
			result.DiagnosticCount++
			fmt.Fprintf(&diagnostics, "tracked digest policy: full digest prohibited at %s\n", path)
			continue
		}
		if blob.Err != nil || blob.ExitCode != 0 {
			result.TrackedFullDigestStatus = CheckStatusFail
			result.DiagnosticCount++
			fmt.Fprintf(&diagnostics, "tracked digest policy: unable to inspect %s\n", path)
		}
	}
	return result, []byte(diagnostics.String())
}

// evaluatePatchHygiene enforces the subject-only patch-hygiene policy
// on the F..S range. The parameter is named freezeCommit (NOT
// baselineCommit) precisely because F is the subject-only origin:
// callers MUST pass the freeze commit F, never plan.baseline B.
// See ProvenanceTopology (run_v2_authority.go) and the explicit
// policy range decision in run_v2_policy.go for the rationale.
func evaluatePatchHygiene(ctx context.Context, git gitClient, root, freezeCommit, subjectCommit string) (PatchHygiene, []byte) {
	rangeSpec := freezeCommit + ".." + subjectCommit
	result := git.Run(ctx, root, "diff", "--check", rangeSpec, "--")
	diagnostics := append(append([]byte(nil), result.Stdout...), result.Stderr...)
	status := CheckStatusPass
	if result.Err != nil || result.ExitCode != 0 || len(bytes.TrimSpace(diagnostics)) != 0 {
		status = CheckStatusFail
	}
	return PatchHygiene{Status: status, DiagnosticCount: countDiagnosticLines(diagnostics)}, diagnostics
}

func countDiagnosticLines(data []byte) int {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return 0
	}
	return bytes.Count(trimmed, []byte{'\n'}) + 1
}
