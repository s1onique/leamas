//go:build unix || darwin || linux

package execution

import (
	"runtime"
	"testing"
)

// TestAdversarialLinuxPlatformClassificationContract is a focused
// classification test that proves Linux rejects
// CodeExecutionProcessTreeCleanupFailed for the canonical SIGTERM
// escalation path. The test SKIPs on Darwin so the same code base
// passes CI on macOS runners without faking the platform.
//
// This is the executable form of the platform-exact invariant that the
// CORRECTION05 ACT requires: any Darwin-specific alternative MUST be
// explicitly runtime-gated and MUST NOT leak into the Linux allow-list.
func TestAdversarialLinuxPlatformClassificationContract(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skipf("darwin keeps the documented alternative; classification contract is non-applicable")
	}

	codes := allowedSigtermCodes()
	if _, ok := codes[CodeExecutionProcessTreeCleanupFailed]; ok {
		t.Errorf("on %s, allowedSigtermCodes() must NOT contain %q",
			runtime.GOOS, CodeExecutionProcessTreeCleanupFailed)
	}
	if _, ok := codes[CodeExecutionCancelled]; !ok {
		t.Errorf("on %s, allowedSigtermCodes() must contain %q",
			runtime.GOOS, CodeExecutionCancelled)
	}
}
