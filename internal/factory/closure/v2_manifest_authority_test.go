// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewV2ManifestRequiresCompleteBinaryIdentity(t *testing.T) {
	base := validV2ManifestBuild(t)
	cases := []struct {
		name   string
		mutate func(*V2ManifestBuild)
	}{
		{name: "missing path", mutate: func(b *V2ManifestBuild) { b.BinaryIdentity.Path = "" }},
		{name: "relative path", mutate: func(b *V2ManifestBuild) { b.BinaryIdentity.Path = "leamas" }},
		{name: "missing sha", mutate: func(b *V2ManifestBuild) { b.BinaryIdentity.SHA256 = "" }},
		{name: "malformed sha", mutate: func(b *V2ManifestBuild) { b.BinaryIdentity.SHA256 = "not-a-sha" }},
		{name: "sha does not bind file", mutate: func(b *V2ManifestBuild) { b.BinaryIdentity.SHA256 = strings.Repeat("a", 64) }},
		{name: "missing revision", mutate: func(b *V2ManifestBuild) { b.BinaryIdentity.VCSRevision = "" }},
		{name: "short revision", mutate: func(b *V2ManifestBuild) { b.BinaryIdentity.VCSRevision = strings.Repeat("a", 39) }},
		{name: "uppercase revision", mutate: func(b *V2ManifestBuild) { b.BinaryIdentity.VCSRevision = strings.Repeat("A", 40) }},
		{name: "missing version", mutate: func(b *V2ManifestBuild) { b.BinaryIdentity.LeamasVersion = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			build := cloneV2ManifestBuild(base)
			tc.mutate(&build)
			_, err := NewV2Manifest(build)
			assertV2ManifestCode(t, err, "binary_identity_invalid")
		})
	}
}

func TestNewV2ManifestRequiresStrictProtocolAndPlanVersions(t *testing.T) {
	base := validV2ManifestBuild(t)
	cases := []struct {
		name string
		edit func(*V2ManifestBuild)
		want V2DiagnosticCode
	}{
		{name: "protocol one", edit: func(b *V2ManifestBuild) { b.ClosureProtocolVersion = ClosureProtocolV1 }, want: V2CodeUnsupportedClosureProtocolVersion},
		{name: "protocol future", edit: func(b *V2ManifestBuild) { b.ClosureProtocolVersion = ClosureProtocolVersion("9") }, want: V2CodeUnsupportedClosureProtocolVersion},
		{name: "plan zero", edit: func(b *V2ManifestBuild) { b.PlanContractVersion = 0 }, want: V2CodeUnsupportedPlanContractVersion},
		{name: "plan future", edit: func(b *V2ManifestBuild) { b.PlanContractVersion = PlanContractVersion(2) }, want: V2CodeUnsupportedPlanContractVersion},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			build := cloneV2ManifestBuild(base)
			tc.edit(&build)
			_, err := NewV2Manifest(build)
			assertV2ManifestCode(t, err, tc.want)
		})
	}
}

func TestNewV2ManifestRejectsUnboundRuntimeIdentity(t *testing.T) {
	base := validV2ManifestBuild(t)
	cases := []struct {
		name string
		edit func(*V2ManifestBuild)
	}{
		{name: "subject commit", edit: func(b *V2ManifestBuild) { b.SubjectCommit = "subject" }},
		{name: "subject tree", edit: func(b *V2ManifestBuild) { b.SubjectTree = "tree" }},
		{name: "freeze commit", edit: func(b *V2ManifestBuild) { b.FreezeCommit = "freeze" }},
		{name: "freeze tree", edit: func(b *V2ManifestBuild) { b.FreezeTree = "tree" }},
		{name: "plan blob", edit: func(b *V2ManifestBuild) { b.PlanBlob = "blob" }},
		{name: "caller head", edit: func(b *V2ManifestBuild) { b.CallerHead = "caller" }},
		{name: "execution tree", edit: func(b *V2ManifestBuild) { b.ExecutionTree = strings.Repeat("c", 40) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			build := cloneV2ManifestBuild(base)
			tc.edit(&build)
			_, err := NewV2Manifest(build)
			assertV2ManifestCode(t, err, "manifest_identity_invalid")
		})
	}
}

func validV2ManifestBuild(t *testing.T) V2ManifestBuild {
	t.Helper()
	binaryPath := filepath.Join(t.TempDir(), "leamas")
	binaryBytes := []byte("deterministic fake leamas binary\n")
	if err := os.WriteFile(binaryPath, binaryBytes, 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(binaryBytes)
	planBytes := []byte(`{"contract_version":1}`)
	planSum := sha256.Sum256(planBytes)
	build := V2ManifestBuild{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    PlanContractV1,
		SubjectCommit:          strings.Repeat("1", 40),
		SubjectTree:            strings.Repeat("2", 40),
		FreezeCommit:           strings.Repeat("3", 40),
		FreezeTree:             strings.Repeat("4", 40),
		PlanPath:               "docs/closure-plans/ACT.json",
		PlanBlob:               strings.Repeat("5", 40),
		PlanSHA256:             hex.EncodeToString(planSum[:]),
		PlanBytes:              planBytes,
		ExecutionTree:          strings.Repeat("2", 40),
		CallerHead:             strings.Repeat("6", 40),
		BinaryIdentity: V2BinaryIdentity{
			Path:          binaryPath,
			SHA256:        hex.EncodeToString(sum[:]),
			VCSRevision:   strings.Repeat("7", 40),
			VCSModified:   false,
			LeamasVersion: "0.1.0+test",
		},
		PlanChecks: []PlanCheck{{ID: "run", Mode: CheckModeRun}},
	}
	build.ExecutionResults = []CheckResult{
		completedV2ExecutionResult("run", build.SubjectTree, 0, 1),
	}
	build.Evidence = v2ResultEvidence("run")
	return build
}

func cloneV2ManifestBuild(in V2ManifestBuild) V2ManifestBuild {
	out := in
	out.PlanBytes = append([]byte(nil), in.PlanBytes...)
	out.PlanChecks = append([]PlanCheck(nil), in.PlanChecks...)
	out.ExecutionResults = append([]CheckResult(nil), in.ExecutionResults...)
	out.Evidence = append([]EvidenceRecord(nil), in.Evidence...)
	out.CheckResults = append([]V2CheckResult(nil), in.CheckResults...)
	return out
}

func assertV2ManifestCode(t *testing.T, err error, want V2DiagnosticCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s, got nil", want)
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(want) {
		t.Fatalf("diagnostics=%v, want %s", v2err.Diags.Codes(), want)
	}
}
