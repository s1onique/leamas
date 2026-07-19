// Package digest provides targeted digest generation for Git repositories.
//
// Tests that bind evidence hashes to the corrected manifest and
// statistics, and that assert that changing the status semantics
// changes the appropriate hashes. These exist to lock the contract
// the ACT introduces: a status change must update CHANGESET_MANIFEST,
// CHANGESET_STATS and the aggregate digest evidence.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// evidenceHash extracts a named hash from the EVIDENCE_HASHES section
// of a digest. Returns "" if missing.
func evidenceHash(digestText, name string) string {
	idx := strings.Index(digestText, "## EVIDENCE_HASHES")
	if idx == -1 {
		return ""
	}
	body := digestText[idx:]
	end := strings.Index(body, "\n## ")
	if end != -1 {
		body = body[:end]
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, name+"=") {
			return strings.TrimPrefix(line, name+"=")
		}
	}
	return ""
}

// extractSection returns the substring starting at a `## <name>`
// heading and ending just before the next `## ` heading (or end of
// document). The heading line itself is included.
func extractSection(digestText, name string) string {
	start := "## " + name
	idx := strings.Index(digestText, start)
	if idx == -1 {
		return ""
	}
	rest := digestText[idx:]
	rel := strings.Index(rest[2:], "## ")
	if rel == -1 {
		return rest
	}
	// rel counts from position 2 to skip the leading "## " of start;
	// add 2 to translate back to rest's offsets.
	return rest[:2+rel]
}

// rebuildEvidenceHashes splices the supplied manifest section and
// statistics section back into the digest body and recomputes the
// digest_evidence_sha256 (and per-section sha256 fields) using the
// digest package's own helpers.
func rebuildEvidenceHashes(original string, manifestSection, statsSection string) string {
	newDigest := strings.Replace(original, extractSection(original, "CHANGESET_MANIFEST"), manifestSection, 1)
	newDigest = strings.Replace(newDigest, extractSection(newDigest, "CHANGESET_STATS"), statsSection, 1)

	m := extractSection(newDigest, "CHANGESET_MANIFEST")
	s := extractSection(newDigest, "CHANGESET_STATS")
	rm := extractSection(newDigest, "REVIEW_MAP")
	rs := extractSection(newDigest, "RISK_SIGNALS")
	ph := extractSection(newDigest, "PATCH_HYGIENE")
	ps := extractSection(newDigest, "PUBLIC_SURFACE_DELTA")
	dd := extractSection(newDigest, "DEPENDENCY_DELTA")
	gs := extractSection(newDigest, "GATE_SUMMARY")
	fe := extractSection(newDigest, "Changed files") + "\n\n" + extractSection(newDigest, "Diffs")

	hashes := ComputeEvidenceHashes(m, s, rm, rs, ph, ps, dd, gs, fe)

	oldEH := extractSection(newDigest, "EVIDENCE_HASHES")
	if oldEH == "" {
		return newDigest
	}
	newEH := RenderEvidenceHashes(hashes)
	return strings.Replace(newDigest, oldEH, newEH, 1)
}

// flipStatusInManifest reclassifies every `M ` entry for `path` in the
// given manifest section to `A `, returning the modified section.
// Used to construct the "defective" historical view the original ACT
// exposes.
func flipStatusInManifest(section, path string, from, to byte) string {
	out := section
	oldLine := string(from) + "  " + path
	newLine := string(to) + "  " + path
	return strings.Replace(out, oldLine, newLine, -1)
}

// flipStatusInStats edits the statistics section to mirror a status
// letter flip. The defective view (the ACT's original defect) would
// say `M -> A` for `modified_files` and `added_files`.
func flipStatusInStats(section, kind string, oldCount, newCount int) string {
	out := section
	out = strings.Replace(out, kind+"="+itoaStr(oldCount), kind+"="+itoaStr(newCount), 1)
	out = strings.Replace(out, kind+"="+itoaStr(newCount), kind+"="+itoaStr(newCount), 1)
	return out
}

// itoaStr mirrors the digest package's intToString which avoids the
// `strconv` import for small integers. We can't import strconv here
// simply because this is a test and we want to keep the helper
// standalone; a tiny int→string table avoids the dependency.
func itoaStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// changedManifestRepo sets up a repo with a tracked file modified
// and staged (the original defect scenario) and returns the repoRoot.
func changedManifestRepo(t *testing.T) string {
	dir := t.TempDir()
	initGit(t, dir)

	writeRepoFile(t, dir, "tracked.go", "v1\n")
	runGit(t, dir, "add", "tracked.go")
	runGit(t, dir, "commit", "-m", "initial")
	writeRepoFile(t, dir, "tracked.go", "v1\nv2\n")
	runGit(t, dir, "add", "tracked.go")
	return dir
}

// TestEvidenceHashes_BindManifestAndStats proves that for the
// corrected manifest the relevant SHA-256 fields are populated, agree
// across runs, and are distinct (manifest != stats != aggregate).
func TestEvidenceHashes_BindManifestAndStats(t *testing.T) {
	dir := changedManifestRepo(t)

	out1, err := Generate(Options{RepoRoot: dir, Mode: ModeStaged})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	out2, err := Generate(Options{RepoRoot: dir, Mode: ModeStaged})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	fields := []string{
		"changeset_manifest_sha256",
		"changeset_stats_sha256",
		"review_map_sha256",
		"risk_signals_sha256",
		"file_evidence_sha256",
		"digest_evidence_sha256",
	}
	for _, f := range fields {
		got1 := evidenceHash(out1, f)
		got2 := evidenceHash(out2, f)
		if got1 == "" {
			t.Fatalf("first digest missing evidence field %q", f)
		}
		if got1 != got2 {
			t.Fatalf("evidence field %q not stable across runs: %q vs %q", f, got1, got2)
		}
		if len(got1) != 64 {
			t.Fatalf("evidence field %q is not 64 hex chars: %q", f, got1)
		}
	}
}

