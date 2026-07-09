// Package main provides tests for factory verify dispatch.
package main

import (
	"testing"
)

func TestKnownFactoryVerifyChecks_IncludesDoctrines(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "doctrine" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'doctrine'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesDocs(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "docs" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'docs'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesForbiddenPatterns(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "forbidden-patterns" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'forbidden-patterns'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesLanguage(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "language" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'language'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesStaticBinary(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "static-binary" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'static-binary'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesToolingBoundaries(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "tooling-boundaries" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'tooling-boundaries'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesLLMFriendly(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "llm-friendly" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'llm-friendly'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesAgentContext(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "agent-context" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'agent-context'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesGitHooks(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "git-hooks" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'git-hooks'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesGithub(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "github" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'github'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesDomainBoundaries(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "domain-boundaries" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'domain-boundaries'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesCoverage(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "coverage" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'coverage'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesDoctrinesAgentContracts(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "doctrine-agent-contracts" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'doctrine-agent-contracts'")
	}
}

func TestKnownFactoryVerifyChecks_IncludesDupcode(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	found := false
	for _, c := range checks {
		if c == "dupcode" {
			found = true
			break
		}
	}
	if !found {
		t.Error("known checks should include 'dupcode'")
	}
}

func TestKnownFactoryVerifyChecks_HasExpectedCount(t *testing.T) {
	checks := knownFactoryVerifyChecks()
	// We expect 15 checks based on the current implementation (dupcode + dupcode-baseline)
	expected := 15
	if len(checks) != expected {
		t.Errorf("expected %d known checks, got %d", expected, len(checks))
	}
}

func TestIsKnownFactoryVerifyCheck_TrueForKnownChecks(t *testing.T) {
	knownChecks := []string{
		"doctrine",
		"doctrine-agent-contracts",
		"docs",
		"forbidden-patterns",
		"language",
		"static-binary",
		"tooling-boundaries",
		"llm-friendly",
		"agent-context",
		"git-hooks",
		"github",
		"domain-boundaries",
		"coverage",
	}

	for _, check := range knownChecks {
		if !isKnownFactoryVerifyCheck(check) {
			t.Errorf("expected %q to be a known check", check)
		}
	}
}

func TestIsKnownFactoryVerifyCheck_FalseForUnknownCheck(t *testing.T) {
	if isKnownFactoryVerifyCheck("unknown-check") {
		t.Error("expected 'unknown-check' to not be a known check")
	}
}

func TestIsKnownFactoryVerifyCheck_FalseForEmptyString(t *testing.T) {
	if isKnownFactoryVerifyCheck("") {
		t.Error("expected empty string to not be a known check")
	}
}

func TestIsKnownFactoryVerifyCheck_FalseForPartialMatch(t *testing.T) {
	if isKnownFactoryVerifyCheck("doctrine-extra") {
		t.Error("expected 'doctrine-extra' to not be a known check")
	}
}

func TestIsKnownFactoryVerifyCheck_CaseSensitive(t *testing.T) {
	if isKnownFactoryVerifyCheck("DOCTRINE") {
		t.Error("expected 'DOCTRINE' to not be a known check (case sensitive)")
	}
}
