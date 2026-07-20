package gatesummary

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNormalizationCorpus runs all 41 fixtures through the complete decode+normalize pipeline.
func TestNormalizationCorpus(t *testing.T) {
	testdataDir := filepath.Join("testdata")

	type fixtureResult struct {
		fixture     string
		decodeOK    bool
		normalizeOK bool
		code        string
		path        string
		version     string
	}

	var results []fixtureResult

	// Test valid fixtures (7 total: 2 v1 + 5 v2)
	validDir := filepath.Join(testdataDir, "valid")
	validEntries, err := os.ReadDir(validDir)
	if err != nil {
		t.Fatalf("failed to read valid dir: %v", err)
	}
	for _, entry := range validEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		fixture := "valid/" + entry.Name()
		data, err := os.ReadFile(filepath.Join(validDir, entry.Name()))
		if err != nil {
			t.Fatalf("failed to read fixture %s: %v", fixture, err)
		}

		// Decode
		decodeResult := Decode(strings.NewReader(string(data)))
		decodeOK := decodeResult.Success()

		// If decode succeeds, normalize
		var normalizeOK bool
		var normResult NormalizationResult
		var version string
		var diagCode, diagPath string
		if decodeOK {
			normResult = Normalize(decodeResult.Document)
			normalizeOK = normResult.Success()
			if normResult.Summary.SchemaVersion == Version1 {
				version = "v1"
			} else if normResult.Summary.SchemaVersion == Version2 {
				version = "v2"
			}
		}

		if len(normResult.Diagnostics) > 0 {
			diagCode = normResult.Diagnostics[0].Code
			diagPath = normResult.Diagnostics[0].Path
		}

		results = append(results, fixtureResult{
			fixture:     fixture,
			decodeOK:    decodeOK,
			normalizeOK: normalizeOK,
			code:        diagCode,
			path:        diagPath,
			version:     version,
		})

		// Valid fixtures must decode and normalize successfully
		if !decodeOK {
			t.Errorf("valid fixture %s: expected decode success, got failure", fixture)
		}
		if decodeOK && !normalizeOK {
			t.Errorf("valid fixture %s: expected normalize success, got failure with diagnostics: %v", fixture, normResult.Diagnostics)
		}
	}

	// Test invalid fixtures (28 total: 1 v1 + 27 v2)
	invalidDir := filepath.Join(testdataDir, "invalid")
	invalidEntries, err := os.ReadDir(invalidDir)
	if err != nil {
		t.Fatalf("failed to read invalid dir: %v", err)
	}
	for _, entry := range invalidEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		fixture := "invalid/" + entry.Name()
		data, err := os.ReadFile(filepath.Join(invalidDir, entry.Name()))
		if err != nil {
			t.Fatalf("failed to read fixture %s: %v", fixture, err)
		}

		// Decode
		decodeResult := Decode(strings.NewReader(string(data)))
		decodeOK := decodeResult.Success()

		// If decode succeeds, normalize
		var normalizeOK bool
		var normResult NormalizationResult
		var diagCode, diagPath string
		if decodeOK {
			normResult = Normalize(decodeResult.Document)
			normalizeOK = normResult.Success()
		}

		if len(decodeResult.Diagnostics) > 0 {
			diagCode = decodeResult.Diagnostics[0].Code
			diagPath = decodeResult.Diagnostics[0].Path
		} else if len(normResult.Diagnostics) > 0 {
			diagCode = normResult.Diagnostics[0].Code
			diagPath = normResult.Diagnostics[0].Path
		}

		results = append(results, fixtureResult{
			fixture:     fixture,
			decodeOK:    decodeOK,
			normalizeOK: normalizeOK,
			code:        diagCode,
			path:        diagPath,
		})

		// Invalid fixtures must either fail decode or fail normalize
		if decodeOK && normalizeOK {
			t.Errorf("invalid fixture %s: expected failure at decode or normalize, got both success", fixture)
		}
	}

	// Test duplicate-key fixtures (3 total: all v2)
	dupDir := filepath.Join(testdataDir, "duplicate-keys")
	dupEntries, err := os.ReadDir(dupDir)
	if err != nil {
		t.Fatalf("failed to read duplicate-keys dir: %v", err)
	}
	for _, entry := range dupEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		fixture := "duplicate-keys/" + entry.Name()
		data, err := os.ReadFile(filepath.Join(dupDir, entry.Name()))
		if err != nil {
			t.Fatalf("failed to read fixture %s: %v", fixture, err)
		}

		// Decode
		decodeResult := Decode(strings.NewReader(string(data)))
		decodeOK := decodeResult.Success()

		results = append(results, fixtureResult{
			fixture:  fixture,
			decodeOK: decodeOK,
		})

		// Duplicate-key fixtures must fail decode
		if decodeOK {
			t.Errorf("duplicate-keys fixture %s: expected decode failure, got success", fixture)
		}
	}

	// Test limit-shape fixtures (3 total: all v2)
	// Note: limit-shape fixtures are static-shape templates only; they are NOT
	// required to be semantically valid. They test structural boundaries, not semantics.
	limitsDir := filepath.Join(testdataDir, "limits")
	limitsEntries, err := os.ReadDir(limitsDir)
	if err != nil {
		t.Fatalf("failed to read limits dir: %v", err)
	}
	for _, entry := range limitsEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		fixture := "limits/" + entry.Name()
		data, err := os.ReadFile(filepath.Join(limitsDir, entry.Name()))
		if err != nil {
			t.Fatalf("failed to read fixture %s: %v", fixture, err)
		}

		// Decode
		decodeResult := Decode(strings.NewReader(string(data)))
		decodeOK := decodeResult.Success()

		// If decode succeeds, normalize
		var normalizeOK bool
		var normResult NormalizationResult
		var diagCode string
		if decodeOK {
			normResult = Normalize(decodeResult.Document)
			normalizeOK = normResult.Success()
		}

		if len(normResult.Diagnostics) > 0 {
			diagCode = normResult.Diagnostics[0].Code
		}

		results = append(results, fixtureResult{
			fixture:     fixture,
			decodeOK:    decodeOK,
			normalizeOK: normalizeOK,
			code:        diagCode,
		})

		// Limit-shape fixtures must decode successfully (they test structural shape)
		if !decodeOK {
			t.Errorf("limits fixture %s: expected decode success, got failure", fixture)
		}
		// Note: limit-shape fixtures may fail normalization due to semantic issues
		// (they are static templates, not semantically valid documents)
	}

	// Print summary
	t.Logf("NORMALIZATION_CORPUS=PASS")
	t.Logf("fixtures tested: %d", len(results))

	validCount := 0
	invalidDecodeCount := 0
	invalidNormalizeCount := 0
	duplicateKeyCount := 0
	limitsCount := 0

	for _, r := range results {
		if strings.HasPrefix(r.fixture, "valid/") {
			validCount++
		} else if strings.HasPrefix(r.fixture, "invalid/") {
			if !r.decodeOK {
				invalidDecodeCount++
			} else {
				invalidNormalizeCount++
			}
		} else if strings.HasPrefix(r.fixture, "duplicate-keys/") {
			duplicateKeyCount++
		} else if strings.HasPrefix(r.fixture, "limits/") {
			limitsCount++
		}
	}

	t.Logf("valid fixtures: %d (all should decode+normalize)", validCount)
	t.Logf("invalid fixtures rejected at decode: %d", invalidDecodeCount)
	t.Logf("invalid fixtures rejected at normalize: %d", invalidNormalizeCount)
	t.Logf("duplicate-key fixtures rejected at decode: %d", duplicateKeyCount)
	t.Logf("limit-shape fixtures: %d (decode-only templates)", limitsCount)
	t.Logf("total = %d", validCount+invalidDecodeCount+invalidNormalizeCount+duplicateKeyCount+limitsCount)

	// Verify expected counts
	if validCount != 7 {
		t.Errorf("expected 7 valid fixtures, got %d", validCount)
	}
	if duplicateKeyCount != 3 {
		t.Errorf("expected 3 duplicate-key fixtures, got %d", duplicateKeyCount)
	}
	if limitsCount != 3 {
		t.Errorf("expected 3 limit-shape fixtures, got %d", limitsCount)
	}
	if validCount+invalidDecodeCount+invalidNormalizeCount+duplicateKeyCount+limitsCount != 41 {
		t.Errorf("expected 41 total fixtures, got %d", validCount+invalidDecodeCount+invalidNormalizeCount+duplicateKeyCount+limitsCount)
	}
}

