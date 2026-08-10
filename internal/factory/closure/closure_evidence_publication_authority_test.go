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

func evidencePubFixture(t *testing.T) (worktree, outside, json string) {
	t.Helper()
	root := t.TempDir()
	worktree = filepath.Join(root, "worktree")
	if err := os.Mkdir(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	outside = filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	json = filepath.Join(outside, "evidence.json")
	return
}

func evidencePubBytes(seed byte) []byte {
	return []byte("{\"contract_version\":1,\"seed\":" + string(rune('a'+seed)) + "}\n")
}

// TestClosureEvidencePublicationAuthoritySuccess pins the
// happy path: Prepare + Publish produces a durable pair whose
// JSON bytes match the candidate, whose sidecar matches the
// digest, and whose contents survive re-open.
func TestClosureEvidencePublicationAuthoritySuccess(t *testing.T) {
	worktree, _, json := evidencePubFixture(t)
	bytes := evidencePubBytes(0)
	digest := sha256.Sum256(bytes)
	auth, err := PrepareEvidencePublication(worktree, json, []CanonicalWorktree{{Path: worktree}})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	defer auth.Close()
	if err := auth.Publish(PublicationCandidate{Bytes: bytes, SHA256: digest}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	got, err := os.ReadFile(json)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	if string(got) != string(bytes) {
		t.Fatalf("bytes mismatch: %q vs %q", got, bytes)
	}
	sidecar, err := os.ReadFile(json + ".sha256")
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	if strings.TrimSpace(string(sidecar)) != hex.EncodeToString(digest[:]) {
		t.Fatalf("sidecar mismatch: %q", sidecar)
	}
}

// TestClosureEvidencePublicationMonotonicity verifies the
// complete-once contract: a second Prepare against the same
// destination fails because the pair already exists.
func TestClosureEvidencePublicationMonotonicity(t *testing.T) {
	worktree, _, json := evidencePubFixture(t)
	bytes := evidencePubBytes(1)
	digest := sha256.Sum256(bytes)
	first, err := PrepareEvidencePublication(worktree, json, []CanonicalWorktree{{Path: worktree}})
	if err != nil {
		t.Fatalf("first prepare: %v", err)
	}
	if err := first.Publish(PublicationCandidate{Bytes: bytes, SHA256: digest}); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	_ = first.Close()
	second, err := PrepareEvidencePublication(worktree, json, []CanonicalWorktree{{Path: worktree}})
	if err == nil {
		t.Fatalf("second prepare must fail; got %v", second)
	}
	original, _ := os.ReadFile(json)
	if string(original) != string(bytes) {
		t.Fatalf("original bytes changed: %q", original)
	}
}
