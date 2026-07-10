// Package adapters provides typed adapters for the execution gateway.
package adapters

import (
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
)

func newTestExecutor() *execution.Executor {
	root := execution.NewTestExecutionRoot()
	budget := execution.DefaultBudget().WithMaxConcurrent(4)
	executor, _ := execution.NewExecutor(budget, root)
	return executor
}

func TestGoAdapterClampParallelism(t *testing.T) {
	executor := newTestExecutor()
	adapter := NewGoAdapter(executor)

	// Test with empty input
	result := adapter.clampParallelism([]string{}, "-p")
	// Just verify it doesn't panic

	// Test with parallelism
	result = adapter.clampParallelism([]string{"-p", "2"}, "-p")
	if len(result) == 0 {
		t.Error("expected result with -p flag")
	}
}

func TestGoAdapterTestFlags(t *testing.T) {
	executor := newTestExecutor()
	adapter := NewGoAdapter(executor)

	tests := []struct {
		name  string
		input []string
	}{
		{"no flags", []string{}},
		{"existing timeout", []string{"-timeout", "60s"}},
		{"high -p clamped", []string{"-p", "16"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.clampTestFlags(tt.input)
			// Verify result has -timeout
			hasTimeout := false
			for _, arg := range result {
				if arg == "-timeout" || strings.HasPrefix(arg, "-timeout=") {
					hasTimeout = true
					break
				}
			}
			if !hasTimeout {
				t.Errorf("expected -timeout in result, got %v", result)
			}
		})
	}
}

func TestMakeAdapterClampJobs(t *testing.T) {
	executor := newTestExecutor()
	adapter := NewMakeAdapter(executor)

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		// No -j adds default
		{"no -j adds default", []string{}, []string{"-j4"}},
		// Short form -jN
		{"-j bare", []string{"-j"}, []string{"-j4"}},
		{"-j0 clamped", []string{"-j0"}, []string{"-j4"}},
		{"-j2 kept", []string{"-j2"}, []string{"-j2"}},
		{"-j32 clamped", []string{"-j32"}, []string{"-j4"}},
		// Short form -j=N
		{"-j=32 clamped", []string{"-j=32"}, []string{"-j4"}},
		// Short form spaced -j N (collapses to single arg -jN)
		{"-j 2 kept", []string{"-j", "2"}, []string{"-j2"}},
		{"-j 32 clamped", []string{"-j", "32"}, []string{"-j4"}},
		// Long form --jobs=N
		{"--jobs bare", []string{"--jobs"}, []string{"--jobs=4"}},
		{"--jobs=2 kept", []string{"--jobs=2"}, []string{"--jobs=2"}},
		{"--jobs=32 clamped", []string{"--jobs=32"}, []string{"--jobs=4"}},
		// Long form spaced --jobs N
		{"--jobs 2 kept", []string{"--jobs", "2"}, []string{"--jobs=2"}},
		{"--jobs 32 clamped", []string{"--jobs", "32"}, []string{"--jobs=4"}},
		// Unrelated flags preserved
		{"--jobserver-auth preserved", []string{"--jobserver-auth=fifo:/tmp/x"}, []string{"--jobserver-auth=fifo:/tmp/x", "-j4"}},
		{"--no-print-directory preserved", []string{"--no-print-directory"}, []string{"--no-print-directory", "-j4"}},
		{"--output-sync preserved", []string{"--output-sync=target"}, []string{"--output-sync=target", "-j4"}},
		// Mixed with unrelated flags
		{"-j4 with other flags", []string{"--no-print-directory", "-j4", "all"}, []string{"--no-print-directory", "-j4", "all"}},
		{"-j32 with --jobserver-auth", []string{"-j32", "--jobserver-auth=fifo:/tmp/x"}, []string{"-j4", "--jobserver-auth=fifo:/tmp/x"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.clampJobs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("expected[%d]=%q, got[%d]=%q", i, tt.expected[i], i, result[i])
				}
			}
		})
	}
}

