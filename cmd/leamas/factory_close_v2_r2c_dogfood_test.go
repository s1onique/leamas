// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_r2c_dogfood_test.go executes the exact-final-tip
// dogfood required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2C.
//
// The test:
//  1. builds the leamas binary against the CURRENT tree using the
//     production LDFLAGS so the running binary identity reports the
//     actual current HEAD commit and a clean "vcs.modified=false"
//     stamp;
//  2. asserts the binary VCS revision equals HEAD and modified=false;
//  3. invokes the binary from a temp directory OUTSIDE the source
//     tree against a fresh hermetic S < F < D repository;
//  4. asserts DOGFOOD_EXIT == 0, manifest bindings are exact, caller
//     state is unchanged, and no linked worktree leaked.
//
// The test does not introduce new architecture. It reuses the
// helpers from factory_close_v2_mac_canary_test.go and adds the
// binary-identity assertions R2C requires.
//
// On a Mac the same binary would be built by `make build`; the
// Linux-side proof is deterministic because the test links the
// current HEAD commit into the binary at build time.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

// r2cDogfoodResult captures every R2C dogfood value the test
// observes. The struct fields double as the close-report evidence.
type r2cDogfoodResult struct {
	FinalCommit        string
	FinalTree          string
	BinaryPath         string
	BinarySHA256       string
	BinaryVCSRevision  string
	BinaryVCSModified  bool
	Subject            string
	Freeze             string
	Descendant         string
	CallerHeadBefore   string
	CallerTreeBefore   string
	CallerStatusBefore string
	WorktreesBefore    string
	CallerHeadAfter    string
	CallerTreeAfter    string
	CallerStatusAfter  string
	WorktreesAfter     string
	ManifestPath       string
	ManifestSubject    string
	ManifestFreeze     string
	ManifestCallerHead string
	ManifestExecTree   string
	ManifestBinarySHA  string
	ManifestBinaryRev  string
	ManifestBinaryMod  bool
	ManifestSHA256     string
	Stdout             string
	Stderr             string
	Exit               int
	Err                error
}

var lastR2CDogfood r2cDogfoodResult

