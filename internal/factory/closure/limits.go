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
)
