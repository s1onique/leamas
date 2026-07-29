// SPDX-License-Identifier: Apache-2.0

package verifierauthority

// ExecutionEnvironmentKind classifies the execution environment of a
// verifier invocation. It is a fail-closed classification: an unobserved
// or malformed context is never collapsed into a permissive kind.
//
// The classification is owned by the authority package and must not be
// inferred as "all relevant strings empty == local". A caller-controlled
// environment variable cannot single-handedly prove an explicit local
// invocation; only the trusted observer's explicit classification does.
type ExecutionEnvironmentKind string

const (
	// EnvironmentLocal means the trusted observer has explicitly
	// classified the invocation as a local execution via the
	// ExecutionContext.LocalTrust field.
	EnvironmentLocal ExecutionEnvironmentKind = "local"

	// EnvironmentGitHubActions means the trusted observer has observed the
	// GitHub Actions default variables. The CI variable is intentionally
	// not consulted because GitHub documents it as overridable while the
	// GITHUB_* variables are reserved.
	EnvironmentGitHubActions ExecutionEnvironmentKind = "github_actions"

	// EnvironmentCI means the trusted observer has observed another CI
	// provider (e.g. CI=true with no GITHUB_ACTIONS signal).
	EnvironmentCI ExecutionEnvironmentKind = "ci"

	// EnvironmentUnknown means the trusted observer could not classify the
	// environment. This is the fail-closed default: partial, contradictory,
	// malformed, or unclassified contexts land here, never on a permissive
	// kind.
	EnvironmentUnknown ExecutionEnvironmentKind = "unknown"
)

// LocalTrustSentinel is the only value of ExecutionContext.LocalTrust that
// permits a context to be classified as EnvironmentLocal. It is set by
// the trusted observer only and is never derived from environment
// variables. Tests use the trusted observer pattern, not the sentinel
// value directly.
const LocalTrustSentinel = "trusted-local-observer"

// ClassifyExecutionEnvironment derives the environment kind from an
// observed ExecutionContext. The function is fail-closed: only a fully
// self-consistent set of signals is mapped onto a permissive kind.
//
// Required classification rules:
//
//	GITHUB_ACTIONS=true                  -> github_actions
//	(authority marker validity, CI value,
//	 or partial population do not change this)
//
//	CI=true without GITHUB_ACTIONS=true  -> ci
//
//	LocalTrust == LocalTrustSentinel     -> local
//	(observed via the trusted observer)
//
//	any other partial, contradictory,
//	or unclassified state                 -> unknown
//
// GitHub Actions detection is intentionally independent of CI because
// GitHub documents CI as overridable while the GITHUB_* variables are
// reserved defaults. A workflow setting CI=false must not weaken the
// GitHub Actions classification.
//
// Empty contexts (LocalTrust unset, all environment strings empty) are
// classified as EnvironmentUnknown. The empty context cannot be
// promoted to local: an observation failure must not be conflated with
// a successful explicit local observation.
func ClassifyExecutionEnvironment(ec ExecutionContext) ExecutionEnvironmentKind {
	// GitHub Actions is detected purely from GITHUB_ACTIONS=true. Any
	// other state of the context does not weaken this classification:
	// the authority marker may be missing, CI may be empty or "false",
	// and the context may otherwise be partial.
	if ec.GitHubActions == "true" {
		return EnvironmentGitHubActions
	}

	// Generic CI signal: CI=true with no GITHUB_ACTIONS claim.
	if ec.CI == "true" {
		return EnvironmentCI
	}

	// Explicit local: only when the trusted observer has set the
	// LocalTrust sentinel. Environment variables do not qualify.
	if ec.LocalTrust == LocalTrustSentinel {
		return EnvironmentLocal
	}

	// Empty or partially populated contexts are unknown.
	return EnvironmentUnknown
}

// Reason codes for environment-classification denials.
const (
	ReasonCodeEnvironmentUnknown          = "environment_unknown"
	ReasonCodeEnvironmentGitHubActions    = "environment_github_actions"
	ReasonCodeEnvironmentCI               = "environment_ci"
	ReasonCodeEnvironmentMalformed        = "environment_malformed"
	ReasonCodeEnvironmentNotExplicitLocal = "environment_not_explicit_local"
)

// ValidateOperationInContext gates a mutation against both the declared
// authority and the classified execution environment. A local_safe
// mutation is admitted only when the environment is explicitly local;
// any GitHub Actions, other CI, partial, contradictory, or unknown
// environment is denied.
//
// Verification operations preserve their existing policy: they are
// admitted under their declared authority regardless of environment, and
// a ci_exact_checkout verifier that is denied by ValidateAuthority is
// still denied here.
//
// Required update-baseline matrix:
//
//	declared local_safe    + explicit local     -> admit
//	declared local_safe    + github_actions     -> deny
//	declared local_safe    + ci                 -> deny
//	declared local_safe    + unknown            -> deny
//	declared local_safe    + malformed          -> deny
//	declared ci_exact_checkout + any            -> deny update
//	unknown authority      + any                -> deny
func ValidateOperationInContext(
	declared ExecutionAuthority,
	operation VerifierOperation,
	environment ExecutionEnvironmentKind,
) error {
	switch declared {
	case AuthorityLocalSafe:
		if operation != OperationUpdateBaseline {
			// Non-mutation operations are admitted under local_safe.
			return nil
		}
		// Update-baseline under local_safe is admitted only in an
		// explicitly classified local environment.
		if environment != EnvironmentLocal {
			return &AuthorityError{
				RequiredAuthority: AuthorityLocalSafe,
				Operation:         operation,
				ReasonCode:        reasonForEnvironment(environment),
				Message:           messageForEnvironment(declared, operation, environment),
			}
		}
		return nil

	case AuthorityCIExactCheckout:
		// ci_exact_checkout never admits update_baseline.
		if operation == OperationUpdateBaseline {
			return &AuthorityError{
				RequiredAuthority: AuthorityCIExactCheckout,
				Operation:         operation,
				ReasonCode:        ReasonCodeOperationDenied,
				Message:           "update_baseline operation is not permitted under ci_exact_checkout authority",
			}
		}
		// Other operations are admitted under their declared authority;
		// ValidateAuthority performs the per-context acceptance checks.
		return nil

	default:
		return &AuthorityError{
			RequiredAuthority: declared,
			Operation:         operation,
			ReasonCode:        ReasonCodeUnknownAuthority,
			Message:           "unknown authority: " + string(declared),
		}
	}
}

// reasonForEnvironment maps an environment kind to the canonical denial
// reason code. The unknown case is the most permissive of the denied
// kinds; the GitHub Actions and CI cases carry their own codes so
// diagnostics can distinguish CI attempts from generic unknowns.
func reasonForEnvironment(env ExecutionEnvironmentKind) string {
	switch env {
	case EnvironmentGitHubActions:
		return ReasonCodeEnvironmentGitHubActions
	case EnvironmentCI:
		return ReasonCodeEnvironmentCI
	case EnvironmentUnknown:
		return ReasonCodeEnvironmentUnknown
	default:
		return ReasonCodeEnvironmentNotExplicitLocal
	}
}

// messageForEnvironment renders a stable denial message that names the
// declared authority, the operation, and the classified environment.
func messageForEnvironment(
	declared ExecutionAuthority,
	operation VerifierOperation,
	environment ExecutionEnvironmentKind,
) string {
	return "operation " + string(operation) +
		" under declared authority " + string(declared) +
		" is denied: classified environment is " + string(environment) +
		" (explicit local required)"
}
