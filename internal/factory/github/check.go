// Package github provides verification for GitHub repository policy compliance.
package github

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// Policy represents the desired GitHub policy for Leamas.
type Policy struct {
	Repository       string             `yaml:"repository"`
	BranchProtection []BranchProtection `yaml:"branch_protection"`
}

// BranchProtection represents a branch protection rule.
type BranchProtection struct {
	Pattern                   string   `yaml:"pattern"`
	RequiredStatusChecks      []string `yaml:"required_status_checks"`
	RequireStrictStatusChecks bool     `yaml:"require_strict_status_checks"`
	EnforceForAdmins          bool     `yaml:"enforce_for_admins"`
	AllowForcePushes          bool     `yaml:"allow_force_pushes"`
	AllowDeletions            bool     `yaml:"allow_deletions"`
}

// BranchProtectionRule represents the GitHub GraphQL response.
type BranchProtectionRule struct {
	ID                          string   `json:"id"`
	Pattern                     string   `json:"pattern"`
	IsAdminEnforced             bool     `json:"isAdminEnforced"`
	AllowsForcePushes           bool     `json:"allowsForcePushes"`
	AllowsDeletions             bool     `json:"allowsDeletions"`
	RequiresStatusChecks        bool     `json:"requiresStatusChecks"`
	RequiresStrictStatusChecks  bool     `json:"requiresStrictStatusChecks"`
	RequiredStatusCheckContexts []string `json:"requiredStatusCheckContexts"`
	MatchingRefs                struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"matchingRefs"`
}

// Finding represents a verification finding.
type Finding struct {
	Path     string
	Kind     string
	Message  string
	Severity string
}

// DefaultPolicy returns the Leamas GitHub policy.
func DefaultPolicy() Policy {
	return Policy{
		Repository: "s1onique/leamas",
		BranchProtection: []BranchProtection{
			{
				Pattern:                   "main",
				RequiredStatusChecks:      []string{"Factory Gates"},
				RequireStrictStatusChecks: true,
				EnforceForAdmins:          false,
				AllowForcePushes:          false,
				AllowDeletions:            false,
			},
		},
	}
}

// CheckRepo verifies GitHub policy compliance.
func CheckRepo(root string) ([]Finding, error) {
	policyPath := filepath.Join(root, "docs", "factory", "github-policy.md")
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		return []Finding{
			{
				Path:     policyPath,
				Kind:     "missing",
				Message:  "GitHub policy document not found",
				Severity: "error",
			},
		}, nil
	}

	// Run gh CLI to get branch protection rules
	rules, err := fetchBranchProtectionRules()
	if err != nil {
		return []Finding{
			{
				Path:     "GitHub API",
				Kind:     "api_error",
				Message:  fmt.Sprintf("failed to fetch branch protection rules: %v", err),
				Severity: "error",
			},
		}, nil
	}

	// Use default policy
	policy := DefaultPolicy()

	return CheckRules(policy, rules), nil
}

// CheckRules verifies that branch protection rules match the policy.
// This is a pure function for testability.
func CheckRules(policy Policy, rules []BranchProtectionRule) []Finding {
	var findings []Finding

	// Find main branch rule
	mainRule := findMainBranchProtection(rules)
	if mainRule == nil {
		findings = append(findings, Finding{
			Path:     "main",
			Kind:     "missing",
			Message:  "main branch protection rule not found",
			Severity: "error",
		})
		return findings
	}

	// Find policy for main
	var mainPolicy BranchProtection
	for _, p := range policy.BranchProtection {
		if p.Pattern == "main" {
			mainPolicy = p
			break
		}
	}

	// Verify enforce_for_admins
	if mainRule.IsAdminEnforced != mainPolicy.EnforceForAdmins {
		expected := "disabled"
		if mainPolicy.EnforceForAdmins {
			expected = "enabled"
		}
		findings = append(findings, Finding{
			Path:     "main",
			Kind:     "policy_drift",
			Message:  fmt.Sprintf("enforce_for_admins: expected %s, got %t", expected, mainRule.IsAdminEnforced),
			Severity: "error",
		})
	}

	// Verify force pushes
	if mainRule.AllowsForcePushes != mainPolicy.AllowForcePushes {
		expected := "disabled"
		if mainPolicy.AllowForcePushes {
			expected = "enabled"
		}
		findings = append(findings, Finding{
			Path:     "main",
			Kind:     "policy_drift",
			Message:  fmt.Sprintf("allow_force_pushes: expected %s, got %t", expected, mainRule.AllowsForcePushes),
			Severity: "error",
		})
	}

	// Verify deletions
	if mainRule.AllowsDeletions != mainPolicy.AllowDeletions {
		expected := "disabled"
		if mainPolicy.AllowDeletions {
			expected = "enabled"
		}
		findings = append(findings, Finding{
			Path:     "main",
			Kind:     "policy_drift",
			Message:  fmt.Sprintf("allow_deletions: expected %s, got %t", expected, mainRule.AllowsDeletions),
			Severity: "error",
		})
	}

	// Verify required status checks enabled
	if !mainRule.RequiresStatusChecks {
		findings = append(findings, Finding{
			Path:     "main",
			Kind:     "policy_drift",
			Message:  "requires_status_checks should be enabled",
			Severity: "error",
		})
	}

	// Verify strict status checks
	if mainRule.RequiresStrictStatusChecks != mainPolicy.RequireStrictStatusChecks {
		findings = append(findings, Finding{
			Path:     "main",
			Kind:     "policy_drift",
			Message:  fmt.Sprintf("require_strict_status_checks: expected %t, got %t", mainPolicy.RequireStrictStatusChecks, mainRule.RequiresStrictStatusChecks),
			Severity: "error",
		})
	}

	// Verify exact required status check contexts match
	if !slicesEqual(mainRule.RequiredStatusCheckContexts, mainPolicy.RequiredStatusChecks) {
		findings = append(findings, Finding{
			Path:     "main",
			Kind:     "policy_drift",
			Message:  fmt.Sprintf("required_status_checks: expected exactly %v, got %v", mainPolicy.RequiredStatusChecks, mainRule.RequiredStatusCheckContexts),
			Severity: "error",
		})
	}

	return findings
}

// slicesEqual compares two string slices for equality (order-independent).
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aSorted := make([]string, len(a))
	bSorted := make([]string, len(b))
	copy(aSorted, a)
	copy(bSorted, b)
	sort.Strings(aSorted)
	sort.Strings(bSorted)
	return reflect.DeepEqual(aSorted, bSorted)
}

// fetchBranchProtectionRules fetches branch protection rules from GitHub API.
func fetchBranchProtectionRules() ([]BranchProtectionRule, error) {
	query := `query {
  repository(owner: "s1onique", name: "leamas") {
    branchProtectionRules(first: 50) {
      nodes {
        id
        pattern
        isAdminEnforced
        allowsForcePushes
        allowsDeletions
        requiresStatusChecks
        requiresStrictStatusChecks
        requiredStatusCheckContexts
        matchingRefs(first: 20) {
          nodes {
            name
          }
        }
      }
    }
  }
}`

	cmd := exec.Command("gh", "api", "graphql",
		"-f", fmt.Sprintf("query=%s", query))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh api failed: %w, output: %s", err, string(output))
	}

	var response struct {
		Data struct {
			Repository struct {
				BranchProtectionRules struct {
					Nodes []BranchProtectionRule `json:"nodes"`
				} `json:"branchProtectionRules"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Data.Repository.BranchProtectionRules.Nodes, nil
}

