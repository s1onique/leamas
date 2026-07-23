package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestClosureCLIEndToEndSubprocess(t *testing.T) {
	binary := buildLeamasForTest(t)
	repository, planPath, subject := prepareClosureCLIRepository(t)
	detached := t.TempDir()
	manifestOutput := filepath.Join(detached, "manifest.json")
	evidenceDirectory := filepath.Join(detached, "evidence")

	assertClosureSubprocess(t, binary, repository, "VALID", true,
		"factory", "close", "plan", "validate", "--file", planPath)

	stdout, stderr, runErr := runClosureSubprocess(binary, repository,
		"factory", "close", "run",
		"--plan", planPath,
		"--plan-freeze", subject+":docs/closure-plans/ACT-LEAMAS-CLI-SUBPROCESS01.json",
		"--subject", subject,
		"--evidence-dir", evidenceDirectory,
		"--manifest-out", manifestOutput)
	verdict := strings.TrimSpace(stdout)
	if verdict != "PASS" && verdict != "FAIL" {
		t.Fatalf("run verdict=%q stderr=%q err=%v", stdout, stderr, runErr)
	}
	if verdict == "PASS" && runErr != nil || verdict == "FAIL" && runErr == nil {
		t.Fatalf("run exit does not match verdict %s: %v", verdict, runErr)
	}

	assertClosureSubprocess(t, binary, repository, verdict, true,
		"factory", "close", "verify", "--manifest", manifestOutput)

	reportPath := filepath.Join(repository, "docs", "close-reports", "ACT-LEAMAS-CLI-SUBPROCESS01.md")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		t.Fatal(err)
	}
	assertClosureSubprocess(t, binary, repository, reportPath, true,
		"factory", "close", "render", "--manifest", manifestOutput, "--output", reportPath)

	manifestPath := filepath.Join(repository, "docs", "closure-manifests", "ACT-LEAMAS-CLI-SUBPROCESS01.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(manifestOutput)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	gitForClosureTest(t, repository, "add", "docs/closure-manifests", "docs/close-reports")
	gitForClosureTest(t, repository, "commit", "-m", "closure commit")

	tagName := "act/leamas-cli-subprocess01"
	_, tagStderr, tagErr := runClosureSubprocess(binary, repository,
		"factory", "close", "tag", "create",
		"--manifest", manifestPath,
		"--report", reportPath,
		"--tag", tagName,
		"--target", "HEAD")
	expectedState := "CLOSED_LOCAL"
	if verdict == "FAIL" {
		expectedState = "IMPLEMENTED"
		if tagErr == nil || !strings.Contains(tagStderr, "passing manifest") {
			t.Fatalf("failed manifest tag result err=%v stderr=%q", tagErr, tagStderr)
		}
	} else if tagErr != nil {
		t.Fatalf("tag create: %v stderr=%q", tagErr, tagStderr)
	}

	assertClosureSubprocess(t, binary, repository, expectedState, true,
		"factory", "close", "status",
		"--manifest", manifestPath,
		"--report", reportPath,
		"--tag", tagName)
}

func TestClosureCLIMissingSubcommand(t *testing.T) {
	binary := buildLeamasForTest(t)
	command := exec.Command(binary, "factory", "close")
	command.Env = withoutLeamasEnv()
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Run(); err == nil {
		t.Fatal("missing subcommand must fail")
	}
	if !strings.Contains(stderr.String(), "missing subcommand") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func prepareClosureCLIRepository(t *testing.T) (string, string, string) {
	t.Helper()
	repository := t.TempDir()
	gitForClosureTest(t, repository, "init", "-b", "main")
	gitForClosureTest(t, repository, "config", "user.name", "Closure CLI Test")
	gitForClosureTest(t, repository, "config", "user.email", "closure-cli@example.invalid")
	if err := os.WriteFile(filepath.Join(repository, "README.md"), []byte("subject baseline\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitForClosureTest(t, repository, "add", "README.md")
	gitForClosureTest(t, repository, "commit", "-m", "baseline")
	baseline := gitForClosureTest(t, repository, "rev-parse", "HEAD")
	baselineTree := gitForClosureTest(t, repository, "rev-parse", "HEAD^{tree}")
	planPath := filepath.Join(repository, "docs", "closure-plans", "ACT-LEAMAS-CLI-SUBPROCESS01.json")
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		t.Fatal(err)
	}
	plan := fmt.Sprintf(`{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-CLI-SUBPROCESS01",
  "baseline": {"commit_oid": %q, "tree_oid": %q},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {
      "id": "go-version",
      "mode": "run",
      "argv": ["go", "version"],
      "working_directory": ".",
      "timeout_seconds": 60,
      "environment": {}
    }
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}
`, baseline, baselineTree)
	if err := os.WriteFile(planPath, []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
	gitForClosureTest(t, repository, "add", "docs/closure-plans")
	gitForClosureTest(t, repository, "commit", "-m", "subject with frozen plan")
	subject := gitForClosureTest(t, repository, "rev-parse", "HEAD")
	return repository, planPath, subject
}

func gitForClosureTest(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func runClosureSubprocess(binary, directory string, args ...string) (string, string, error) {
	command := exec.Command(binary, args...)
	command.Dir = directory
	command.Env = withoutLeamasEnv()
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return stdout.String(), stderr.String(), err
}

func assertClosureSubprocess(t *testing.T, binary, directory, expected string, expectSuccess bool, args ...string) {
	t.Helper()
	stdout, stderr, err := runClosureSubprocess(binary, directory, args...)
	if expectSuccess && err != nil {
		t.Fatalf("%v: %v stderr=%q", args, err, stderr)
	}
	if strings.TrimSpace(stdout) != expected {
		t.Fatalf("%v stdout=%q, want %q; stderr=%q", args, stdout, expected, stderr)
	}
}
