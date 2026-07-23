package closure

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestClosureTagBindsSubjectAndArtifactHashes(t *testing.T) {
	fixture := prepareClosureFixture(t)
	message, err := CreateTag(context.Background(), fixture.options())
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(fixture.manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	reportBytes, err := os.ReadFile(fixture.reportPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest, _, err := LoadManifest(fixture.manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, binding := range []string{
		"subject_commit_oid: " + manifest.Subject.CommitOID,
		"subject_tree_oid: " + manifest.Subject.TreeOID,
		"manifest_sha256: " + SHA256Hex(manifestBytes),
		"report_sha256: " + SHA256Hex(reportBytes),
	} {
		if !bytes.Contains(message, []byte(binding+"\n")) {
			t.Fatalf("tag message missing %q:\n%s", binding, message)
		}
	}
}

func TestClosureTagRejectsFailedManifest(t *testing.T) {
	fixture := prepareClosureFixture(t)
	manifest, _, err := LoadManifest(fixture.manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	plan, _, err := LoadPlan(filepath.Join(fixture.repository, filepath.FromSlash(manifest.Plan.Path)))
	if err != nil {
		t.Fatal(err)
	}
	manifest.Runner.VCSModified = true
	manifest.Verdict = VerdictFail
	manifestBytes, err := MarshalManifest(manifest, plan)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Render(manifest, plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.manifestPath, manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.reportPath, report, 0o644); err != nil {
		t.Fatal(err)
	}
	commitClosureChanges(t, fixture.repository, "failed closure")
	if _, err := CreateTag(context.Background(), fixture.options()); err == nil || !strings.Contains(err.Error(), "passing manifest") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosureTagRejectsReportDrift(t *testing.T) {
	fixture := prepareClosureFixture(t)
	report, err := os.ReadFile(fixture.reportPath)
	if err != nil {
		t.Fatal(err)
	}
	report = bytes.Replace(report, []byte("## Verdict"), []byte("## Edited verdict"), 1)
	if err := os.WriteFile(fixture.reportPath, report, 0o644); err != nil {
		t.Fatal(err)
	}
	commitClosureChanges(t, fixture.repository, "manual report drift")
	if _, err := CreateTag(context.Background(), fixture.options()); err == nil || !strings.Contains(err.Error(), "deterministic") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosureTagRejectsManifestDrift(t *testing.T) {
	fixture := prepareClosureFixture(t)
	manifest, err := os.ReadFile(fixture.manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest = append(manifest, '\n')
	if err := os.WriteFile(fixture.manifestPath, manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTag(context.Background(), fixture.options()); err == nil || !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosureTagHasNoForceOption(t *testing.T) {
	typeOfOptions := reflect.TypeOf(TagOptions{})
	for index := range typeOfOptions.NumField() {
		if strings.Contains(strings.ToLower(typeOfOptions.Field(index).Name), "force") {
			t.Fatalf("TagOptions exposes a force option: %s", typeOfOptions.Field(index).Name)
		}
	}
}

func commitClosureChanges(t *testing.T, repository, message string) {
	t.Helper()
	if _, err := runGitValue(context.Background(), realGitClient{}, repository, "add", "docs/closure-manifests", "docs/close-reports"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := runGitValue(context.Background(), realGitClient{}, repository, "commit", "-m", message); err != nil {
		t.Fatalf("git commit: %v", err)
	}
}
