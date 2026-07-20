// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"strings"
	"testing"
	"time"
)

// ContractStatsKeysV3 is the canonical v3 key order emitted by
// RenderStats. The order is load-bearing for downstream consumers
// and any reordering constitutes a contract-version bump.
var ContractStatsKeysV3 = []string{
	"files_changed",
	"added_files",
	"modified_files",
	"deleted_files",
	"type_changed_files",
	"renamed_files",
	"copied_files",
	"unmerged_files",
	"unknown_files",
	"broken_pair_files",
	"untracked_files",
	"binary_files",
	"generated_files",
	"test_files",
	"doc_files",
	"source_files",
	"config_files",
}

func TestContractVersion_IsThree(t *testing.T) {
	if ContractVersion != 3 {
		t.Errorf("ContractVersion = %d, want 3", ContractVersion)
	}
}

func TestContractHeaderFields_ExpectedCount(t *testing.T) {
	if len(ContractHeaderFields) != 6 {
		t.Errorf("len(ContractHeaderFields) = %d, want 6", len(ContractHeaderFields))
	}
}

func TestContractHeaderFields_StableOrder(t *testing.T) {
	expected := []string{
		ContractFieldVersion,
		ContractFieldAppVer,
		ContractFieldCommit,
		ContractFieldBuildTime,
		ContractFieldMode,
		ContractFieldCreatedAt,
	}
	for i, field := range expected {
		if ContractHeaderFields[i] != field {
			t.Errorf("ContractHeaderFields[%d] = %q, want %q", i, ContractHeaderFields[i], field)
		}
	}
}

func TestRenderContractHeader_AllFieldsPresent(t *testing.T) {
	info := HeaderInfo{
		Version:   "1.0.0",
		Commit:    "abc123",
		BuildTime: "2026-01-01T00:00:00Z",
		Mode:      ModeDirty,
		CreatedAt: "2026-09-07T12:00:00Z",
	}
	header := RenderContractHeader(info)

	for _, field := range ContractHeaderFields {
		if !strings.Contains(header, field+":") {
			t.Errorf("header missing field %q", field)
		}
	}
}

func TestRenderContractHeader_ContractVersionIsInteger(t *testing.T) {
	info := HeaderInfo{
		Mode:      ModeDirty,
		CreatedAt: "2026-09-07T12:00:00Z",
	}
	header := RenderContractHeader(info)

	if !strings.Contains(header, "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3") {
		t.Error("header should contain contract version as integer 3")
	}
}

func TestRenderContractHeader_UsesProvidedVersion(t *testing.T) {
	info := HeaderInfo{
		Version:   "3.0.0",
		Commit:    "test-commit",
		BuildTime: "2026-01-01T00:00:00Z",
		Mode:      ModeStaged,
		CreatedAt: "2026-09-07T12:00:00Z",
	}
	header := RenderContractHeader(info)

	if !strings.Contains(header, "LEAMAS_VERSION: 3.0.0") {
		t.Errorf("header should contain provided version, got: %s", header)
	}
	if !strings.Contains(header, "LEAMAS_COMMIT: test-commit") {
		t.Errorf("header should contain provided commit, got: %s", header)
	}
	if !strings.Contains(header, "DIGEST_MODE: staged") {
		t.Errorf("header should contain mode, got: %s", header)
	}
}

