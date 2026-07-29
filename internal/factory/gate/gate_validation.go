// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/s1onique/leamas/internal/factory/registry"
)

// envKeyRegex validates environment variable names: [A-Za-z_][A-Za-z0-9_]*
var envKeyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ErrInvalidLane indicates a verifier has an unrecognized lane.
var ErrInvalidLane = errors.New("invalid verifier lane")

// ErrLanePartitionIncomplete indicates not all verifiers were assigned to a lane.
var ErrLanePartitionIncomplete = errors.New("verifier lane partition is incomplete")

// ValidateVerifier checks that a verifier has all required metadata.
//
// Command-only verifiers are accepted with a nil Run function: the typed
// binder is their only execution path, and registry.Validate enforces
// that gate-scoped verifiers must have a non-nil Run function.
func ValidateVerifier(v registry.Verifier) error {
	if v.Name == "" {
		return fmt.Errorf("verifier name is required")
	}
	if v.Run == nil && v.Scope != registry.InvocationCommandOnly {
		return fmt.Errorf("verifier %q has nil Run function", v.Name)
	}
	// Validate lane: only fast and dupcode are supported
	switch v.Lane {
	case registry.VerifierLaneFast, registry.VerifierLaneDupcode:
		// valid
	default:
		return fmt.Errorf("%w: %q is not %q or %q",
			ErrInvalidLane, v.Lane, registry.VerifierLaneFast, registry.VerifierLaneDupcode)
	}
	if v.Execution.Kind != registry.ExecutionInProcess && v.Execution.Kind != registry.ExecutionChild {
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
	case registry.CacheRelevant, registry.CacheNotRelevant, registry.CacheNotApplicable:
		// valid
	default:
		return fmt.Errorf("verifier %q has invalid GoBuildCache: %q", v.Name, v.Cache.GoBuildCache)
	}
	switch v.Cache.GoTestResultCache {
	case registry.CacheModeEnabled, registry.CacheModeDisabled, registry.CacheModeNA:
		// valid
	default:
		return fmt.Errorf("verifier %q has invalid GoTestResultCache: %q", v.Name, v.Cache.GoTestResultCache)
	}
	return nil
}

// ValidateVerifiers checks that all verifiers have valid metadata.
func ValidateVerifiers(verifiers []registry.Verifier) error {
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
//
// Command-only definitions are excluded from the partitioned lanes and from
// the partition completeness check. They are reachable via
// DispatcherForVerifier only.
func PartitionVerifiers(verifiers []registry.Verifier) (fast, dupcode []registry.Verifier, err error) {
	// Filter out command-only entries before validation so the
	// gate-scoped validation rules (Run required) do not apply to them.
	eligible := make([]registry.Verifier, 0, len(verifiers))
	for _, v := range verifiers {
		if v.Scope == registry.InvocationCommandOnly {
			continue
		}
		eligible = append(eligible, v)
	}

	// Fail closed: validate ALL eligible verifiers first
	if err := ValidateVerifiers(eligible); err != nil {
		return nil, nil, fmt.Errorf("verifier registry validation failed: %w", err)
	}

	for _, v := range eligible {
		switch v.Lane {
		case registry.VerifierLaneFast:
			fast = append(fast, v)
		case registry.VerifierLaneDupcode:
			dupcode = append(dupcode, v)
		default:
			// This should be unreachable due to ValidateVerifiers, but defense in depth
			return nil, nil, fmt.Errorf("%w: verifier %q has unknown lane %q",
				ErrInvalidLane, v.Name, v.Lane)
		}
	}

	// Fail closed: verify no eligible verifier was dropped
	if len(fast)+len(dupcode) != len(eligible) {
		return nil, nil, ErrLanePartitionIncomplete
	}

	return fast, dupcode, nil
}