func TestMakeAdapterClampJobsInString(t *testing.T) {
	executor := newTestExecutor()
	adapter := NewMakeAdapter(executor)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// No -j adds default
		{"empty string", "", ""},
		// Short form -jN
		{"-j bare", "-j", "-j4"},
		{"-j0 clamped", "-j0", "-j4"},
		{"-j2 kept", "-j2", "-j2"},
		{"-j32 clamped", "-j32", "-j4"},
		// Short form -j=N
		{"-j=32 clamped", "-j=32", "-j4"},
		// Short form spaced -j N (rare in MAKEFLAGS but supported)
		{"-j 2 kept", "-j 2", "-j2"},
		{"-j 32 clamped", "-j 32", "-j4"},
		// Long form --jobs=N
		{"--jobs bare", "--jobs", "--jobs=4"},
		{"--jobs=2 kept", "--jobs=2", "--jobs=2"},
		{"--jobs=32 clamped", "--jobs=32", "--jobs=4"},
		// Long form spaced --jobs N
		{"--jobs 2 kept", "--jobs 2", "--jobs=2"},
		{"--jobs 32 clamped", "--jobs 32", "--jobs=4"},
		// CRITICAL: --jobserver-auth must NOT be corrupted
		{"--jobserver-auth preserved", "--jobserver-auth=fifo:/tmp/x", "--jobserver-auth=fifo:/tmp/x"},
		{"--jobserver-auth with -j", "-j4 --jobserver-auth=fifo:/tmp/x", "-j4 --jobserver-auth=fifo:/tmp/x"},
		// CRITICAL: other --options must be preserved
		{"--no-print-directory preserved", "--no-print-directory", "--no-print-directory"},
		{"--output-sync preserved", "--output-sync=target", "--output-sync=target"},
		// Mixed flags
		{"full MAKEFLAGS example", "--no-print-directory -j32 --jobserver-auth=fifo:/tmp/x", "--no-print-directory -j4 --jobserver-auth=fifo:/tmp/x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.clampJobsInString(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMakeAdapterHasUnboundedJobs(t *testing.T) {
	executor := newTestExecutor()
	adapter := NewMakeAdapter(executor)

	tests := []struct {
		name     string
		input    []string
		expected bool
	}{
		{"no -j", []string{}, false},
		{"-j with number", []string{"-j4"}, false},
		{"unlimited -j", []string{"-j"}, true},
		{"-j0", []string{"-j0"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.HasUnboundedJobs(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGoAdapterRequestCreation(t *testing.T) {
	executor := newTestExecutor()
	adapter := NewGoAdapter(executor)

	req := adapter.BuildRequest("/project", "bin/out", []string{"./cmd/..."})
	if req.Args[0] != "go" || req.Args[1] != "build" {
		t.Errorf("unexpected args: %v", req.Args)
	}

	req = adapter.TestRequest("/project", []string{"./..."})
	if req.Args[0] != "go" || req.Args[1] != "test" {
		t.Errorf("unexpected args: %v", req.Args)
	}
}

func TestMakeAdapterRequestCreation(t *testing.T) {
	executor := newTestExecutor()
	adapter := NewMakeAdapter(executor)

	// Test GateRequest with exact argument assertions
	req := adapter.GateRequest("/project", nil)
	want := []string{"make", "-j4", "gate"}
	if len(req.Args) != len(want) {
		t.Errorf("GateRequest args: expected %v, got %v", want, req.Args)
	}
	for i := range want {
		if req.Args[i] != want[i] {
			t.Errorf("GateRequest args[%d]: expected %q, got %q", i, want[i], req.Args[i])
		}
	}

	// Test FactorizeRequest with exact argument assertions
	req = adapter.FactorizeRequest("/project", nil)
	want = []string{"make", "-j4", "factorize"}
	if len(req.Args) != len(want) {
		t.Errorf("FactorizeRequest args: expected %v, got %v", want, req.Args)
	}
	for i := range want {
		if req.Args[i] != want[i] {
			t.Errorf("FactorizeRequest args[%d]: expected %q, got %q", i, want[i], req.Args[i])
		}
	}
}

func TestGitAdapterRequestCreation(t *testing.T) {
	executor := newTestExecutor()
	adapter := NewGitAdapter(executor)

	req := adapter.Status("/project")
	if req.Args[0] != "git" || req.Args[1] != "status" {
		t.Errorf("unexpected args: %v", req.Args)
	}

	req = adapter.Diff("/project", "-M")
	if req.Args[0] != "git" || req.Args[1] != "diff" {
		t.Errorf("unexpected args: %v", req.Args)
	}
}
