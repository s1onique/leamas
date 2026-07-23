package closure

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func commitFile(t *testing.T, repository, path, content, message string) string {
	t.Helper()
	absolute := filepath.Join(repository, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absolute, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "--", path}, {"commit", "-m", message}} {
		if _, err := runGitValue(context.Background(), realGitClient{}, repository, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	commit, err := runGitValue(context.Background(), realGitClient{}, repository, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return commit
}

func TestClosurePolicyRejectsNewTrackedFullDigest(t *testing.T) {
	repository, baseline := newGitRepository(t)
	subject := commitFile(t, repository, "docs/evidence.txt", "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\nbody\n", "add digest")
	result, _ := evaluateTrackedDigestPolicy(context.Background(), realGitClient{}, repository, baseline, subject)
	if result.TrackedFullDigestStatus != CheckStatusFail || result.DiagnosticCount != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestClosurePolicyAllowsCompactManifest(t *testing.T) {
	repository, baseline := newGitRepository(t)
	subject := commitFile(t, repository, "docs/closure-manifests/ACT-TEST.json", `{"verdict":"pass","digest_sha256":"abc"}`+"\n", "add manifest")
	result, _ := evaluateTrackedDigestPolicy(context.Background(), realGitClient{}, repository, baseline, subject)
	if result.TrackedFullDigestStatus != CheckStatusPass {
		t.Fatalf("result = %+v", result)
	}
}

func TestClosurePolicyAllowsCompactCloseReport(t *testing.T) {
	repository, baseline := newGitRepository(t)
	subject := commitFile(t, repository, "docs/close-reports/ACT-TEST.md", "# Close Report\n\nDigest SHA-256: abc\n", "add report")
	result, _ := evaluateTrackedDigestPolicy(context.Background(), realGitClient{}, repository, baseline, subject)
	if result.TrackedFullDigestStatus != CheckStatusPass {
		t.Fatalf("result = %+v", result)
	}
}

func TestClosurePolicyDoesNotReopenUnchangedLegacyDigest(t *testing.T) {
	repository, _ := newGitRepository(t)
	baseline := commitFile(t, repository, "docs/legacy.txt", "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 2\nlegacy\n", "legacy digest")
	subject := commitFile(t, repository, "normal.txt", "normal\n", "normal change")
	result, _ := evaluateTrackedDigestPolicy(context.Background(), realGitClient{}, repository, baseline, subject)
	if result.TrackedFullDigestStatus != CheckStatusPass {
		t.Fatalf("result = %+v", result)
	}
}
