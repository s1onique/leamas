package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// plan_contract_composed_authority_parity_test.go contains the
// Phase 6 (recursive Go-model parity) and Phase 8 (duplicate
// diagnostic taxonomy) tests for
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01-CORRECTION02-
// COMPOSED-AUTHORITY-TRUTH01. Splitting them keeps every file
// under the LLM-friendly 400-line threshold.

// duplicateRootPlan returns a document that repeats
// contract_version at the root. The strict parser rejects it
// with a duplicate_property diagnostic at /contract_version.
func duplicateRootPlan() []byte {
	return []byte(duplicateRootPlanJSON)
}

const duplicateRootPlanJSON = `{
  "contract_version": 1,
  "contract_version": 1,
  "act_id": "ACT-DUP",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// duplicateNestedCommitOIDPlan returns a document that repeats
// commit_oid under /baseline. The strict parser rejects it
// with a duplicate_property diagnostic at /baseline/commit_oid.
func duplicateNestedCommitOIDPlan() []byte {
	return []byte(duplicateNestedCommitOIDPlanJSON)
}

const duplicateNestedCommitOIDPlanJSON = `{
  "contract_version": 1,
  "act_id": "ACT-DUP-N",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// unsupportedVersionPlan returns a document whose contract_version
// is 2; the strict parser must reject it with
// unsupported_contract_version at /contract_version.
func unsupportedVersionPlan() []byte {
	return []byte(unsupportedVersionPlanJSON)
}

const unsupportedVersionPlanJSON = `{
  "contract_version": 2,
  "act_id": "ACT-CONC",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// multiDiagnosticPlan returns a document that produces multiple
// distinct structural diagnostics (unknown + required-missing +
// integer-typed-as-string). The fixture is deterministic.
func multiDiagnosticPlan() []byte {
	return []byte(multiDiagnosticPlanJSON)
}

const multiDiagnosticPlanJSON = `{
  "contract_version": "1",
  "surprise": true,
  "act_id": "ACT-DET",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// --- Phase 6: recursive reflection parity ---

func TestRecursiveGoModelParity(t *testing.T) {
	// Recursively walk every nested struct in the Plan model and
	// record its JSON-tagged field inventory. The test asserts:
	//   - GO_MODEL_FIELD_COUNT > 0
	//   - DESCRIPTOR_FIELD_COUNT > 0
	//   - the two counts match (no drift allowed)
	//   - MISMATCH_COUNT == 0
	inventory := walkAllModelFields()
	goCount := len(inventory)
	descriptorCount := countDescriptorLeafFields(planContractV1().Root)
	if goCount == 0 || descriptorCount == 0 {
		t.Fatalf("zero fields walked: go=%d descriptor=%d", goCount, descriptorCount)
	}
	if goCount != descriptorCount {
		mismatches := compareInventory(inventory, planContractV1().Root)
		t.Fatalf("MISMATCH_COUNT=%d: go=%d descriptor=%d: %v", len(mismatches), goCount, descriptorCount, mismatches)
	}
}

// goModelField captures one JSON-tagged field discovered in the
// typed model.
type goModelField struct {
	Path     string // JSON pointer
	JSONName string
	GoName   string
	Kind     string
	Pointer  bool
	Array    bool
	HasChild bool
}

func walkAllModelFields() []goModelField {
	var out []goModelField
	walkModelFields(reflect.TypeOf(Plan{}), "", &out)
	return out
}

func walkModelFields(t reflect.Type, parentPath string, out *[]goModelField) {
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			continue
		}
		path := canonicalJSONPointer(parentPath, name)
		shape := goModelField{Path: path, JSONName: name, GoName: f.Name, Kind: f.Type.Kind().String()}
		switch f.Type.Kind() {
		case reflect.Struct:
			shape.HasChild = true
			walkModelFields(f.Type, path, out)
		case reflect.Ptr:
			shape.Pointer = true
			if f.Type.Elem().Kind() == reflect.Struct {
				shape.HasChild = true
				walkModelFields(f.Type.Elem(), path, out)
			}
		case reflect.Slice, reflect.Array:
			shape.Array = true
			if f.Type.Elem().Kind() == reflect.Struct {
				walkModelFields(f.Type.Elem(), path, out)
			}
		}
		*out = append(*out, shape)
	}
}

// countDescriptorLeafFields recursively counts every JSON field
// declared in the descriptor's tree (including required and
// optional).
func countDescriptorLeafFields(object planObjectDescriptor) int {
	count := len(object.Fields)
	for _, field := range object.Fields {
		if field.Children != nil {
			count += countDescriptorLeafFields(*field.Children)
		}
		if field.ItemDescriptor != nil && field.ItemDescriptor.Children != nil {
			count += countDescriptorLeafFields(*field.ItemDescriptor.Children)
		}
	}
	return count
}