// findMainBranchProtection finds the main branch protection rule.
func findMainBranchProtection(rules []BranchProtectionRule) *BranchProtectionRule {
	for _, rule := range rules {
		if rule.Pattern == "main" {
			return &rule
		}
		// Also check matchingRefs
		for _, ref := range rule.MatchingRefs.Nodes {
			if ref.Name == "main" {
				return &rule
			}
		}
	}
	return nil
}

// CheckPolicyDocument verifies the policy document contains required fields.
func CheckPolicyDocument(root string) []Finding {
	var findings []Finding

	policyPath := filepath.Join(root, "docs", "factory", "github-policy.md")
	data, err := os.ReadFile(policyPath)
	if err != nil {
		findings = append(findings, Finding{
			Path:     policyPath,
			Kind:     "missing",
			Message:  "policy document not found",
			Severity: "error",
		})
		return findings
	}

	content := string(data)

	// Check for required policy fields
	requiredFields := map[string]string{
		"enforce_for_admins: false":          "enforce_for_admins should be false",
		"Factory Gates":                      "required_status_checks should include 'Factory Gates'",
		"require_strict_status_checks: true": "require_strict_status_checks should be true",
		"allow_force_pushes: false":          "allow_force_pushes should be false",
		"allow_deletions: false":             "allow_deletions should be false",
		"pattern: main":                      "main branch pattern should be defined",
	}

	for pattern, desc := range requiredFields {
		if !strings.Contains(content, pattern) {
			findings = append(findings, Finding{
				Path:     policyPath,
				Kind:     "missing_field",
				Message:  desc,
				Severity: "error",
			})
		}
	}

	return findings
}
