package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution"
	"github.com/s1onique/leamas/internal/factory/closure"
)

const cliV2SubjectExactActID = "ACT-LEAMAS-CLI-V2-SUBJECT-EXACT01"

func TestClosureCLIV2SubjectExactSuccessAndIdempotence(t *testing.T) {
	repository := copyCurrentLeamasSource(t)
	planPath, freeze, subject := prepareSubjectExactV2History(t, repository)
	binary := buildSubjectExactLeamas(t, repository)
	assertBinarySubjectExact(t, binary, subject)

	first := runSuccessfulV2CLI(t, binary, repository, planPath, subject)
	assertSuccessfulV2CLIResult(t, repository, freeze, subject, first)
	before := captureCLIV2State(t, repository, first)

	second := runSuccessfulV2CLI(t, binary, repository, planPath, subject)
	assertSuccessfulV2CLIResult(t, repository, freeze, subject, second)
	after := captureCLIV2State(t, repository, second)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("idempotent result changed:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("second CLI invocation mutated closure:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func runSuccessfulV2CLI(t *testing.T, binary, repository, planPath, subject string) closure.TransactionResult {
	t.Helper()
	stdout, stderr, err := runClosureSubprocess(binary, repository,
		"factory", "close", "run", "--protocol", "v2", "--json",
		"--plan", planPath, "--subject", subject)
	if err != nil {
		t.Fatalf("subject-exact v2 run: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stderr, "Closure Protocol v2:") {
		t.Fatalf("missing v2 success summary: %q", stderr)
	}
	var result closure.TransactionResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode v2 JSON: %v\n%s", err, stdout)
	}
	return result
}

func assertSuccessfulV2CLIResult(t *testing.T, repository, freeze, subject string, result closure.TransactionResult) {
	t.Helper()
	if result.ActID != cliV2SubjectExactActID || result.FreezeCommit != freeze ||
		result.SubjectCommit != subject || result.ClosureCommit == "" || result.ClosureTree == "" ||
		result.TagObject == "" || result.TagTarget != result.ClosureCommit ||
		result.EvidenceHash == "" || result.Verdict != closure.VerdictPass {
		t.Fatalf("incomplete v2 CLI result: %+v", result)
	}
	if got := gitForClosureTest(t, repository, "rev-parse", "refs/heads/main"); got != result.ClosureCommit {
		t.Fatalf("branch = %s, want C %s", got, result.ClosureCommit)
	}
	if got := gitForClosureTest(t, repository, "rev-parse", "refs/tags/"+result.TagName+"^{tag}"); got != result.TagObject {
		t.Fatalf("tag object = %s, want T %s", got, result.TagObject)
	}
	if status := gitForClosureTest(t, repository, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
		t.Fatalf("successful closure left dirty worktree: %q", status)
	}
}

func copyCurrentLeamasSource(t *testing.T) string {
	t.Helper()
	source := gitForClosureTest(t, ".", "rev-parse", "--show-toplevel")
	result, err := execution.RunGit(t.Context(), source, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("list source files: %v (exit %d): %s", err, result.ExitCode, result.Stderr)
	}
	listed := result.Stdout
	destination := t.TempDir()
	for _, raw := range bytes.Split(listed, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		relative := string(raw)
		from := filepath.Join(source, filepath.FromSlash(relative))
		to := filepath.Join(destination, filepath.FromSlash(relative))
		info, err := os.Lstat(from)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(from)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, to); err != nil {
				t.Fatal(err)
			}
			continue
		}
		data, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(to, data, info.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
	gitForClosureTest(t, destination, "init", "-b", "main")
	gitForClosureTest(t, destination, "config", "user.name", "CLI V2 Subject Exact")
	gitForClosureTest(t, destination, "config", "user.email", "cli-v2@example.invalid")
	gitForClosureTest(t, destination, "add", "-A")
	gitForClosureTest(t, destination, "commit", "-m", "source baseline")
	return destination
}

func prepareSubjectExactV2History(t *testing.T, repository string) (string, string, string) {
	t.Helper()
	baseline := gitForClosureTest(t, repository, "rev-parse", "HEAD")
	baselineTree := gitForClosureTest(t, repository, "rev-parse", "HEAD^{tree}")
	planPath := filepath.Join(repository, "docs", "closure-plans", cliV2SubjectExactActID+".json")
	plan := fmt.Sprintf(`{
  "contract_version": 1,
  "act_id": %q,
  "baseline": {"commit_oid": %q, "tree_oid": %q},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id":"go-version","mode":"run","argv":["go","version"],"working_directory":".","timeout_seconds":60,"environment":{"CGO_ENABLED":"0"}}],
  "artifacts": [],
  "policy": {"require_clean_before":true,"require_clean_after":true,"forbid_tracked_full_digests":true,"require_diff_check":true}
}
`, cliV2SubjectExactActID, baseline, baselineTree)
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
	gitForClosureTest(t, repository, "add", "docs/closure-plans")
	gitForClosureTest(t, repository, "commit", "-m", "freeze v2 plan")
	freeze := gitForClosureTest(t, repository, "rev-parse", "HEAD")
	gitForClosureTest(t, repository, "commit", "--allow-empty", "-m", "subject")
	return planPath, freeze, gitForClosureTest(t, repository, "rev-parse", "HEAD")
}

func buildSubjectExactLeamas(t *testing.T, repository string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "leamas-subject-exact")
	runBoundedV2TestCommand(t, repository, []string{"CGO_ENABLED=0"},
		"go", "build", "-trimpath", "-o", binary, "./cmd/leamas")
	return binary
}

func assertBinarySubjectExact(t *testing.T, binary, subject string) {
	t.Helper()
	metadata := string(runBoundedV2TestCommand(t, ".", nil, "go", "version", "-m", binary))
	if !strings.Contains(metadata, "vcs.revision="+subject) || !strings.Contains(metadata, "vcs.modified=false") {
		t.Fatalf("binary is not subject-exact:\n%s", metadata)
	}
}

func runBoundedV2TestCommand(t *testing.T, directory string, environment []string, args ...string) []byte {
	t.Helper()
	budget := execution.DefaultBudget().WithTimeout(2 * time.Minute).WithMaxConcurrent(1).WithMaxStarts(1)
	executor, err := execution.NewExecutor(budget, execution.NewTestExecutionRoot())
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	result := executor.Execute(t.Context(), &execution.Request{
		Name: "v2 subject-exact test command", Args: args, Dir: directory,
		Env: environment, Timeout: 2 * time.Minute,
	})
	if !result.Success() {
		t.Fatalf("command %v failed (exit %d, err=%v):\n%s", args, result.ExitCode, result.Error, result.Stderr)
	}
	return result.Stdout
}

type cliV2State struct {
	refs, status, head, indexTree, objects string
	files                                  map[string]string
}

func captureCLIV2State(t *testing.T, repository string, result closure.TransactionResult) cliV2State {
	t.Helper()
	state := cliV2State{
		refs:      gitForClosureTest(t, repository, "for-each-ref", "--format=%(refname)%00%(objectname)"),
		status:    gitForClosureTest(t, repository, "status", "--porcelain=v1", "--untracked-files=all"),
		head:      gitForClosureTest(t, repository, "rev-parse", "HEAD"),
		indexTree: gitForClosureTest(t, repository, "write-tree"),
		objects:   gitForClosureTest(t, repository, "cat-file", "--batch-all-objects", "--batch-check=%(objectname)"),
		files:     make(map[string]string),
	}
	roots := []string{
		result.EvidencePath,
		filepath.Join(repository, "docs", "closure-manifests", cliV2SubjectExactActID+".json"),
		filepath.Join(repository, "docs", "close-reports", cliV2SubjectExactActID+".md"),
	}
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(repository, path)
			if err != nil {
				return err
			}
			state.files[relative] = fmt.Sprintf("%o:%d:%s", info.Mode().Perm(), info.ModTime().UnixNano(), data)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return state
}
