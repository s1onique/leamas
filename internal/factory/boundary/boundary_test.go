package boundary

import (
	"testing"
)

// TestCurrentRepoPoliciesPass verifies that current protected packages pass.
func TestCurrentRepoPoliciesPass(t *testing.T) {
	result := Check(".")
	if !result.OK() {
		for _, f := range result.Findings {
			t.Errorf("boundary violation: %s imports %s: %s", f.File, f.Import, f.Reason)
		}
	}
}

// TestHulkRunbundleAllowsSort verifies that hulk runbundle allows sort import.
func TestHulkRunbundleAllowsSort(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/hulk/runbundle/runbundle.go" && f.Import == "sort" {
			t.Errorf("runbundle should allow sort import, but got violation: %s", f.Reason)
		}
	}
}

// TestHulkClaimevidenceAllowsSort verifies that hulk claimevidence allows sort import.
func TestHulkClaimevidenceAllowsSort(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/hulk/claimevidence/claimevidence.go" && f.Import == "sort" {
			t.Errorf("claimevidence should allow sort import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessProxyAllowsNetHTTP verifies that witness proxy allows net/http.
func TestWitnessProxyAllowsNetHTTP(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/witness/proxy/proxy.go" && f.Import == "net/http" {
			t.Errorf("witness proxy should allow net/http import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessProxyAllowsHttputil verifies that witness proxy allows net/http/httputil.
func TestWitnessProxyAllowsHttputil(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/witness/proxy/proxy.go" && f.Import == "net/http/httputil" {
			t.Errorf("witness proxy should allow httputil import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitAllowsEmbed verifies that cockpit allows embed.
func TestCockpitAllowsEmbed(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/web/cockpit/cockpit.go" && f.Import == "embed" {
			t.Errorf("cockpit should allow embed import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitAllowsEncodingJSON verifies that cockpit allows encoding/json.
func TestCockpitAllowsEncodingJSON(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/web/cockpit/cockpit.go" && f.Import == "encoding/json" {
			t.Errorf("cockpit should allow encoding/json import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitAllowsNetHTTP verifies that cockpit allows net/http.
func TestCockpitAllowsNetHTTP(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/web/cockpit/cockpit.go" && f.Import == "net/http" {
			t.Errorf("cockpit should allow net/http import, but got violation: %s", f.Reason)
		}
	}
}

// TestFindingsDeterministic verifies that findings are in deterministic order.
func TestFindingsDeterministic(t *testing.T) {
	result1 := Check(".")
	result2 := Check(".")

	if len(result1.Findings) != len(result2.Findings) {
		t.Fatalf("different number of findings: %d vs %d", len(result1.Findings), len(result2.Findings))
	}

	for i := range result1.Findings {
		if result1.Findings[i] != result2.Findings[i] {
			t.Errorf("finding at index %d differs:\n  first:  %+v\n  second: %+v", i, result1.Findings[i], result2.Findings[i])
		}
	}
}

// TestResultOK verifies the Result.OK() method.
func TestResultOK(t *testing.T) {
	emptyResult := Result{}
	if !emptyResult.OK() {
		t.Error("empty result should be OK")
	}

	nonEmptyResult := Result{
		Findings: []Finding{
			{File: "test.go", Import: "net/http", Reason: "forbidden"},
		},
	}
	if nonEmptyResult.OK() {
		t.Error("non-empty result should not be OK")
	}
}
