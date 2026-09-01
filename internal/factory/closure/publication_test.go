// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// requireLinuxSameFS skips a test that exercises the
// production Linux same-filesystem transactional
// publication path. The guard is preserved on Linux so
// the test continues to assert against production
// semantics there; macOS / other platforms skip because
// the production refuse-to-publish behaviour is the
// intended outcome, not a defect.
//
// ACT-LEAMAS-CLOSURE-CANONICAL-VERIFICATION-DEFECTS-CORRECTION01
// §18 permits this classification: production semantics
// are unchanged; only the test's reachability is gated.
func requireLinuxSameFS(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux same-filesystem transaction support")
	}
}

func TestArtifactRoleModelClassifiesClosureOutputs(t *testing.T) {
	cases := []struct {
		id   string
		path string
		role ArtifactRole
	}{
		{"manifest", "docs/closure-manifests/ACT-LEAMAS-TEST01.json", ArtifactRoleGeneratedOutput},
		{"report", "docs/close-reports/ACT-LEAMAS-TEST01.md", ArtifactRoleGeneratedOutput},
		{"success_erratum", "docs/lifecycle-errata/ACT-LEAMAS-TEST01.json", ArtifactRoleNotRequired},
		{"failure_erratum", "docs/lifecycle-errata/ACT-LEAMAS-TEST01.failure.json", ArtifactRoleFailureErratum},
		{"attestation", "docs/closure-manifests/ACT-LEAMAS-TEST01.attestation.json", ArtifactRolePostCommitEvidence},
		{"input", ".factory/summary.json", ArtifactRoleInput},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			got := ArtifactRoleFor(PlanArtifact{ID: tc.id, Path: tc.path})
			if got != tc.role {
				t.Fatalf("role=%q want=%q", got, tc.role)
			}
		})
	}
}

func TestCloseRunPublishesCompleteArtifactSet(t *testing.T) {
	requireLinuxSameFS(t)
	destination := t.TempDir()
	files := publicationTestFiles()
	if err := PublishArtifactSet(PublicationOptions{Destination: destination, Files: files}); err != nil {
		t.Fatalf("PublishArtifactSet: %v", err)
	}
	for path, want := range files {
		got, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(got) != string(want) {
			t.Errorf("%s=%q want=%q", path, got, want)
		}
	}
}

func TestCloseRunFailureLeavesNoPartialCanonicalSet(t *testing.T) {
	requireLinuxSameFS(t)
	points := []PublicationFailurePoint{
		PublicationFailureStagingDirectory,
		PublicationFailureManifestStagedWrite,
		PublicationFailureReportStagedWrite,
		PublicationFailureErratumStagedWrite,
		PublicationFailureSchemaValidation,
		PublicationFailureBoundValidation,
		PublicationFailureHashValidation,
		PublicationFailureFirstPublication,
		PublicationFailureLaterPublication,
	}
	for _, point := range points {
		t.Run(string(point), func(t *testing.T) {
			destination := t.TempDir()
			err := PublishArtifactSet(PublicationOptions{Destination: destination, Files: publicationTestFiles(), FailurePoint: point})
			if err == nil {
				t.Fatalf("failure point %q did not fail", point)
			}
			if got := canonicalPublicationFiles(t, destination); len(got) != 0 {
				t.Fatalf("partial canonical set after %s: %v", point, got)
			}
		})
	}
}

func TestCloseRunExistingIdenticalSetIsIdempotent(t *testing.T) {
	requireLinuxSameFS(t)
	destination := t.TempDir()
	files := publicationTestFiles()
	if err := PublishArtifactSet(PublicationOptions{Destination: destination, Files: files}); err != nil {
		t.Fatal(err)
	}
	if err := PublishArtifactSet(PublicationOptions{Destination: destination, Files: files}); err != nil {
		t.Fatalf("identical publication: %v", err)
	}
}

func TestCloseRunExistingConflictingSetFailsClosed(t *testing.T) {
	requireLinuxSameFS(t)
	destination := t.TempDir()
	files := publicationTestFiles()
	if err := PublishArtifactSet(PublicationOptions{Destination: destination, Files: files}); err != nil {
		t.Fatal(err)
	}
	files["docs/close-reports/ACT-LEAMAS-TEST01.md"] = []byte("different\n")
	if err := PublishArtifactSet(PublicationOptions{Destination: destination, Files: files}); err == nil {
		t.Fatal("conflicting publication was accepted")
	}
}

func TestCloseRunRecoversInterruptedPublication(t *testing.T) {
	requireLinuxSameFS(t)
	destination := t.TempDir()
	if err := PublishArtifactSet(PublicationOptions{
		Destination:  destination,
		Files:        publicationTestFiles(),
		FailurePoint: PublicationFailureInterruptedPublication,
	}); err == nil {
		t.Fatal("interrupted publication did not fail")
	}
	if _, err := os.Stat(filepath.Join(destination, publicationMarkerName)); err != nil {
		t.Fatalf("interrupted publication did not leave recovery marker: %v", err)
	}
	if err := PublishArtifactSet(PublicationOptions{Destination: destination, Files: publicationTestFiles()}); err != nil {
		t.Fatalf("recovery publication: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, publicationMarkerName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recovery marker remains: %v", err)
	}
}

func publicationTestFiles() map[string][]byte {
	return map[string][]byte{
		"docs/closure-manifests/ACT-LEAMAS-TEST01.json":        []byte("{\"act_id\":\"ACT-LEAMAS-TEST01\"}\n"),
		"docs/close-reports/ACT-LEAMAS-TEST01.md":              []byte("# report\n"),
		"docs/lifecycle-errata/ACT-LEAMAS-TEST01.failure.json": []byte("{\"reason_code\":\"check_failed\"}\n"),
	}
}

func canonicalPublicationFiles(t *testing.T, destination string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(destination, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == destination || entry.IsDir() || filepath.Base(path) == publicationMarkerName || strings.Contains(filepath.Base(path), ".leamas-close-stage-") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return paths
}