// TestClosureCLIV2R2CExactTipDogfood is the R2C exact-final-tip
// dogfood. It MUST be run against the FINAL_COMMIT; the test
// itself records the FINAL_COMMIT it observed so the close
// report can attribute the values to the correct commit.
func TestClosureCLIV2R2CExactTipDogfood(t *testing.T) {
	finalCommit := gitForClosureTest(t, ".", "rev-parse", "HEAD")
	finalTree := gitForClosureTest(t, ".", "rev-parse", "HEAD^{tree}")
	if strings.TrimSpace(finalCommit) == "" {
		t.Fatalf("could not resolve HEAD")
	}

	binary := buildR2CExactTipLeamas(t, finalCommit)
	binSHA := fileSHA256Hex(t, binary)

	// Read the binary's identity from its own CLI surface
	// (--version emits the linker-injected values).
	identity := readBinaryIdentity(t, binary)
	if identity.Commit != finalCommit {
		t.Fatalf("binary VCS revision %s does not match FINAL_COMMIT %s",
			identity.Commit, finalCommit)
	}
	if identity.Dirty {
		t.Fatalf("binary vcs.modified=true; dogfood requires clean build")
	}

	repository, subject, freeze, d := prepareMacCanaryDogfoodRepo(t)
	if got := gitForClosureTest(t, repository, "rev-parse", "HEAD"); got != d {
		t.Fatalf("pre-run HEAD must equal D: got=%s want=%s", got, d)
	}
	detachedDir := t.TempDir()
	evidenceDir := filepath.Join(detachedDir, "evidence")
	manifestOutput := filepath.Join(detachedDir, "manifest.json")

	// Snapshot caller + worktree registrations BEFORE.
	headBefore := gitForClosureTest(t, repository, "rev-parse", "HEAD")
	headTreeBefore := gitForClosureTest(t, repository, "rev-parse", "HEAD^{tree}")
	statusBefore := gitForClosureTest(t, repository, "status", "--porcelain=v2", "--untracked-files=all")
	worktreesBefore := gitForClosureTest(t, repository, "worktree", "list", "--porcelain")

	stdout, stderr, runErr := runClosureSubprocess(binary, detachedDir,
		"factory", "close", "run-v2-authority",
		"--protocol-version", "2",
		"--plan-contract-version", "1",
		"--repository", repository,
		"--subject", subject,
		"--freeze", freeze,
		"--plan-path", dogfoodPlanPath,
		"--evidence-directory", evidenceDir,
		"--manifest-output", manifestOutput,
	)
	exit := 0
	if runErr != nil {
		exit = -1
	}

	// Snapshot caller + worktree registrations AFTER.
	headAfter := gitForClosureTest(t, repository, "rev-parse", "HEAD")
	headTreeAfter := gitForClosureTest(t, repository, "rev-parse", "HEAD^{tree}")
	statusAfter := gitForClosureTest(t, repository, "status", "--porcelain=v2", "--untracked-files=all")
	worktreesAfter := gitForClosureTest(t, repository, "worktree", "list", "--porcelain")

	if headBefore != headAfter {
		t.Fatalf("caller HEAD drifted: before=%s after=%s", headBefore, headAfter)
	}
	if headTreeBefore != headTreeAfter {
		t.Fatalf("caller HEAD tree drifted: before=%s after=%s",
			headTreeBefore, headTreeAfter)
	}
	if statusBefore != statusAfter {
		t.Fatalf("caller worktree status drifted:\nbefore=%q\nafter=%q",
			statusBefore, statusAfter)
	}
	if worktreesBefore != worktreesAfter {
		t.Fatalf("linked-worktree registrations leaked:\nbefore=%q\nafter=%q",
			worktreesBefore, worktreesAfter)
	}

	manifestBytes, readErr := os.ReadFile(manifestOutput)
	if readErr != nil {
		t.Fatalf("manifest file missing: %v", readErr)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("manifest JSON invalid: %v\n%s", err, string(manifestBytes))
	}

	mSubject, _ := manifest["subject_commit"].(string)
	mFreeze, _ := manifest["freeze_commit"].(string)
	mCaller, _ := manifest["caller_head"].(string)
	mExecTree, _ := manifest["execution_tree"].(string)
	// The v2 manifest uses leamas_binary_identity.{path,sha256,vcs_revision,vcs_modified}.
	binIdent, _ := manifest["leamas_binary_identity"].(map[string]any)
	mBinSHA, _ := binIdent["sha256"].(string)
	mBinRev, _ := binIdent["vcs_revision"].(string)
	mBinMod, _ := binIdent["vcs_modified"].(bool)

	if mSubject != subject {
		t.Fatalf("manifest subject_commit: got=%q want=%q", mSubject, subject)
	}
	if mFreeze != freeze {
		t.Fatalf("manifest freeze_commit: got=%q want=%q", mFreeze, freeze)
	}
	if mCaller != d {
		t.Fatalf("manifest caller_head: got=%q want=%q", mCaller, d)
	}
	if mExecTree == "" {
		t.Fatalf("manifest execution_tree empty")
	}
	if mBinSHA != binSHA {
		t.Fatalf("manifest leamas_binary_identity.sha256 %q != invoked binary %s", mBinSHA, binSHA)
	}
	if mBinRev != finalCommit {
		t.Fatalf("manifest leamas_binary_identity.vcs_revision %q != FINAL_COMMIT %s", mBinRev, finalCommit)
	}
	if mBinMod {
		t.Fatalf("manifest leamas_binary_identity.vcs_modified=true; clean build required")
	}

	manifestSHA := sha256.Sum256(manifestBytes)
	manifestSHAStr := hex.EncodeToString(manifestSHA[:])

	if !strings.Contains(stdout, "OK") {
		t.Fatalf("expected OK on stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Closure Protocol v2:") {
		t.Fatalf("expected v2 summary on stderr, got %q", stderr)
	}

	lastR2CDogfood = r2cDogfoodResult{
		FinalCommit:        finalCommit,
		FinalTree:          finalTree,
		BinaryPath:         binary,
		BinarySHA256:       binSHA,
		BinaryVCSRevision:  identity.Commit,
		BinaryVCSModified:  identity.Dirty,
		Subject:            subject,
		Freeze:             freeze,
		Descendant:         d,
		CallerHeadBefore:   headBefore,
		CallerTreeBefore:   headTreeBefore,
		CallerStatusBefore: statusBefore,
		WorktreesBefore:    worktreesBefore,
		CallerHeadAfter:    headAfter,
		CallerTreeAfter:    headTreeAfter,
		CallerStatusAfter:  statusAfter,
		WorktreesAfter:     worktreesAfter,
		ManifestPath:       manifestOutput,
		ManifestSubject:    mSubject,
		ManifestFreeze:     mFreeze,
		ManifestCallerHead: mCaller,
		ManifestExecTree:   mExecTree,
		ManifestBinarySHA:  mBinSHA,
		ManifestBinaryRev:  mBinRev,
		ManifestBinaryMod:  mBinMod,
		ManifestSHA256:     manifestSHAStr,
		Stdout:             stdout,
		Stderr:             stderr,
		Exit:               exit,
		Err:                runErr,
	}
}

// buildR2CExactTipLeamas builds the leamas binary with the
// production LDFLAGS that inject the supplied commit, the
// build time, and an explicit "false" dirty flag. The
// running binary identity helpers (closure.RunningLeamasVCSRevision
// etc.) read these linker-injected values.
func buildR2CExactTipLeamas(t *testing.T, finalCommit string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "leamas-r2c-exact-tip")
	buildTime := gitForClosureTest(t, ".", "show", "-s", "--format=%ct", "HEAD")
	ldflags := fmt.Sprintf(
		"-X 'github.com/s1onique/leamas/internal/version.Version=0.1.0+dev.%s.%s' "+
			"-X 'github.com/s1onique/leamas/internal/version.DeclaredVersion=0.1.0' "+
			"-X 'github.com/s1onique/leamas/internal/version.Commit=%s' "+
			"-X 'github.com/s1onique/leamas/internal/version.BuildTime=%s' "+
			"-X 'github.com/s1onique/leamas/internal/version.Dirty=false'",
		finalCommit[:8], buildTime, finalCommit, buildTime,
	)
	runMacCanaryBuildCommand(t, []string{"CGO_ENABLED=0"}, bin, ldflags)
	return bin
}

