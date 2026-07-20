// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// envKeyRegex validates environment variable names: [A-Za-z_][A-Za-z0-9_]*
var envKeyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ErrInvalidLane indicates a verifier has an unrecognized lane.
var ErrInvalidLane = errors.New("invalid verifier lane")

// ErrLanePartitionIncomplete indicates not all verifiers were assigned to a lane.
var ErrLanePartitionIncomplete = errors.New("verifier lane partition is incomplete")

// ValidateVerifier checks that a verifier has all required metadata.
func ValidateVerifier(v Verifier) error {
	if v.Name == "" {
		return fmt.Errorf("verifier name is required")
	}
	if v.Run == nil {
		return fmt.Errorf("verifier %q has nil Run function", v.Name)
	}
	// Validate lane: only fast and dupcode are supported
	switch v.Lane {
	case VerifierLaneFast, VerifierLaneDupcode:
		// valid
	default:
		return fmt.Errorf("%w: %q is not %q or %q",
			ErrInvalidLane, v.Lane, VerifierLaneFast, VerifierLaneDupcode)
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

// PartitionVerifiers partitions the verifier registry into fast and dupcode lanes.
// It validates all verifiers before partitioning and fails closed if any verifier
// has an invalid or unknown lane.
func PartitionVerifiers(verifiers []Verifier) (fast, dupcode []Verifier, err error) {
	// Fail closed: validate ALL verifiers first
	if err := ValidateVerifiers(verifiers); err != nil {
		return nil, nil, fmt.Errorf("verifier registry validation failed: %w", err)
	}

	for _, v := range verifiers {
		switch v.Lane {
		case VerifierLaneFast:
			fast = append(fast, v)
		case VerifierLaneDupcode:
			dupcode = append(dupcode, v)
		default:
			// This should be unreachable due to ValidateVerifiers, but defense in depth
			return nil, nil, fmt.Errorf("%w: verifier %q has unknown lane %q",
				ErrInvalidLane, v.Name, v.Lane)
		}
	}

	// Fail closed: verify no verifier was dropped
	if len(fast)+len(dupcode) != len(verifiers) {
		return nil, nil, ErrLanePartitionIncomplete
	}

	return fast, dupcode, nil
}
