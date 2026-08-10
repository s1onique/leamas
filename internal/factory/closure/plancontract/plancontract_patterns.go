// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_patterns.go exposes
// the canonical regex patterns and placeholder detector
// the closure package uses as thin aliases.
//
// B2-R7 single-source rule: every pattern, limit, and
// placeholder detection rule for the Plan Contract v1
// wire format lives in this leaf. The closure package
// references these via the canonical exports; no closure
// file may carry a duplicate pattern literal.
package plancontract

import "regexp"

// ActIDPattern is the canonical regex the leaf enforces
// on the act_id field. The closure package aliases this
// value; any drift is a contract bug caught by the
// execution/evidence parity matrix.
var ActIDPattern = regexp.MustCompile(`^ACT-[A-Z0-9][A-Z0-9-]{2,199}$`)

// ItemIDPattern is the canonical regex the leaf enforces
// on every item-id field (check id, artifact id, ...).
var ItemIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)

// OIDPattern is the canonical regex the leaf enforces on
// every Git OID field (baseline.commit_oid, baseline.tree_oid,
// runner_authority.tool.revision, ...).
var OIDPattern = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// EnvironmentNamePattern is the canonical regex the leaf
// enforces on every environment-map key.
var EnvironmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ContainsClosurePlaceholder is the canonical
// placeholder detector. The closure package aliases this
// function so callers do not duplicate the placeholder
// rule. A true return means the value carries a closure
// placeholder and the leaf MUST reject it on the
// wire-contract path.
func ContainsClosurePlaceholder(value string) bool {
	return containsClosurePlaceholder(value)
}
