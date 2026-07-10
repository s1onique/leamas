// Package adapters provides typed adapters for the execution gateway.
package adapters

import (
	"testing"

	"github.com/s1onique/leamas/internal/execution"
)

func newTestExecutor() *execution.Executor {
	root := execution.NewTestExecutionRoot()
	budget := execution.DefaultBudget().WithMaxConcurrent(4)
	return execution.NewExecutor(budget, root)
}

func TestGoAdapterClampParallelism(t *testing.T) {
	executor := newTestExecutor()
	adapter := NewGoAdapter(executor)

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"no flags", []string{}, []string{}},
		{"low parallelism", []string{"-p", "2"}, []string{"-p", "2"}},
		{"high clamped", []string{"-p", "16"}, []string{"-p", "4"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.clampParallelism(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
			for i := range tt.expected {
				if result[i] != tt.expected[i] {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestGoAdapterTestFlags(t *testing.T) {
	executor := newTestExecutor()
	adapter := NewGoAdapter(executor)

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"no flags", []string{}, []string{"-timeout", "120s"}},
		{"existing timeout", []string{"-timeout", "60s"}, []string{"-timeout", "60s"}},
		{"high -p clamped", []string{"-p", "16"}, []string{"-p", "4", "-timeout", "120s"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.clampTestFlags(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
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
		{"no -j", []string{}, []string{}},
		{"low -jN", []string{"-j2"}, []string{"-j2"}},
		{"high clamped", []string{"-j16"}, []string{"-j4"}},
		{"unlimited", []string{"-j"}, []string{"-j4"}},
		{"-j0", []string{"-j0"}, []string{"-j4"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.clampJobs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
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
