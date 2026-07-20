// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"fmt"
	"regexp"
	"strings"
)

// envKeyRegex validates environment variable names: [A-Za-z_][A-Za-z0-9_]*
var envKeyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidateVerifier checks that a verifier has all required metadata.
func ValidateVerifier(v Verifier) error {
	if v.Name == "" {
		return fmt.Errorf("verifier name is required")
	}
	if v.Run == nil {
		return fmt.Errorf("verifier %q has nil Run function", v.Name)
	}
	if v.Execution.Kind != ExecutionInProcess && v.Execution.Kind != ExecutionChild {
		return fmt.Errorf("verifier %q has invalid execution kind: %q", v.Name, v.Execution.Kind)
	}
	if v.Execution.ImplementationID == "" {
		return fmt.Errorf("verifier %q has empty ImplementationID", v.Name)
	}

	// Validate environment keys: no empty, no duplicates, must match env var name pattern
	seen := make(map[string]bool)
	for _, key := range v.Execution.EnvVars {
		if key == "" {
			return fmt.Errorf("verifier %q has empty environment key", v.Name)
		}
		if strings.Contains(key, "=") {
			return fmt.Errorf("verifier %q has malformed environment key %q (contains =)", v.Name, key)
		}
		if strings.TrimSpace(key) != key {
			return fmt.Errorf("verifier %q has malformed environment key %q (has whitespace)", v.Name, key)
		}
		if !envKeyRegex.MatchString(key) {
			return fmt.Errorf("verifier %q has malformed environment key %q (invalid name)", v.Name, key)
		}
		if seen[key] {
			return fmt.Errorf("verifier %q has duplicate environment key %q", v.Name, key)
		}
		seen[key] = true
	}

	// Check cache semantics validity
	switch v.Cache.GoBuildCache {
	case CacheRelevant, CacheNotRelevant, CacheNotApplicable:
		// valid
	default:
		return fmt.Errorf("verifier %q has invalid GoBuildCache: %q", v.Name, v.Cache.GoBuildCache)
	}
	switch v.Cache.GoTestResultCache {
	case CacheModeEnabled, CacheModeDisabled, CacheModeNA:
		// valid
	default:
		return fmt.Errorf("verifier %q has invalid GoTestResultCache: %q", v.Name, v.Cache.GoTestResultCache)
	}
	return nil
}

// ValidateVerifiers checks that all verifiers have valid metadata.
func ValidateVerifiers(verifiers []Verifier) error {
	seen := make(map[string]bool)
	for _, v := range verifiers {
		if err := ValidateVerifier(v); err != nil {
			return err
		}
		if seen[v.Name] {
			return fmt.Errorf("duplicate verifier name: %q", v.Name)
		}
		seen[v.Name] = true
	}
	return nil
}
