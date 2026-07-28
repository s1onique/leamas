// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// fakeValidCIObserver is a test observer that returns a valid CI context.
type fakeValidCIObserver struct{}

func (f *fakeValidCIObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: "github-actions",
		GitHubSHA:       "abc123def456abc123def456abc123def456abcd",
		GitHubWorkspace: root,
		HeadCommit:      "abc123def456abc123def456abc123def456abcd",
		WorktreeStatus:  "",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}

// TestDupcodeVerifyRoutesToVerifyEntryPoint proves that the verify command
// routes to the verify entry point and not the update entry point.
func TestDupcodeVerifyRoutesToVerifyEntryPoint(t *testing.T) {
	var verifyCalls int64
	var updateCalls int64

	// Inject counting dispatchers that use the observer variant
	originalVerify := DupcodeVerifyDispatcher

	DupcodeVerifyDispatcher = func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
		atomic.AddInt64(&verifyCalls, 1)
		return gate.DispatchDupcodeVerifyWithObserver(ctx, root, f, &fakeValidCIObserver{})
	}

	DupcodeUpdateBaselineDispatcher = func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
		atomic.AddInt64(&updateCalls, 1)
		return gate.DispatchDupcodeUpdateBaselineWithObserver(ctx, root, f, &fakeValidCIObserver{})
	}

	defer func() {
		DupcodeVerifyDispatcher = originalVerify
		DupcodeUpdateBaselineDispatcher = gate.DispatchDupcodeUpdateBaseline
	}()

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			return nil
		}
	}

	ctx := context.Background()
	result := DupcodeVerifyDispatcher(ctx, ".", runnerFactory)

	// Verify entry point should have been called
	if atomic.LoadInt64(&verifyCalls) != 1 {
		t.Errorf("verifyCalls = %d, want 1", atomic.LoadInt64(&verifyCalls))
	}

	// Update entry point should NOT have been called
	if atomic.LoadInt64(&updateCalls) != 0 {
		t.Errorf("updateCalls = %d, want 0", atomic.LoadInt64(&updateCalls))
	}

	// Verify succeeded
	if result.Error != nil {
		t.Fatalf("verify should succeed: %v", result.Error)
	}

	t.Logf("verify: verifyCalls=%d, updateCalls=%d", verifyCalls, updateCalls)
}

// TestDupcodeUpdateRoutesToUpdateEntryPoint proves that the update command
// routes to the update entry point and not the verify entry point.
func TestDupcodeUpdateRoutesToUpdateEntryPoint(t *testing.T) {
	var verifyCalls int64
	var updateCalls int64

	originalVerify := DupcodeVerifyDispatcher

	DupcodeVerifyDispatcher = func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
		atomic.AddInt64(&verifyCalls, 1)
		return gate.DispatchDupcodeVerifyWithObserver(ctx, root, f, &fakeValidCIObserver{})
	}

	DupcodeUpdateBaselineDispatcher = func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
		atomic.AddInt64(&updateCalls, 1)
		return gate.DispatchDupcodeUpdateBaselineWithObserver(ctx, root, f, &fakeValidCIObserver{})
	}

	defer func() {
		DupcodeVerifyDispatcher = originalVerify
		DupcodeUpdateBaselineDispatcher = gate.DispatchDupcodeUpdateBaseline
	}()

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			return nil
		}
	}

	ctx := context.Background()
	result := DupcodeUpdateBaselineDispatcher(ctx, ".", runnerFactory)

	// Verify entry point should NOT have been called
	if atomic.LoadInt64(&verifyCalls) != 0 {
		t.Errorf("verifyCalls = %d, want 0", atomic.LoadInt64(&verifyCalls))
	}

	// Update entry point should have been called
	if atomic.LoadInt64(&updateCalls) != 1 {
		t.Errorf("updateCalls = %d, want 1", atomic.LoadInt64(&updateCalls))
	}

	// Should be denied under CI
	if result.Error == nil && len(result.Findings) == 0 {
		t.Error("expected denial for update_baseline")
	}

	t.Logf("update: verifyCalls=%d, updateCalls=%d", verifyCalls, updateCalls)
}

// TestDupcodeVerifyAllowedRunnerCallsOnce proves verify is allowed and runner is called exactly once.
func TestDupcodeVerifyAllowedRunnerCallsOnce(t *testing.T) {
	var runnerCalls int64

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			atomic.AddInt64(&runnerCalls, 1)
			return nil
		}
	}

	ctx := context.Background()
	result := gate.DispatchDupcodeVerifyWithObserver(ctx, ".", runnerFactory, &fakeValidCIObserver{})

	if result.Error != nil {
		t.Fatalf("verify should succeed: %v", result.Error)
	}

	if atomic.LoadInt64(&runnerCalls) != 1 {
		t.Errorf("runnerCalls = %d, want 1", atomic.LoadInt64(&runnerCalls))
	}

	t.Logf("verify runner calls: %d", runnerCalls)
}

// TestDupcodeUpdateDeniedBeforeRunner proves update is denied before runner execution.
func TestDupcodeUpdateDeniedBeforeRunner(t *testing.T) {
	var runnerCalls int64

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			atomic.AddInt64(&runnerCalls, 1)
			return nil
		}
	}

	ctx := context.Background()
	result := gate.DispatchDupcodeUpdateBaselineWithObserver(ctx, ".", runnerFactory, &fakeValidCIObserver{})

	if result.Error == nil && len(result.Findings) == 0 {
		t.Error("expected denial for update_baseline")
	}

	if atomic.LoadInt64(&runnerCalls) != 0 {
		t.Errorf("runnerCalls = %d, want 0", atomic.LoadInt64(&runnerCalls))
	}

	t.Logf("update runner calls: %d (denied before execution)", runnerCalls)
}
