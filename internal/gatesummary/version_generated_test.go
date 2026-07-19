package gatesummary

import (
	"fmt"
	"strings"
	"testing"
)

const (
	v1Template = `{"schema_version": %s, "generated_at": "2026-07-19T08:43:26Z", "overall_status": "pass", "checks": []}`

	// v2Template is split across two consts to keep each line under
	// the 240-character LLM-friendliness cap.
	v2Prefix = `{"schema_version": %s, "generated_at": "2026-07-19T08:43:26Z", ` +
		`"scope_id": "ACT-X", "scope_status": "CLOSED", ` +
		`"scope_disposition": "d", "parent_act": "", ` +
		`"parent_status": "CLOSED", "parent_disposition": "d", ` +
		`"overall_status": "pass", "overall_disposition": "d", `
	v2Suffix = `"execution_head_oid": "0123456789abcdef0123456789abcdef01234567", ` +
		`"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567", ` +
		`"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567", ` +
		`"worktree_clean_before": true, "worktree_clean_after": true, ` +
		`"checks": []}`

	v2Template = v2Prefix + v2Suffix
)

func TestVersionLexicalCaseAccounting(t *testing.T) {
	const normativeContractCases = 100 + 6 + 9 + 8
	const templateExpandedExecutions = 100 + 12 + 18 + 16
	if normativeContractCases != 123 || templateExpandedExecutions != 146 {
		t.Fatalf("lexical accounting drift: normative=%d expanded=%d",
			normativeContractCases, templateExpandedExecutions)
	}
}

func TestVersionGeneratedWhitespaceMatrix(t *testing.T) {
	whitespace := []string{"", " ", "\t", "\n", "\r"}
	placements := []string{"before-comma", "before-brace"}
	count := 0
	for _, version := range []string{"1", "2"} {
		template := templateForVersion(version)
		wantVersion := versionFromString(version)
		for _, prefix := range whitespace {
			for _, suffix := range whitespace {
				for _, placement := range placements {
					count++
					raw := injectWhitespace(template, version, prefix, suffix, placement)
					trace := decodeTrace{}
					res := decodeWithTrace(strings.NewReader(raw), &trace)
					if !res.Success() || res.Document.Version() != wantVersion {
						t.Errorf("case %d (v=%s/p=%q/s=%q/%s): result=%+v",
							count, version, prefix, suffix, placement, res)
					}
					assertGeneratedTrace(t, count, trace, stageWireDecode, wantVersion, true, true)
				}
			}
		}
	}
	if count != 100 {
		t.Errorf("expected 100 whitespace cases, got %d", count)
	}
}

func TestVersionGeneratedLeadingZeroPlus(t *testing.T) {
	lexemes := []string{"01", "02", "-01", "-02", "+1", "+2"}
	count := runRejectedVersionMatrix(t, lexemes, CodeMalformedJSON, stageSyntaxScan)
	if count != 12 {
		t.Errorf("expected 12 leading-zero/plus executions, got %d", count)
	}
}

func TestVersionGeneratedDecimalExponent(t *testing.T) {
	lexemes := []string{"1.0", "2.0", "2.00", "-2.0", "1e0", "2e0", "2E0", "2e+0", "2e-0"}
	count := runRejectedVersionMatrix(t, lexemes, CodeInvalidVersionType, stageVersionProbe)
	if count != 18 {
		t.Errorf("expected 18 decimal/exponent executions, got %d", count)
	}
}

func TestVersionGeneratedUnsupportedInteger(t *testing.T) {
	lexemes := []string{
		"-2", "-1", "-0", "0", "3", "4",
		"99999999999999999999", "-99999999999999999999",
	}
	count := runRejectedVersionMatrix(t, lexemes, CodeUnsupportedVersion, stageVersionDispatch)
	if count != 16 {
		t.Errorf("expected 16 unsupported-integer executions, got %d", count)
	}
}

func runRejectedVersionMatrix(t *testing.T, lexemes []string, code string, owner stage) int {
	t.Helper()
	count := 0
	for _, raw := range lexemes {
		for _, version := range []string{"1", "2"} {
			count++
			body := fmt.Sprintf(templateForVersion(version), raw)
			trace := decodeTrace{}
			res := decodeWithTrace(strings.NewReader(body), &trace)
			if res.Success() || res.Err != nil || len(res.Diagnostics) != 1 ||
				res.Diagnostics[0].Code != code {
				t.Errorf("case %d (raw=%s/template=%s): diagnostics=%+v err=%v",
					count, raw, version, res.Diagnostics, res.Err)
			}
			assertGeneratedTrace(t, count, trace, owner, 0, false, false)
		}
	}
	return count
}

func assertGeneratedTrace(
	t *testing.T,
	caseNumber int,
	got decodeTrace,
	owner stage,
	selected Version,
	schemaInvoked bool,
	wireDecoded bool,
) {
	t.Helper()
	if got.Stage != owner || got.SchemaSelected != selected ||
		got.SchemaInvoked != schemaInvoked || got.WireDecoded != wireDecoded {
		t.Errorf("case %d trace=%+v, want stage=%s selected=%s schema=%v wire=%v",
			caseNumber, got, owner, selected, schemaInvoked, wireDecoded)
	}
}

func templateForVersion(version string) string {
	if version == "2" {
		return v2Template
	}
	return v1Template
}

// injectWhitespace inserts prefix and suffix around lexeme in the
// schema_version position. placement controls whether the lexeme
// lives at the closing-brace position or before a comma.
func injectWhitespace(template, lexeme, prefix, suffix, placement string) string {
	body := strings.Replace(template, "%s", prefix+lexeme+suffix, 1)
	if placement == "before-brace" {
		body = strings.Replace(body,
			`"schema_version": `+prefix+lexeme+suffix+`,`,
			``, 1)
		body = strings.TrimRight(body, " ")
		body = strings.TrimSuffix(body, "}")
		body = body + `, "schema_version": ` + prefix + lexeme + suffix + `}`
	}
	return body
}
