// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRunClosureV2RejectsUnauthorizedPreparedEvidenceBeforeObjectOrRefMutation(t *testing.T) {
	mutations := []struct {
		name   string
		mutate func(*v2RuntimeEvidence)
	}{
		{name: "missing frozen checks", mutate: func(runtime *v2RuntimeEvidence) {
			runtime.Checks = nil
			runtime.CheckEvidence = nil
		}},
		{name: "runner binary mismatch", mutate: func(runtime *v2RuntimeEvidence) {
			runtime.Runner.BinarySHA256 = strings.Repeat("b", 64)
		}},
		{name: "check command mismatch", mutate: func(runtime *v2RuntimeEvidence) {
			runtime.Checks[0].Argv = []string{"false"}
		}},
		{name: "failed check", mutate: func(runtime *v2RuntimeEvidence) {
			exitCode := 1
			runtime.Checks[0].ExitCode = &exitCode
			runtime.Checks[0].Status = CheckStatusFail
		}},
		{name: "failed patch policy", mutate: func(runtime *v2RuntimeEvidence) {
			runtime.PatchHygiene.Status = CheckStatusFail
		}},
		{name: "detached output mismatch", mutate: func(runtime *v2RuntimeEvidence) {
			runtime.CheckEvidence[0].SHA256 = strings.Repeat("c", 64)
		}},
	}

	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			fixture := prepareV2RecoveryState(t, "PREPARED")
			rewriteV2RuntimeEvidence(t, fixture, mutation.mutate)
			before := captureV2StableState(t, fixture)
			recorder := &v2RecordingGit{delegate: RealGit{}}
			checks := 0

			_, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, recorder, &checks))
			if err == nil || !strings.Contains(err.Error(), "authorize recovery evidence") {
				t.Fatalf("err = %v, want recovery-evidence authority rejection", err)
			}
			if checks != 0 {
				t.Fatalf("recovery checks = %d, want 0", checks)
			}
			if len(recorder.objectWrites) != 0 {
				t.Fatalf("object writes preceded evidence rejection: %v", recorder.objectWrites)
			}
			after := captureV2StableState(t, fixture)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("evidence rejection mutated repository:\nbefore=%#v\nafter=%#v", before, after)
			}
			if got := v2Git(t, fixture.root, "rev-parse", "HEAD"); got != fixture.subject {
				t.Fatalf("branch moved to %s", got)
			}
			assertV2TagAbsent(t, fixture)
		})
	}
}

func rewriteV2RuntimeEvidence(t *testing.T, fixture v2RepositoryFixture, mutate func(*v2RuntimeEvidence)) {
	t.Helper()
	evidenceDir := v2EvidencePath(fixture)
	runtimePath := filepath.Join(evidenceDir, "runtime.json")
	data, err := os.ReadFile(runtimePath)
	if err != nil {
		t.Fatal(err)
	}
	var runtime v2RuntimeEvidence
	if err := json.Unmarshal(data, &runtime); err != nil {
		t.Fatal(err)
	}
	mutate(&runtime)
	data, err = json.MarshalIndent(runtime, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimePath, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(evidenceDir, v2EvidenceIndexName)); err != nil {
		t.Fatal(err)
	}
	if _, err := buildV2EvidenceIndex(evidenceDir); err != nil {
		t.Fatal(err)
	}
}
