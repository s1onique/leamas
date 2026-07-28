// SPDX-License-Identifier: Apache-2.0

// Package protectedverifier provides the canonical adapter for protected dupcode capabilities.
//
// This package is the ONLY production package allowed to call protected dupcode
// capabilities (internal/factory/dupcode). All other production code must route
// through this adapter.
//
// The adapter is invoked by the verifierdispatch.Dispatcher only after authority
// validation passes. This ensures that:
//   - Protected dupcode operations are never started without CI-exact-checkout authority
//   - Local execution is always denied
//   - Baseline mutations require appropriate authority
package protectedverifier