// TestEvidenceHashes_StatusReclassificationChangesHashes proves that
// a status flip (M→A in the manifest, modified_files→added_files in
// the stats) flows through to the manifest, statistics, and aggregate
// digest hashes. This mirrors the original defect: the digest
// misclassifies a tracked file's modification as an addition, and
// the (erroneously derived) stats report `added_files=1,
// modified_files=0` instead of the correct `added_files=0,
// modified_files=1`. The corrected digest must therefore differ in
// all three corresponding evidence hashes.
func TestEvidenceHashes_StatusReclassificationChangesHashes(t *testing.T) {
	dir := changedManifestRepo(t)

	out, err := Generate(Options{RepoRoot: dir, Mode: ModeStaged})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	manifest := extractSection(out, "CHANGESET_MANIFEST")
	stats := extractSection(out, "CHANGESET_STATS")
	if !strings.Contains(manifest, "M  tracked.go") {
		t.Fatalf("corrected manifest must contain 'M  tracked.go', got:\n%s", manifest)
	}

	// Defective view: tracked.go is reported as `A` in the manifest
	// and the corresponding stats line moves from `modified_files`
	// to `added_files`.
	defectiveManifest := flipStatusInManifest(manifest, "tracked.go", 'M', 'A')
	defectiveStats := flipStatusInStats(stats, "added_files", 0, 1)
	defectiveStats = strings.Replace(defectiveStats, "modified_files=1", "modified_files=0", 1)

	if strings.Contains(defectiveManifest, "M  tracked.go") {
		t.Fatal("defective manifest should no longer contain the correct M line")
	}
	if !strings.Contains(defectiveStats, "added_files=1") {
		t.Fatalf("defective stats should report added_files=1, got:\n%s", defectiveStats)
	}

	defective := rebuildEvidenceHashes(out, defectiveManifest, defectiveStats)

	// All three hashes (manifest, stats, aggregate) must change.
	fields := []string{
		"changeset_manifest_sha256",
		"changeset_stats_sha256",
		"digest_evidence_sha256",
	}
	for _, f := range fields {
		if evidenceHash(out, f) == evidenceHash(defective, f) {
			t.Errorf("field %q must change between corrected and defective digest", f)
		}
	}

}

// TestEvidenceHashes_UnrelatedSectionsStable verifies that re-running
// the digest on the same repository state yields identical hashes
// for sections that depend only on repository state.
func TestEvidenceHashes_UnrelatedSectionsStable(t *testing.T) {
	dir := changedManifestRepo(t)

	out1, err := Generate(Options{RepoRoot: dir, Mode: ModeStaged})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	out2, err := Generate(Options{RepoRoot: dir, Mode: ModeStaged})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	stableFields := []string{
		"changeset_manifest_sha256",
		"changeset_stats_sha256",
		"review_map_sha256",
		"risk_signals_sha256",
	}
	for _, f := range stableFields {
		if evidenceHash(out1, f) != evidenceHash(out2, f) {
			t.Errorf("repeated digest must produce stable %s: got %q vs %q",
				f, evidenceHash(out1, f), evidenceHash(out2, f))
		}
	}
}

// TestEvidenceHashes_NotHardcoded verifies that hashes reflect the
// repository content. Two repositories with different file contents
// must produce different file-evidence or aggregate hashes; they
// must not all collapse to a single hardcoded value.
func TestEvidenceHashes_NotHardcoded(t *testing.T) {
	mkFixture := func(content string) string {
		dir := t.TempDir()
		initGit(t, dir)
		writeRepoFile(t, dir, "tracked.go", "v1\n")
		runGit(t, dir, "add", "tracked.go")
		runGit(t, dir, "commit", "-m", "initial")
		writeRepoFile(t, dir, "tracked.go", content)
		runGit(t, dir, "add", "tracked.go")
		return dir
	}

	dirA := mkFixture("v1\nA\n")
	dirB := mkFixture("v1\nB\n")

	outA, err := Generate(Options{RepoRoot: dirA, Mode: ModeStaged})
	if err != nil {
		t.Fatalf("Generate A: %v", err)
	}
	outB, err := Generate(Options{RepoRoot: dirB, Mode: ModeStaged})
	if err != nil {
		t.Fatalf("Generate B: %v", err)
	}

	fields := []string{
		"file_evidence_sha256",
		"digest_evidence_sha256",
	}
	differing := 0
	for _, f := range fields {
		if evidenceHash(outA, f) != evidenceHash(outB, f) {
			differing++
		}
	}
	if differing == 0 {
		t.Fatalf("expected at least one per-content-derived hash to differ; got equal across both repos")
	}
}

// silenceUnusedImport keeps `os` and `path/filepath` available for
// future tests without producing lint warnings.
var _ = os.Stat
var _ = filepath.Join
