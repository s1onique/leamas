// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import "fmt"

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
	for _, key := range v.Execution.EnvVars {
		if key == "" {
			return fmt.Errorf("verifier %q has empty environment key", v.Name)
		}
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