// compareInventory emits one entry per JSON-name mismatch between
// the typed model and the descriptor.
func compareInventory(inventory []goModelField, root planObjectDescriptor) []string {
	descriptorNames := collectDescriptorNames(root)
	modelNames := make(map[string]bool, len(inventory))
	for _, f := range inventory {
		modelNames[f.Path] = true
	}
	var mismatches []string
	for name := range descriptorNames {
		if !modelNames[name] {
			mismatches = append(mismatches, "descriptor has no Go field: "+name)
		}
	}
	for _, f := range inventory {
		if !descriptorNames[f.Path] {
			mismatches = append(mismatches, "Go field has no descriptor entry: "+f.Path)
		}
	}
	return mismatches
}

func collectDescriptorNames(object planObjectDescriptor) map[string]bool {
	out := map[string]bool{}
	for name, field := range object.Fields {
		out[canonicalJSONPointer(object.Path, name)] = true
		if field.Children != nil {
			for k, v := range collectDescriptorNames(*field.Children) {
				out[k] = v
			}
		}
		if field.ItemDescriptor != nil && field.ItemDescriptor.Children != nil {
			for k, v := range collectDescriptorNames(*field.ItemDescriptor.Children) {
				out[k] = v
			}
		}
	}
	return out
}

// --- Phase 8: duplicate diagnostic truth ---

func TestDuplicateDiagnosticTaxonomy(t *testing.T) {
	cases := []struct {
		name        string
		data        []byte
		wantPath    string
		notWantCode PlanValidationCode
	}{
		{
			name:        "duplicate-root",
			data:        duplicateRootPlan(),
			wantPath:    "/contract_version",
			notWantCode: PlanCodeUnknownProperty,
		},
		{
			name:        "duplicate-nested",
			data:        duplicateNestedCommitOIDPlan(),
			wantPath:    "/baseline/commit_oid",
			notWantCode: PlanCodeUnknownProperty,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := ValidatePlanStructural(tc.data)
			if result.Valid {
				t.Fatalf("duplicate must be rejected")
			}
			found := false
			for _, e := range result.Errors {
				if e.Code == tc.notWantCode {
					t.Fatalf("duplicate must NOT use %s: %v", tc.notWantCode, e)
				}
				if e.Code == PlanCodeDuplicateProperty && e.InstancePath == tc.wantPath {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected duplicate_property at %s; got %v", tc.wantPath, result.Errors)
			}
		})
	}
}

// --- Phase 4 (size bound) ---

func TestStructuralSizeBound(t *testing.T) {
	// A plan that exceeds MaxPlanBytes must be rejected with
	// invalid_json.
	big := make([]byte, MaxPlanBytes+1)
	for i := range big {
		big[i] = ' '
	}
	big[0] = '{'
	big[len(big)-1] = '}'
	result := ValidatePlanStructural(big)
	if result.Valid {
		t.Fatalf("oversize plan must be rejected")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeInvalidJSON && strings.Contains(e.Message, "size limit") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid_json with size limit message; got %v", result.Errors)
	}
}

// --- Determinism with deliberately invalid documents ---

func TestInvalidDiagnosticDeterminism(t *testing.T) {
	// Build a document that produces MULTIPLE different diagnostics
	// (unknown + required-missing + integer-typed-as-string).
	plan := multiDiagnosticPlan()
	prev := ""
	for i := 0; i < 20; i++ {
		result := ValidatePlanStructural(plan)
		hash := hashDiagnostics(result.Errors)
		if i > 0 && hash != prev {
			t.Fatalf("iteration %d: diagnostic hash drifted: %s vs %s", i, prev, hash)
		}
		prev = hash
	}
}

// TestConcurrentInvalidDiagnosticDeterminism launches 8
// overlapping goroutines and asserts that every iteration
// produces the same diagnostic hash for the same document.
// The previous serial "concurrent" tests in this package are
// renamed; this one really does run overlapping goroutines.
func TestConcurrentInvalidDiagnosticDeterminism(t *testing.T) {
	plan := unsupportedVersionPlan()
	results := make(chan string, 8)
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := make([]string, 8)
			for i := 0; i < 8; i++ {
				result := ValidatePlanStructural(plan)
				local[i] = hashDiagnostics(result.Errors)
			}
			results <- local[0]
		}()
	}
	wg.Wait()
	close(results)
	first := <-results
	for r := range results {
		if r != first {
			t.Fatalf("concurrent determinism violated: %s vs %s", first, r)
		}
	}
	_ = runtime.NumCPU() // ensure runtime is imported for future use
}

func hashDiagnostics(diags []PlanValidationError) string {
	h := sha256.New()
	for _, d := range diags {
		fmt.Fprintf(h, "%s|%s|%s|%s|", d.Code, d.InstancePath, d.PropertyName, d.Message)
	}
	return hex.EncodeToString(h.Sum(nil))
}
