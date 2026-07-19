package gatesummary

// V1Summary mirrors the frozen v1 wire contract described in
// gate-summary-v1-spec.md. Optional fields use json.Number or pointer
// types so absent-versus-present distinctions are preserved.
type V1Summary struct {
	SchemaVersion int       `json:"schema_version"`
	GeneratedAt   string    `json:"generated_at"`
	Tool          *string   `json:"tool,omitempty"`
	OverallStatus string    `json:"overall_status"`
	Checks        []V1Check `json:"checks"`
}

// V1Check is the v1 per-check wire record.
type V1Check struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	DurationMs *int64  `json:"duration_ms,omitempty"`
	Evidence   *string `json:"evidence,omitempty"`
}
