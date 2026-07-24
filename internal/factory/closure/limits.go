// SPDX-License-Identifier: Apache-2.0

package closure

const (
	MaxPlanBytes           = 1 << 20
	MaxManifestBytes       = 8 << 20
	MaxChecks              = 10_000
	MaxArtifacts           = 10_000
	MaxArgvElements        = 1_024
	MaxEnvironmentEntries  = 1_024
	MaxJSONDepth           = 128
	MaxReportBytes         = 32 << 10
	MaxReportLines         = 200
	MaxCheckTimeoutSeconds = 600

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