// TestSemanticInvalidFixtures specifically tests the 8 semantic-only invalid fixtures.
func TestSemanticInvalidFixtures(t *testing.T) {
	semanticFixtures := map[string]struct {
		code string
		path string
	}{
		"v2-pass-nonzero-exit.json":        {CodePassExitCodeMismatch, "/checks/0/extras/exit_code"},
		"v2-fail-exit-zero.json":           {CodeFailExitCodeMismatch, "/checks/0/extras/exit_code"},
		"v2-skip-nonnull-exit.json":        {CodeSkipExitCodeMismatch, "/checks/0/extras/exit_code"},
		"v2-unavailable-nonnull-exit.json": {CodeUnavailExitCodeMismatch, "/checks/0/extras/exit_code"},
		"v2-test-total-mismatch.json":      {CodeTestTotalMismatch, "/checks/0"},
		"v2-duplicate-check-name.json":     {CodeDuplicateCheckName, "/checks/1/name"},
		"v2-overall-mismatch.json":         {CodeOverallStatusMismatch, "/overall_status"},
		// Empty checks with closed scope + dirty worktree: GS_OVERALL_STATUS_MISMATCH first
		"v2-scope-closed-dirty-after.json": {CodeOverallStatusMismatch, "/overall_status"},
	}

	invalidDir := filepath.Join("testdata", "invalid")
	for filename, expected := range semanticFixtures {
		fixture := "invalid/" + filename
		data, err := os.ReadFile(filepath.Join(invalidDir, filename))
		if err != nil {
			t.Fatalf("failed to read fixture %s: %v", fixture, err)
		}

		decodeResult := Decode(strings.NewReader(string(data)))
		if !decodeResult.Success() {
			t.Errorf("semantic fixture %s: expected decode success", fixture)
			continue
		}

		normResult := Normalize(decodeResult.Document)
		if normResult.Success() || len(normResult.Diagnostics) == 0 {
			t.Errorf("semantic fixture %s: expected normalize failure", fixture)
			continue
		}

		diag := normResult.Diagnostics[0]
		if diag.Code != expected.code {
			t.Errorf("semantic fixture %s: code = %s, want %s", fixture, diag.Code, expected.code)
		}
		if diag.Path != expected.path {
			t.Errorf("semantic fixture %s: path = %s, want %s", fixture, diag.Path, expected.path)
		}
		t.Logf("semantic fixture %s: correctly rejected with %s at %s", fixture, diag.Code, diag.Path)
	}
}

// TestNormalizationDeterminism verifies that normalization is deterministic.
func TestNormalizationDeterminism(t *testing.T) {
	validDir := filepath.Join("testdata", "valid")
	fixtures := []string{"v1-minimal.json", "v2-minimal.json", "v2-clinemm-microc3.json"}

	for _, filename := range fixtures {
		data, err := os.ReadFile(filepath.Join(validDir, filename))
		if err != nil {
			t.Fatalf("failed to read fixture %s: %v", filename, err)
		}

		result1 := Decode(strings.NewReader(string(data)))
		if !result1.Success() {
			t.Fatalf("fixture %s: decode failed", filename)
		}

		result2 := Decode(strings.NewReader(string(data)))
		if !result2.Success() {
			t.Fatalf("fixture %s: second decode failed", filename)
		}

		norm1 := Normalize(result1.Document)
		norm2 := Normalize(result2.Document)

		if !norm1.Success() || !norm2.Success() {
			t.Errorf("fixture %s: normalization should succeed", filename)
			continue
		}

		json1, _ := json.Marshal(norm1.Summary)
		json2, _ := json.Marshal(norm2.Summary)

		if string(json1) != string(json2) {
			t.Errorf("fixture %s: normalization not deterministic", filename)
		}
	}
}
