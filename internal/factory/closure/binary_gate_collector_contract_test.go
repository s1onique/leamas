// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"sync"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// TestClosureBinaryGateCollectorIdentityMismatch proves the full request
// identity contract at the narrow GateCollector boundary. A second capture
// with a different live subject root must be rejected without invoking the
// underlying bounded runner again.
func TestClosureBinaryGateCollectorIdentityMismatch(t *testing.T) {
	runner := &r6BCollectorCountingRunner{}
	collector := evidence.NewGateCollector(runner)
	first := evidence.GateCaptureRequest{
		RepositoryRoot: "/repo",
		SubjectRoot:    "/subject-a",
		EvidenceDir:    t.TempDir(),
		RunID:          "run-a",
		MakeExecutable: []string{"/binary", "factory", "gate", "--lane=fast"},
	}
	if _, err := collector.Capture(context.Background(), first); err != nil {
		t.Fatalf("first capture: %v", err)
	}

	second := first
	second.SubjectRoot = "/subject-b"
	if _, err := collector.Capture(context.Background(), second); !evidence.CollectorRequestMismatch(err) {
		t.Fatalf("second capture error = %v, want collector request mismatch", err)
	}
	if got := runner.Calls(); got != 1 {
		t.Fatalf("underlying runner calls = %d, want 1", got)
	}
	if got := collector.Calls(); got != 1 {
		t.Fatalf("collector calls = %d, want 1", got)
	}
}

type r6BCollectorCountingRunner struct {
	mu    sync.Mutex
	calls int
}

func (r *r6BCollectorCountingRunner) Run(
	context.Context,
	string,
	[]string,
	string,
	[]string,
) evidence.CommandResult {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return evidence.CommandResult{
		ExitCode: 0,
		Stdout:   []byte("EXEC_GATE_OBSERVED_STATUS:OK\n"),
	}
}

func (r *r6BCollectorCountingRunner) Calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}
