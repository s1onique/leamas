// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_patterns.go is the B2-R7 single-
// source patterns and placeholder alias surface. The
// closure package references the canonical plancontract
// patterns and detector through these aliases; no closure
// file may carry a duplicate regex literal or a duplicate
// placeholder set.
//
// B2-R7 single-source rule: every Plan Contract v1 wire
// pattern and every closure-placeholder rule lives in the
// plancontract leaf. This file is the typed alias that
// keeps the closure package's existing API stable while
// making drift impossible.
package closure

import (
	"regexp"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// actIDPattern is the closure-package alias for
// plancontract.ActIDPattern. The closure runner code paths
// that previously declared this pattern locally now
// reference the canonical leaf value via this alias.
var actIDPattern = plancontract.ActIDPattern

// itemIDPattern is the closure-package alias for
// plancontract.ItemIDPattern.
var itemIDPattern = plancontract.ItemIDPattern

// oidPattern is the closure-package alias for
// plancontract.OIDPattern.
var oidPattern = plancontract.OIDPattern

// environmentNamePattern is the closure-package alias for
// plancontract.EnvironmentNamePattern.
var environmentNamePattern = plancontract.EnvironmentNamePattern

// containsClosurePlaceholder is the closure-package alias
// for plancontract.ContainsClosurePlaceholder. The
// underlying placeholder set lives exclusively in the
// plancontract leaf; this alias is a thin wrapper that
// preserves the closure package's existing call sites.
func containsClosurePlaceholder(value string) bool {
	return plancontract.ContainsClosurePlaceholder(value)
}

// Compile-time references ensure the canonical patterns
// remain compiled even when no closure file references
// the alias in a particular build target.
var (
	_ = regexp.MustCompile
	_ = plancontract.ActIDPattern
)
