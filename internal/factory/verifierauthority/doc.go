// SPDX-License-Identifier: Apache-2.0

// Package verifierauthority provides the generic verifier execution authority model.
//
// This package implements a capability-level authority model that governs verifier
// execution. It replaces command-specific guards with a generic authority classification.
//
// Authority classifications:
//   - AuthorityLocalSafe: permitted locally
//   - AuthorityCIExactCheckout: permitted only in authorized GitHub Actions exact-checkout context
//
// The package uses the bounded execution.RunGit for all Git observations to ensure
// fail-closed behavior on Git failures.
package verifierauthority
