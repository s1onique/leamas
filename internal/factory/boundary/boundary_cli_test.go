// Package boundary provides verification for domain boundary import policies.
package boundary

import (
	"testing"
)

// TestCockpitCLIAllowsContext verifies that cockpit CLI allows context.
func TestCockpitCLIAllowsContext(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/cockpit.go" && f.Import == "context" {
			t.Errorf("cockpit CLI should allow context import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitCLIAllowsNetHTTP verifies that cockpit CLI allows net/http.
func TestCockpitCLIAllowsNetHTTP(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/cockpit.go" && f.Import == "net/http" {
			t.Errorf("cockpit CLI should allow net/http import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitCLIAllowsOSSignal verifies that cockpit CLI allows os/signal.
func TestCockpitCLIAllowsOSSignal(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/cockpit.go" && f.Import == "os/signal" {
			t.Errorf("cockpit CLI should allow os/signal import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitCLIAllowsInternalCockpit verifies that cockpit CLI allows internal/web/cockpit.
func TestCockpitCLIAllowsInternalCockpit(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/cockpit.go" && f.Import == "github.com/s1onique/leamas/internal/web/cockpit" {
			t.Errorf("cockpit CLI should allow internal/web/cockpit import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessCLIAllowsContext verifies that witness CLI allows context.
func TestWitnessCLIAllowsContext(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/witness.go" && f.Import == "context" {
			t.Errorf("witness CLI should allow context import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessCLIAllowsNetHTTP verifies that witness CLI allows net/http.
func TestWitnessCLIAllowsNetHTTP(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/witness.go" && f.Import == "net/http" {
			t.Errorf("witness CLI should allow net/http import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessCLIAllowsOSSignal verifies that witness CLI allows os/signal.
func TestWitnessCLIAllowsOSSignal(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/witness.go" && f.Import == "os/signal" {
			t.Errorf("witness CLI should allow os/signal import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessCLIAllowsInternalWitnessProxy verifies that witness CLI allows internal/witness/proxy.
func TestWitnessCLIAllowsInternalWitnessProxy(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/witness.go" && f.Import == "github.com/s1onique/leamas/internal/witness/proxy" {
			t.Errorf("witness CLI should allow internal/witness/proxy import, but got violation: %s", f.Reason)
		}
	}
}

// TestCLIRuntimeRejectsDatabaseSQL verifies that CLI runtime files reject database/sql.
func TestCLIRuntimeRejectsDatabaseSQL(t *testing.T) {
	tmpDir := t.TempDir()
	createProtectedDirsWithCLI(t, tmpDir)
	createCLIRuntimeFile(t, tmpDir, "cmd/leamas/cockpit.go", `
package main

import (
	"context"
	"database/sql"
)

func main() {}
`)

	result := Check(tmpDir)

	found := false
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/cockpit.go" && f.Import == "database/sql" {
			found = true
			if f.Reason != "CLI runtime must not import database packages" {
				t.Errorf("unexpected reason: %s", f.Reason)
			}
		}
	}
	if !found {
		t.Error("expected finding for database/sql in CLI runtime file")
	}
}

// TestCLIRuntimeRejectsOsExec verifies that CLI runtime files reject os/exec.
func TestCLIRuntimeRejectsOsExec(t *testing.T) {
	tmpDir := t.TempDir()
	createProtectedDirsWithCLI(t, tmpDir)
	createCLIRuntimeFile(t, tmpDir, "cmd/leamas/witness.go", `
package main

import (
	"os/exec"
)

func main() {}
`)

	result := Check(tmpDir)

	found := false
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/witness.go" && f.Import == "os/exec" {
			found = true
			if f.Reason != "CLI runtime must not import process execution" {
				t.Errorf("unexpected reason: %s", f.Reason)
			}
		}
	}
	if !found {
		t.Error("expected finding for os/exec in CLI runtime file")
	}
}

// TestCLIRuntimeRejectsProviderImport verifies that CLI runtime files reject provider imports.
func TestCLIRuntimeRejectsProviderImport(t *testing.T) {
	tmpDir := t.TempDir()
	createProtectedDirsWithCLI(t, tmpDir)
	createCLIRuntimeFile(t, tmpDir, "cmd/leamas/cockpit.go", `
package main

import (
	"context"
	"github.com/somebody/openai-sdk"
)

func main() {}
`)

	result := Check(tmpDir)

	found := false
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/cockpit.go" && f.Import == "github.com/somebody/openai-sdk" {
			found = true
			if f.Reason != "imports provider/control-plane package containing: openai" {
				t.Errorf("unexpected reason: %s", f.Reason)
			}
		}
	}
	if !found {
		t.Error("expected finding for openai provider import in CLI runtime file")
	}
}

// TestCLIRuntimeRejectsAuthImport verifies that CLI runtime files reject auth/session imports.
func TestCLIRuntimeRejectsAuthImport(t *testing.T) {
	tmpDir := t.TempDir()
	createProtectedDirsWithCLI(t, tmpDir)
	createCLIRuntimeFile(t, tmpDir, "cmd/leamas/witness.go", `
package main

import (
	"context"
	"github.com/auth/session-manager"
)

func main() {}
`)

	result := Check(tmpDir)

	found := false
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/witness.go" && f.Import == "github.com/auth/session-manager" {
			found = true
			if f.Reason != "imports provider/control-plane package containing: session" {
				t.Errorf("unexpected reason: %s", f.Reason)
			}
		}
	}
	if !found {
		t.Error("expected finding for session auth import in CLI runtime file")
	}
}

// TestHulkStillRejectsNetHTTP verifies that Hulk packages still reject net/http.
func TestHulkStillRejectsNetHTTP(t *testing.T) {
	tmpDir := t.TempDir()
	createProtectedDirsWithCLI(t, tmpDir)
	createHulkFile(t, tmpDir, "internal/hulk/runbundle/example.go", `
package runbundle

import (
	"net/http"
)

func Example() {}
`)

	result := Check(tmpDir)

	found := false
	for _, f := range result.Findings {
		if f.File == "internal/hulk/runbundle/example.go" && f.Import == "net/http" {
			found = true
			if f.Reason != "Hulk domain core must not import HTTP/network packages" {
				t.Errorf("unexpected reason: %s", f.Reason)
			}
		}
	}
	if !found {
		t.Error("expected finding for net/http in Hulk package")
	}
}

// TestHulkStillRejectsTime verifies that Hulk packages still reject time.
func TestHulkStillRejectsTime(t *testing.T) {
	tmpDir := t.TempDir()
	createProtectedDirsWithCLI(t, tmpDir)
	createHulkFile(t, tmpDir, "internal/hulk/claimevidence/example.go", `
package claimevidence

import (
	"time"
)

func Example() {}
`)

	result := Check(tmpDir)

	found := false
	for _, f := range result.Findings {
		if f.File == "internal/hulk/claimevidence/example.go" && f.Import == "time" {
			found = true
			if f.Reason != "Hulk domain core must not import time" {
				t.Errorf("unexpected reason: %s", f.Reason)
			}
		}
	}
	if !found {
		t.Error("expected finding for time in Hulk package")
	}
}

// TestWebCockpitStillRejectsHttputil verifies that Web cockpit still rejects net/http/httputil.
func TestWebCockpitStillRejectsHttputil(t *testing.T) {
	tmpDir := t.TempDir()
	createProtectedDirsWithCLI(t, tmpDir)
	createCockpitFile(t, tmpDir, "internal/web/cockpit/example.go", `
package cockpit

import (
	"net/http/httputil"
)

func Example() {}
`)

	result := Check(tmpDir)

	found := false
	for _, f := range result.Findings {
		if f.File == "internal/web/cockpit/example.go" && f.Import == "net/http/httputil" {
			found = true
			if f.Reason != "Web cockpit must not import reverse proxy utilities" {
				t.Errorf("unexpected reason: %s", f.Reason)
			}
		}
	}
	if !found {
		t.Error("expected finding for httputil in Web cockpit package")
	}
}

// TestWitnessProxyStillRejectsDatabaseSQL verifies that Witness proxy still rejects database/sql.
func TestWitnessProxyStillRejectsDatabaseSQL(t *testing.T) {
	tmpDir := t.TempDir()
	createProtectedDirsWithCLI(t, tmpDir)
	createWitnessFile(t, tmpDir, "internal/witness/proxy/example.go", `
package proxy

import (
	"database/sql"
)

func Example() {}
`)

	result := Check(tmpDir)

	found := false
	for _, f := range result.Findings {
		if f.File == "internal/witness/proxy/example.go" && f.Import == "database/sql" {
			found = true
			if f.Reason != "Witness proxy must not import database packages" {
				t.Errorf("unexpected reason: %s", f.Reason)
			}
		}
	}
	if !found {
		t.Error("expected finding for database/sql in Witness proxy package")
	}
}

// TestCLIRuntimeRejectsForbiddenInternal verifies that CLI runtime files reject forbidden internal imports.
func TestCLIRuntimeRejectsForbiddenInternal(t *testing.T) {
	tmpDir := t.TempDir()
	createProtectedDirsWithCLI(t, tmpDir)
	createCLIRuntimeFile(t, tmpDir, "cmd/leamas/cockpit.go", `
package main

import (
	"github.com/s1onique/leamas/internal/hulk/runbundle"
)

func main() {}
`)

	result := Check(tmpDir)

	found := false
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/cockpit.go" && f.Import == "github.com/s1onique/leamas/internal/hulk/runbundle" {
			found = true
			if f.Reason != "CLI runtime must not import Hulk runbundle package" {
				t.Errorf("unexpected reason: %s", f.Reason)
			}
		}
	}
	if !found {
		t.Error("expected finding for forbidden internal import in CLI runtime file")
	}
}

// TestBoundaryTestFilesIgnored verifies that *_test.go files are ignored in boundary tests.
func TestBoundaryTestFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	createProtectedDirsWithCLI(t, tmpDir)
	createHulkFile(t, tmpDir, "internal/hulk/runbundle/example_test.go", `
package runbundle

import (
	"net/http"
)

func TestExample() {}
`)

	result := Check(tmpDir)

	for _, f := range result.Findings {
		if f.File == "internal/hulk/runbundle/example_test.go" {
			t.Error("test files should be ignored but were checked")
		}
	}
}
