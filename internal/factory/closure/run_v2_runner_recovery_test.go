// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"reflect"
	"strings"
	"testing"
)

func TestRunClosureV2RunnerAuthorityRejectsEveryRecoveryStateWithoutMutation(t *testing.T) {
	states := []string{"PREPARED", "POST_REF", "VERIFIED"}
	variants := []struct {
		name   string
		mutate func(v2RepositoryFixture, *v2Dependencies)
	}{
		{name: "stale", mutate: func(f v2RepositoryFixture, deps *v2Dependencies) {
			deps.Runner = fixedRunnerIdentity{value: RunnerIdentity{
				VCSRevision: strings.Repeat("f", len(f.subject)), BinarySHA256: testV2BinaryHash,
			}}
		}},
		{name: "modified", mutate: func(f v2RepositoryFixture, deps *v2Dependencies) {
			deps.Runner = fixedRunnerIdentity{value: RunnerIdentity{
				VCSRevision: f.subject, VCSModified: true, BinarySHA256: testV2BinaryHash,
			}}
		}},
		{name: "hash-mismatched", mutate: func(_ v2RepositoryFixture, deps *v2Dependencies) {
			deps.RunningBinarySHA256 = func() (string, error) {
				return strings.Repeat("b", 64), nil
			}
		}},
	}

	for _, state := range states {
		for _, variant := range variants {
			t.Run(state+"/"+variant.name, func(t *testing.T) {
				fixture := prepareV2RecoveryState(t, state)
				before := captureV2StableState(t, fixture)
				recorder := &v2RecordingGit{delegate: RealGit{}}
				checks := 0
				deps := productionV2TestDependencies(fixture, recorder, &checks)
				variant.mutate(fixture, &deps)

				_, err := runProductionV2Test(fixture, deps)
				if err == nil || !strings.Contains(err.Error(), "runner") {
					t.Fatalf("err = %v, want runner authority rejection", err)
				}
				if checks != 0 {
					t.Fatalf("recovery checks = %d, want 0", checks)
				}
				if len(recorder.objectWrites) != 0 {
					t.Fatalf("object writes before rejection: %v", recorder.objectWrites)
				}
				after := captureV2StableState(t, fixture)
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("authority rejection mutated %s state:\nbefore=%#v\nafter=%#v", state, before, after)
				}
			})
		}
	}
}

func prepareV2RecoveryState(t *testing.T, state string) v2RepositoryFixture {
	t.Helper()
	fixture := prepareV2Repository(t)
	var git gitClient = RealGit{}
	switch state {
	case "PREPARED":
		git = &v2FailingGit{failCommand: "update-ref"}
	case "POST_REF":
		git = &v2FailingGit{failCommand: "reset"}
	case "VERIFIED":
	default:
		t.Fatalf("unknown recovery state %q", state)
	}
	_, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, git, nil))
	if state == "VERIFIED" && err != nil {
		t.Fatalf("prepare VERIFIED: %v", err)
	}
	if state != "VERIFIED" && err == nil {
		t.Fatalf("prepare %s unexpectedly completed", state)
	}
	return fixture
}