func TestRenderContractHeader_TimestampIsRFC3339(t *testing.T) {
	info := HeaderInfo{
		Mode:      ModeDirty,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	header := RenderContractHeader(info)

	// Extract timestamp line
	var tsLine string
	for _, line := range strings.Split(header, "\n") {
		if strings.HasPrefix(line, "DIGEST_CREATED_AT:") {
			tsLine = strings.TrimPrefix(line, "DIGEST_CREATED_AT: ")
			break
		}
	}

	if tsLine == "" {
		t.Fatal("DIGEST_CREATED_AT not found in header")
	}

	// Parse as RFC3339
	ts, err := time.Parse(time.RFC3339, tsLine)
	if err != nil {
		t.Errorf("timestamp %q is not valid RFC3339: %v", tsLine, err)
	}

	// Verify it's UTC
	if ts.Location() != time.UTC {
		t.Errorf("timestamp should be UTC, got %s", ts.Location())
	}
}

func TestRenderContractHeader_TrailingBlankLine(t *testing.T) {
	info := HeaderInfo{
		Mode:      ModeDirty,
		CreatedAt: "2026-09-07T12:00:00Z",
	}
	header := RenderContractHeader(info)

	// Header should end with blank line
	if !strings.HasSuffix(header, "\n\n") {
		t.Errorf("header should end with blank line separator, got: %q", header)
	}
}

func TestParseContractHeader_ExtractsHeaderAndBody(t *testing.T) {
	// Use the actual rendered header from RenderContractHeader
	info := HeaderInfo{
		Version:   "dev",
		Commit:    "abc",
		BuildTime: "unknown",
		Mode:      ModeDirty,
		CreatedAt: "2026-09-07T12:00:00Z",
	}
	header := RenderContractHeader(info)
	body := "# Targeted digest\n\nSome body content"
	content := header + body

	parsedHeader, parsedBody := ParseContractHeader(content)

	if !strings.Contains(parsedHeader, "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3") {
		t.Error("parsed header should contain contract version")
	}
	if !strings.Contains(parsedBody, "# Targeted digest") {
		t.Errorf("parsed body should contain digest header, got: %q", parsedBody)
	}
	if !strings.Contains(parsedBody, "Some body content") {
		t.Errorf("parsed body should contain remaining content, got: %q", parsedBody)
	}
}

func TestParseContractHeader_InvalidContent(t *testing.T) {
	content := `# Targeted digest
Some content`

	header, body := ParseContractHeader(content)

	if header != "" {
		t.Error("invalid content should return empty header")
	}
	if body != content {
		t.Error("invalid content should return original content as body")
	}
}

func TestValidateContractHeader_ValidHeader(t *testing.T) {
	header := `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3
LEAMAS_VERSION: dev
LEAMAS_COMMIT: abc
LEAMAS_BUILD_TIME: unknown
DIGEST_MODE: dirty
DIGEST_CREATED_AT: 2026-01-01T00:00:00Z
`
	err := ValidateContractHeader(header)
	if err != nil {
		t.Errorf("valid header should not error: %v", err)
	}
}

func TestValidateContractHeader_WrongFieldOrder(t *testing.T) {
	header := `LEAMAS_VERSION: dev
LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3
LEAMAS_COMMIT: abc
LEAMAS_BUILD_TIME: unknown
DIGEST_MODE: dirty
DIGEST_CREATED_AT: 2026-01-01T00:00:00Z
`
	err := ValidateContractHeader(header)
	if err == nil {
		t.Error("wrong field order should error")
	}
}

func TestValidateContractHeader_MissingField(t *testing.T) {
	header := `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3
LEAMAS_VERSION: dev
LEAMAS_COMMIT: abc
LEAMAS_BUILD_TIME: unknown
DIGEST_MODE: dirty
`
	err := ValidateContractHeader(header)
	if err == nil {
		t.Error("missing field should error")
	}
}

// TestRenderStats_V3CanonicalKeyOrder locks the v3 stats-key
// sequence. The test feeds a fully populated FileStats and parses
// the rendered section line-by-line, then asserts the keys appear
// in the exact v3 order. Any reordering will fail this test and is
// not permitted without bumping the contract version.
func TestRenderStats_V3CanonicalKeyOrder(t *testing.T) {
	stats := FileStats{
		FilesChanged:     5,
		AddedFiles:       1,
		ModifiedFiles:    2,
		DeletedFiles:     0,
		TypeChangedFiles: 1,
		RenamedFiles:     1,
		CopiedFiles:      0,
		UntrackedFiles:   2,
		UnmergedFiles:    0,
		UnknownFiles:     0,
		BrokenPairFiles:  0,
		BinaryFiles:      0,
		GeneratedFiles:   0,
		TestFiles:        1,
		DocFiles:         1,
		SourceFiles:      2,
		ConfigFiles:      0,
	}
	out := RenderStats(stats)

	wantKeys := ContractStatsKeysV3
	gotKeys := make([]string, 0, len(wantKeys))
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") || !strings.Contains(line, "=") {
			continue
		}
		gotKeys = append(gotKeys, strings.SplitN(line, "=", 2)[0])
	}

	if len(gotKeys) != len(wantKeys) {
		t.Fatalf("rendered stats key count = %d, want %d\nrendered:\n%s",
			len(gotKeys), len(wantKeys), out)
	}
	for i := range wantKeys {
		if gotKeys[i] != wantKeys[i] {
			t.Fatalf("stats key order mismatch at index %d: got %q, want %q\nfull order: %#v",
				i, gotKeys[i], wantKeys[i], gotKeys)
		}
	}
}

// TestRenderStats_V3IncludesNewFields makes sure every v3 stats key
// appears in the rendered output at least once. If a future
// regression drops one of the new fields (type_changed_files,
// unknown_files, broken_pair_files) this test catches it.
func TestRenderStats_V3IncludesNewFields(t *testing.T) {
	stats := FileStats{
		TypeChangedFiles: 1,
		UnknownFiles:     1,
		BrokenPairFiles:  1,
	}
	out := RenderStats(stats)
	for _, key := range []string{"type_changed_files", "unknown_files", "broken_pair_files"} {
		if !strings.Contains(out, key+"=") {
			t.Errorf("rendered stats missing v3 key %q\nout:\n%s", key, out)
		}
	}
}
