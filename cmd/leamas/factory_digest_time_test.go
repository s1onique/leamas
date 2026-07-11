// Package main provides elapsed-time tests for the factory digest command.
package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution"
	"github.com/s1onique/leamas/internal/factory/digest"
)

var digestStatusPattern = regexp.MustCompile(
	`^digest: mode=\S+ output=.+ time=\d+\.\d{2}s OK\n$`,
)

var digestTimingField = regexp.MustCompile(
	`(?:^|[ \n])time=\d+\.\d{2}s(?:[ \n]|$)`,
)

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{0, "0.00s"},
		{5 * time.Millisecond, "0.01s"},
		{time.Second + 5*time.Millisecond, "1.00s"},
		{time.Second + 6*time.Millisecond, "1.01s"},
		{121*time.Second + 424*time.Millisecond, "121.42s"},
	}

	for _, tt := range tests {
		if got := formatElapsed(tt.duration); got != tt.want {
			t.Errorf("formatElapsed(%s) = %q, want %q", tt.duration, got, tt.want)
		}
	}
}

func TestRunFactoryDigest_SuccessOutputIncludesElapsedTime(t *testing.T) {
	var captured digest.Options
	fakeWrite := fakeWriteDigest(&captured, nil)

	var stdout, stderr bytes.Buffer
	code := runFactoryDigest([]string{"--output", "/tmp/d.md"}, &stdout, &stderr, fakeWrite)

	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !digestStatusPattern.MatchString(stdout.String()) {
		t.Fatalf("stdout %q does not match %q", stdout.String(), digestStatusPattern)
	}
}

func TestRunFactoryDigest_ElapsedTimeIncludesSuccessfulWrite(t *testing.T) {
	writeComplete := false
	sinceCalled := false
	start := time.Unix(100, 0)

	writeDigest := func(opts digest.Options) error {
		if sinceCalled {
			t.Fatal("elapsed time was measured before digest writing")
		}
		writeComplete = true
		return nil
	}
	now := func() time.Time {
		return start
	}
	since := func(gotStart time.Time) time.Duration {
		sinceCalled = true
		if !writeComplete {
			t.Fatal("elapsed time was measured before digest writing completed")
		}
		if !gotStart.Equal(start) {
			t.Fatalf("start time = %s, want %s", gotStart, start)
		}
		return time.Second + 6*time.Millisecond
	}

	var stdout, stderr bytes.Buffer
	code := runFactoryDigestWithClock(
		[]string{"--output", "/tmp/d.md"},
		&stdout,
		&stderr,
		writeDigest,
		now,
		since,
	)

	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !sinceCalled {
		t.Fatal("expected elapsed time to be measured")
	}
	if !strings.Contains(stdout.String(), " time=1.01s OK\n") {
		t.Fatalf("stdout should include deterministic elapsed time: %q", stdout.String())
	}
}

func TestRunFactoryDigest_ProductionDigestFileExcludesElapsedTime(t *testing.T) {
	repoRoot := t.TempDir()
	initDigestGitRepo(t, repoRoot)

	trackedFile := filepath.Join(repoRoot, "tracked.txt")
	if err := os.WriteFile(trackedFile, []byte("initial\n"), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	runDigestGit(t, repoRoot, "add", "tracked.txt")
	runDigestGit(t, repoRoot, "commit", "-m", "initial")

	if err := os.WriteFile(trackedFile, []byte("initial\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify tracked file: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("failed to enter repo: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	outputPath := filepath.Join(t.TempDir(), "digest.txt")
	var stdout, stderr bytes.Buffer
	code := runFactoryDigest(
		[]string{"--dirty", "--output", outputPath},
		&stdout,
		&stderr,
		digest.Write,
	)

	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !digestStatusPattern.MatchString(stdout.String()) {
		t.Fatalf("stdout %q does not match %q", stdout.String(), digestStatusPattern)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read digest file: %v", err)
	}
	if !bytes.Contains(data, []byte("# Targeted digest")) {
		t.Fatalf("digest file does not look like production digest output")
	}
	if digestTimingField.Match(data) {
		t.Fatalf("digest contains CLI timing field")
	}
}

func initDigestGitRepo(t *testing.T, dir string) {
	t.Helper()
	runDigestGit(t, dir, "init")
	runDigestGit(t, dir, "config", "user.email", "test@example.com")
	runDigestGit(t, dir, "config", "user.name", "Test User")
	runDigestGit(t, dir, "config", "commit.gpgsign", "false")
}

func runDigestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	budget := execution.DefaultBudget().WithTimeout(30 * time.Second)
	budget.MaxStarts = 1
	executor, err := execution.NewExecutor(budget, execution.NewTestExecutionRoot())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	reqArgs := append([]string{"git"}, args...)
	result := executor.Execute(context.Background(), &execution.Request{
		Name:      "git " + strings.Join(args, " "),
		Args:      reqArgs,
		Dir:       dir,
		OutputCap: 1024 * 1024,
	})
	if !result.Success() {
		t.Fatalf(
			"git %v failed: exit=%d err=%v\nstdout:\n%s\nstderr:\n%s",
			args,
			result.ExitCode,
			result.Error,
			string(result.Stdout),
			string(result.Stderr),
		)
	}
}
