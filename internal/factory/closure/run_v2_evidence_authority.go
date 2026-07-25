// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// validateV2EvidenceAuthority proves that a qualified recovery snapshot came
// from the current subject-exact runner, was sealed on the currently attached
// branch, and contains exactly the frozen runnable check matrix with passing
// outcomes. Qualification alone proves file integrity; this function proves
// those files are authorized by F2 and bound to the same publication target
// the original invocation approved.
func validateV2EvidenceAuthority(plan Plan, subjectTree string, current RunnerIdentity,
	currentBranch string, evidence v2EvidenceSnapshot) error {
	if !evidence.Present {
		return fmt.Errorf("qualified evidence snapshot is absent")
	}
	runtime := evidence.Runtime
	if !reflect.DeepEqual(runtime.Runner, current) {
		return fmt.Errorf("runtime runner identity does not match current subject-exact runner")
	}
	if runtime.PublicationBranch == "" {
		return fmt.Errorf("runtime publication branch is empty")
	}
	if currentBranch == "" {
		return fmt.Errorf("current attached branch is empty")
	}
	if runtime.PublicationBranch != currentBranch {
		return fmt.Errorf("runtime publication branch %q does not match current attached branch %q",
			runtime.PublicationBranch, currentBranch)
	}

	runnable := make([]PlanCheck, 0, len(plan.Checks))
	for _, check := range plan.Checks {
		if check.Mode == CheckModeRun {
			runnable = append(runnable, check)
		}
	}
	if len(runtime.Checks) != len(runnable) {
		return fmt.Errorf("runtime check count %d does not match frozen runnable count %d", len(runtime.Checks), len(runnable))
	}
	expectedEvidence := make([]EvidenceRecord, 0, len(runtime.Checks)*2)
	for i, check := range runnable {
		result := runtime.Checks[i]
		if result.CheckID != check.ID {
			return fmt.Errorf("runtime check order mismatch at index %d", i)
		}
		if err := verifyRunResult(check, result, subjectTree); err != nil {
			return err
		}
		if !canonicalCheckPassed(stableV2Check(result)) {
			return fmt.Errorf("runtime check %q is not a complete pass", check.ID)
		}
		expectedEvidence = append(expectedEvidence,
			EvidenceRecord{LogicalName: check.ID + ".stdout", MediaType: "text/plain; charset=utf-8",
				SHA256: result.StdoutSHA256, ByteCount: result.StdoutByteCount, Availability: "detached"},
			EvidenceRecord{LogicalName: check.ID + ".stderr", MediaType: "text/plain; charset=utf-8",
				SHA256: result.StderrSHA256, ByteCount: result.StderrByteCount, Availability: "detached"},
		)
	}
	if !reflect.DeepEqual(runtime.CheckEvidence, expectedEvidence) {
		return fmt.Errorf("runtime check evidence does not exactly bind frozen check outputs")
	}
	if runtime.PatchHygiene.Status != CheckStatusPass ||
		runtime.ClosurePolicy.TrackedFullDigestStatus != CheckStatusPass {
		return fmt.Errorf("runtime policy evidence is not passing")
	}

	entries, err := decodeV2EvidenceEntries(evidence.IndexBytes)
	if err != nil {
		return err
	}
	for _, record := range expectedEvidence {
		entry, ok := entries[record.LogicalName]
		if !ok || entry.MediaType != record.MediaType || entry.ByteSize != record.ByteCount || entry.SHA256 != record.SHA256 {
			return fmt.Errorf("indexed evidence for %q does not match runtime record", record.LogicalName)
		}
	}
	diagnostics := []struct{ name, hash string }{
		{name: "patch-policy.log", hash: runtime.PatchDiagnosticsSHA256},
		{name: "closure-policy.log", hash: runtime.ClosureDiagnosticsSHA256},
	}
	for _, diagnostic := range diagnostics {
		entry, ok := entries[diagnostic.name]
		if !ok || entry.SHA256 != diagnostic.hash {
			return fmt.Errorf("indexed diagnostic %q does not match runtime record", diagnostic.name)
		}
	}
	return nil
}

func decodeV2EvidenceEntries(indexBytes []byte) (map[string]v2EvidenceIndexEntry, error) {
	var index v2EvidenceIndex
	decoder := json.NewDecoder(strings.NewReader(string(indexBytes)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&index); err != nil {
		return nil, fmt.Errorf("decode evidence index authority snapshot: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("evidence index authority snapshot has trailing data")
	}
	entries := make(map[string]v2EvidenceIndexEntry, len(index.Entries))
	for _, entry := range index.Entries {
		if _, exists := entries[entry.RelativePath]; exists {
			return nil, fmt.Errorf("duplicate evidence index path %q", entry.RelativePath)
		}
		entries[entry.RelativePath] = entry
	}
	return entries, nil
}
