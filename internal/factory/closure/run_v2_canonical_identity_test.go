// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestV2CanonicalIdentityInvarianceMatrix(t *testing.T) {
	base := canonicalArtifactTestInput()
	baseArtifacts := mustGenerateCanonicalArtifacts(t, base)
	cases := []struct {
		name   string
		mutate func(*v2CanonicalArtifactInput)
	}{
		{name: "branch", mutate: func(in *v2CanonicalArtifactInput) { in.Branch = "renamed" }},
		{name: "runner version", mutate: func(in *v2CanonicalArtifactInput) { in.Runner.LeamasVersion = "other" }},
		{name: "binary hash", mutate: func(in *v2CanonicalArtifactInput) { in.Runner.BinarySHA256 = strings.Repeat("9", 64) }},
		{name: "stdout hash", mutate: func(in *v2CanonicalArtifactInput) { in.Checks[0].StdoutSHA256 = strings.Repeat("8", 64) }},
		{name: "stderr hash", mutate: func(in *v2CanonicalArtifactInput) { in.Checks[0].StderrSHA256 = strings.Repeat("7", 64) }},
		{name: "byte counts", mutate: func(in *v2CanonicalArtifactInput) {
			in.Checks[0].StdoutByteCount = 91
			in.Checks[0].StderrByteCount = 92
			in.Checks[0].OutputBytesObserved = 183
		}},
		{name: "timestamps and duration", mutate: func(in *v2CanonicalArtifactInput) {
			in.Checks[0].StartedAtUTC = "2030-01-01T00:00:00Z"
			in.Checks[0].FinishedAtUTC = "2030-01-01T00:00:05Z"
			in.Checks[0].DurationMS = 5000
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := cloneCanonicalArtifactInput(base)
			tc.mutate(&input)
			got := mustGenerateCanonicalArtifacts(t, input)
			assertCanonicalArtifactsEqual(t, baseArtifacts, got)
		})
	}
}

func TestV2CanonicalIdentitySensitivityMatrix(t *testing.T) {
	base := canonicalArtifactTestInput()
	baseArtifacts := mustGenerateCanonicalArtifacts(t, base)
	failedExit := 7
	cases := []struct {
		name   string
		mutate func(*v2CanonicalArtifactInput)
	}{
		{name: "plan SHA-256", mutate: func(in *v2CanonicalArtifactInput) { in.PlanSHA256 = strings.Repeat("9", 64) }},
		{name: "freeze commit", mutate: func(in *v2CanonicalArtifactInput) { in.FreezeCommit = strings.Repeat("8", 40) }},
		{name: "subject commit", mutate: func(in *v2CanonicalArtifactInput) { in.SubjectCommit = strings.Repeat("7", 40) }},
		{name: "frozen argv", mutate: func(in *v2CanonicalArtifactInput) { in.Checks[0].Argv = []string{"go", "env"} }},
		{name: "check status", mutate: func(in *v2CanonicalArtifactInput) { in.Checks[0].Status = CheckStatusFail }},
		{name: "exit code", mutate: func(in *v2CanonicalArtifactInput) { in.Checks[0].ExitCode = &failedExit }},
		{name: "truncated output", mutate: func(in *v2CanonicalArtifactInput) { in.Checks[0].OutputTruncated = true }},
		{name: "incomplete output", mutate: func(in *v2CanonicalArtifactInput) { in.Checks[0].OutputIncomplete = true }},
		{name: "patch policy", mutate: func(in *v2CanonicalArtifactInput) { in.PatchHygiene.Status = CheckStatusFail }},
		{name: "closure policy", mutate: func(in *v2CanonicalArtifactInput) { in.ClosurePolicy.TrackedFullDigestStatus = CheckStatusFail }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := cloneCanonicalArtifactInput(base)
			tc.mutate(&input)
			got := mustGenerateCanonicalArtifacts(t, input)
			if bytes.Equal(baseArtifacts.ManifestBytes, got.ManifestBytes) &&
				bytes.Equal(baseArtifacts.ReportBytes, got.ReportBytes) {
				t.Fatal("semantic identity change left both canonical artifacts unchanged")
			}
		})
	}
}

func TestV2CanonicalIdentityProducesSameObjectsForRuntimeVariants(t *testing.T) {
	repo, subject, subjectTree := prepareObjectTransactionRepository(t, ObjectFormatSHA1)
	base := canonicalArtifactTestInput()
	firstArtifacts := mustGenerateCanonicalArtifacts(t, base)
	first, err := buildV2ClosureObjects(context.Background(), RealGit{}, repo,
		ObjectFormatSHA1, subject, subjectTree, objectTransactionActID, firstArtifacts)
	if err != nil {
		t.Fatal(err)
	}
	variant := cloneCanonicalArtifactInput(base)
	variant.Branch = "renamed"
	variant.Runner.LeamasVersion = "different-build"
	variant.Runner.BinarySHA256 = strings.Repeat("9", 64)
	variant.Checks[0].StdoutSHA256 = strings.Repeat("8", 64)
	variant.Checks[0].StderrSHA256 = strings.Repeat("7", 64)
	variant.Checks[0].StdoutByteCount = 11
	variant.Checks[0].StderrByteCount = 12
	variant.Checks[0].OutputBytesObserved = 23
	variant.Checks[0].StartedAtUTC = "2030-01-01T00:00:00Z"
	variant.Checks[0].FinishedAtUTC = "2030-01-01T00:00:09Z"
	variant.Checks[0].DurationMS = 9000
	secondArtifacts := mustGenerateCanonicalArtifacts(t, variant)
	second, err := buildV2ClosureObjects(context.Background(), RealGit{}, repo,
		ObjectFormatSHA1, subject, subjectTree, objectTransactionActID, secondArtifacts)
	if err != nil {
		t.Fatal(err)
	}
	assertCanonicalArtifactsEqual(t, firstArtifacts, secondArtifacts)
	if first != second {
		t.Fatalf("runtime variants changed canonical objects: first=%+v second=%+v", first, second)
	}
}

func cloneCanonicalArtifactInput(input v2CanonicalArtifactInput) v2CanonicalArtifactInput {
	clone := input
	clone.Checks = append([]CheckResult(nil), input.Checks...)
	for i := range clone.Checks {
		clone.Checks[i].Argv = append([]string(nil), input.Checks[i].Argv...)
		clone.Checks[i].OverriddenEnvironment = append([]string(nil), input.Checks[i].OverriddenEnvironment...)
		if input.Checks[i].ExitCode != nil {
			exitCode := *input.Checks[i].ExitCode
			clone.Checks[i].ExitCode = &exitCode
		}
	}
	return clone
}

func mustGenerateCanonicalArtifacts(t *testing.T, input v2CanonicalArtifactInput) v2CanonicalArtifacts {
	t.Helper()
	artifacts, err := generateV2CanonicalArtifacts(input)
	if err != nil {
		t.Fatal(err)
	}
	return artifacts
}

func assertCanonicalArtifactsEqual(t *testing.T, want, got v2CanonicalArtifacts) {
	t.Helper()
	if !bytes.Equal(want.ManifestBytes, got.ManifestBytes) ||
		!bytes.Equal(want.ReportBytes, got.ReportBytes) ||
		want.ManifestSHA256 != got.ManifestSHA256 || want.ReportSHA256 != got.ReportSHA256 {
		t.Fatal("runtime-only variation changed canonical artifact identity")
	}
}