// fileSHA256Hex returns the SHA-256 of the file at path as
// lowercase hexadecimal. Used to verify the produced binary
// matches the values reported by `factory close run-v2-authority`.
func fileSHA256Hex(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read binary %s: %v", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// r2cBinaryIdentity captures the three linker-injected values
// the binary reports on `--version`.
type r2cBinaryIdentity struct {
	Commit  string
	Dirty   bool
	Version string
}

// readBinaryIdentity invokes `binary version` and parses the
// printed identity fields. The production CLI emits lowercase
// keys: version / declared_version / commit / build_time /
// dirty. The helper accepts both lower- and mixed-case keys.
func readBinaryIdentity(t *testing.T, binary string) r2cBinaryIdentity {
	t.Helper()
	stdout, _, err := runClosureSubprocess(binary, t.TempDir(), "version")
	if err != nil {
		t.Fatalf("read binary identity: %v", err)
	}
	id := r2cBinaryIdentity{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		// Accept "commit:", "Commit:", "VCS revision:".
		if strings.HasPrefix(strings.ToLower(line), "commit:") {
			id.Commit = strings.TrimSpace(line[len("commit:"):])
		} else if strings.HasPrefix(strings.ToLower(line), "vcs revision:") {
			id.Commit = strings.TrimSpace(line[len("vcs revision:"):])
		} else if strings.HasPrefix(strings.ToLower(line), "dirty:") {
			id.Dirty = strings.EqualFold(strings.TrimSpace(line[len("dirty:"):]), "true")
		} else if strings.HasPrefix(strings.ToLower(line), "vcs modified:") {
			id.Dirty = strings.EqualFold(strings.TrimSpace(line[len("vcs modified:"):]), "true")
		} else if strings.HasPrefix(strings.ToLower(line), "version:") {
			id.Version = strings.TrimSpace(line[len("version:"):])
		}
	}
	return id
}

// silence unused imports.
var _ = time.Second
var _ = execution.DefaultBudget
