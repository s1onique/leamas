// Package github provides verification tests for GitHub repository policy compliance.
package github

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckPolicyDocument_AllFieldsPresent(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs", "factory")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	validPolicy := `# GitHub Policy

## Branch Protection Policy

github:
  repository: s1onique/leamas
  branch_protection:
    - pattern: main
      required_status_checks:
        - Factory Gates
      require_strict_status_checks: true
      enforce_for_admins: false
      allow_force_pushes: false
      allow_deletions: false
`
	policyPath := filepath.Join(docsDir, "github-policy.md")
	if err := os.WriteFile(policyPath, []byte(validPolicy), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckPolicyDocument(tmpDir)
	for _, f := range findings {
		if f.Severity == "error" {
			t.Errorf("unexpected error: %s: %s", f.Kind, f.Message)
		}
	}
}

func TestCheckPolicyDocument_MissingFields(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs", "factory")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	missingFieldsPolicy := `# GitHub Policy

github:
  repository: s1onique/leamas
  branch_protection:
    - pattern: main
`
	policyPath := filepath.Join(docsDir, "github-policy.md")
	if err := os.WriteFile(policyPath, []byte(missingFieldsPolicy), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckPolicyDocument(tmpDir)

	errorCount := 0
	for _, f := range findings {
		if f.Severity == "error" && f.Kind == "missing_field" {
			errorCount++
		}
	}

	if errorCount < 4 {
		t.Errorf("expected at least 4 missing field errors, got %d", errorCount)
	}
}

func TestCheckPolicyDocument_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	findings := CheckPolicyDocument(tmpDir)

	if len(findings) == 0 {
		t.Error("expected finding for missing file")
	}

	if findings[0].Kind != "missing" {
		t.Errorf("expected 'missing' kind, got '%s'", findings[0].Kind)
	}
}

func TestFindMainBranchProtection(t *testing.T) {
	rules := []BranchProtectionRule{
		{Pattern: "main"},
		{Pattern: "develop"},
	}

	mainRule := findMainBranchProtection(rules)
	if mainRule == nil {
		t.Fatal("expected to find main branch protection rule")
	}
	if mainRule.Pattern != "main" {
		t.Errorf("expected pattern 'main', got '%s'", mainRule.Pattern)
	}
}

func TestFindMainBranchProtection_ByMatchingRefs(t *testing.T) {
	rules := []BranchProtectionRule{
		{
			Pattern: "release/*",
			MatchingRefs: struct {
				Nodes []struct {
					Name string `json:"name"`
				} `json:"nodes"`
			}{
				Nodes: []struct {
					Name string `json:"name"`
				}{
					{Name: "main"},
				},
			},
		},
	}

	mainRule := findMainBranchProtection(rules)
	if mainRule == nil {
		t.Fatal("expected to find main branch protection rule by matchingRefs")
	}
}

func TestFindMainBranchProtection_NotFound(t *testing.T) {
	rules := []BranchProtectionRule{
		{Pattern: "develop"},
		{Pattern: "feature/*"},
	}

	mainRule := findMainBranchProtection(rules)
	if mainRule != nil {
		t.Error("expected nil for missing main branch protection rule")
	}
}

func TestCheckRules_AdminEnforcementTrue(t *testing.T) {
	policy := DefaultPolicy()
	rules := []BranchProtectionRule{
		{
			Pattern:                     "main",
			IsAdminEnforced:             true,
			AllowsForcePushes:           false,
			AllowsDeletions:             false,
			RequiresStatusChecks:        true,
			RequiresStrictStatusChecks:  true,
			RequiredStatusCheckContexts: []string{"Factory Gates"},
		},
	}

	findings := CheckRules(policy, rules)

	if len(findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "policy_drift" {
		t.Errorf("expected policy_drift, got %s", findings[0].Kind)
	}
}

func TestCheckRules_AdminEnforcementFalse(t *testing.T) {
	policy := DefaultPolicy()
	rules := []BranchProtectionRule{
		{
			Pattern:                     "main",
			IsAdminEnforced:             false,
			AllowsForcePushes:           false,
			AllowsDeletions:             false,
			RequiresStatusChecks:        true,
			RequiresStrictStatusChecks:  true,
			RequiredStatusCheckContexts: []string{"Factory Gates"},
		},
	}

	findings := CheckRules(policy, rules)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestCheckRules_ForcePushEnabled(t *testing.T) {
	policy := DefaultPolicy()
	rules := []BranchProtectionRule{
		{
			Pattern:                     "main",
			IsAdminEnforced:             false,
			AllowsForcePushes:           true,
			AllowsDeletions:             false,
			RequiresStatusChecks:        true,
			RequiresStrictStatusChecks:  true,
			RequiredStatusCheckContexts: []string{"Factory Gates"},
		},
	}

	findings := CheckRules(policy, rules)

	found := false
	for _, f := range findings {
		if f.Kind == "policy_drift" && f.Message == "allow_force_pushes: expected disabled, got true" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for force push enabled")
	}
}

func TestCheckRules_DeletionEnabled(t *testing.T) {
	policy := DefaultPolicy()
	rules := []BranchProtectionRule{
		{
			Pattern:                     "main",
			IsAdminEnforced:             false,
			AllowsForcePushes:           false,
			AllowsDeletions:             true,
			RequiresStatusChecks:        true,
			RequiresStrictStatusChecks:  true,
			RequiredStatusCheckContexts: []string{"Factory Gates"},
		},
	}

	findings := CheckRules(policy, rules)

	found := false
	for _, f := range findings {
		if f.Kind == "policy_drift" && f.Message == "allow_deletions: expected disabled, got true" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for deletion enabled")
	}
}

func TestCheckRules_MissingFactoryGates(t *testing.T) {
	policy := DefaultPolicy()
	rules := []BranchProtectionRule{
		{
			Pattern:                     "main",
			IsAdminEnforced:             false,
			AllowsForcePushes:           false,
			AllowsDeletions:             false,
			RequiresStatusChecks:        true,
			RequiresStrictStatusChecks:  true,
			RequiredStatusCheckContexts: []string{"Other Check"},
		},
	}

	findings := CheckRules(policy, rules)

	found := false
	for _, f := range findings {
		if f.Kind == "policy_drift" && f.Message == "required_status_checks: expected exactly [Factory Gates], got [Other Check]" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for missing Factory Gates")
	}
}

func TestCheckRules_ExtraRequiredCheck(t *testing.T) {
	policy := DefaultPolicy()
	rules := []BranchProtectionRule{
		{
			Pattern:                     "main",
			IsAdminEnforced:             false,
			AllowsForcePushes:           false,
			AllowsDeletions:             false,
			RequiresStatusChecks:        true,
			RequiresStrictStatusChecks:  true,
			RequiredStatusCheckContexts: []string{"Factory Gates", "Extra Check"},
		},
	}

	findings := CheckRules(policy, rules)

	if len(findings) != 1 {
		t.Errorf("expected 1 finding, got %d: %v", len(findings), findings)
	}
	if findings[0].Kind != "policy_drift" {
		t.Errorf("expected policy_drift, got %s", findings[0].Kind)
	}
}

func TestCheckRules_MissingMainRule(t *testing.T) {
	policy := DefaultPolicy()
	rules := []BranchProtectionRule{
		{Pattern: "develop"},
	}

	findings := CheckRules(policy, rules)

	if len(findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "missing" {
		t.Errorf("expected missing, got %s", findings[0].Kind)
	}
}

func TestSlicesEqual(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{"empty", []string{}, []string{}, true},
		{"single equal", []string{"a"}, []string{"a"}, true},
		{"single not equal", []string{"a"}, []string{"b"}, false},
		{"same two elements different order", []string{"b", "a"}, []string{"a", "b"}, true},
		{"different length", []string{"a"}, []string{"a", "b"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slicesEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("slicesEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
