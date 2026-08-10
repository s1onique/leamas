// SPDX-License-Identifier: Apache-2.0

// Package closure - limits.go is the B2-R7 single-source
// limits surface. The Plan-Contract-related limits
// (MaxChecks, MaxArtifacts, MaxArgvElements,
// MaxEnvironmentEntries, MaxCheckTimeoutSeconds) are
// aliases of the canonical plancontract values; no
// closure file may carry a Plan-Contract limit literal.
//
// Non-Plan-Contract limits (manifest size, sidecar size,
// report size, JSON depth, etc.) remain closure-local
// because they describe the closure package's own
// bounded-output surface and not the wire contract.
package closure

import "github.com/s1onique/leamas/internal/factory/closure/plancontract"

const (
	MaxPlanBytes           = plancontract.MaxPlanBytes
	MaxManifestBytes       = 8 << 20
	MaxChecks              = plancontract.MaxChecks
	MaxArtifacts           = plancontract.MaxArtifacts
	MaxArgvElements        = plancontract.MaxArgvElements
	MaxEnvironmentEntries  = plancontract.MaxEnvironmentEntries
	MaxJSONDepth           = 128
	MaxReportBytes         = 32 << 10
	MaxReportLines         = 200
	MaxCheckTimeoutSeconds = plancontract.MaxCheckTimeoutSeconds

	// Slice 4 — bounded core manifest + detached evidence. The numbers
	// below are pinned so the core manifest stays a small summary
	// record and sidecar evidence stays reviewable by an LLM.
	CoreManifestMaxLines          = 400
	CoreManifestMaxBytes          = 32 << 10
	CoreManifestMaxNestingDepth   = 6
	SidecarPerFileMaxBytes        = 2 << 20
	SidecarTotalMaxBytes          = 16 << 20
	SidecarMaxRecordCount         = 10_000
	SidecarMaxStringLength        = 4 << 10
	SidecarMaxStdoutMetadataBytes = 4 << 10
)
