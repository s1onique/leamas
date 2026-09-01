// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
)

func TestCloseRunDoesNotRequireItsOwnOutputsAsInputs(t *testing.T) {
	repo, freezeArg, subject, planPath := prepareFreezeAndSubject(t)
	executor := &recordingExecutor{results: []*execution.Result{successExecution("one", "")}}
	destination := t.TempDir()
	options := RunOptions{
		PlanPath:            planPath,
		Subject:             subject,
		EvidenceDirectory:   filepath.Join(t.TempDir(), "evidence"),
		ManifestOutput:      filepath.Join(destination, "manifest.json"),
		RepositoryDirectory: repo,
		PlanFreeze:          freezeArg,
	}
	if _, _, err := runClosureWithDependencies(context.Background(), options, passingRunDependencies(subject, executor)); err != nil {
		t.Fatalf("close run with generated outputs not yet on disk: %v", err)
	}
}

func TestCloseRunRefusesPartialPublicationSet(t *testing.T) {
	// ACT-LEAMAS-CLOSURE-CANONICAL-VERIFICATION-DEFECTS-CORRECTION01
	// §18: the partial-publication guard is a Linux same-
	// filesystem transactional guarantee. Production
	// refuses to publish on macOS by design; the test is
	// platform-conditional so the guard itself stays
	// exercised on Linux CI.
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux same-filesystem transaction support")
	}
	destination := t.TempDir()
	files := map[string][]byte{
		"docs/closure-manifests/ACT-LEAMAS-TEST01.json": []byte("{\"act_id\":\"ACT-LEAMAS-TEST01\"}\n"),
		"docs/close-reports/ACT-LEAMAS-TEST01.md":       []byte("# report\n"),
	}
	if err := PublishArtifactSet(PublicationOptions{Destination: destination, Files: files}); err != nil {
		t.Fatalf("baseline full publication: %v", err)
	}
	if err := os.Remove(filepath.Join(destination, "docs/close-reports/ACT-LEAMAS-TEST01.md")); err != nil {
		t.Fatal(err)
	}
	// A conflicting set (one existing file changed) is also partial-conflict
	// and must fail closed.
	conflict := map[string][]byte{
		"docs/closure-manifests/ACT-LEAMAS-TEST01.json": []byte("{\"act_id\":\"ACT-LEAMAS-TEST01\"}\n"),
		"docs/close-reports/ACT-LEAMAS-TEST01.md":       []byte("# different\n"),
	}
	if err := PublishArtifactSet(PublicationOptions{Destination: destination, Files: conflict}); err == nil || !strings.Contains(err.Error(), "partial") {
		t.Fatalf("conflicting publication was accepted: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destination, "docs/close-reports/ACT-LEAMAS-TEST01.md"), []byte("# report\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := PublishArtifactSet(PublicationOptions{Destination: destination, Files: files}); err != nil {
		t.Fatalf("idempotent publication: %v", err)
	}
}
