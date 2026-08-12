package gatesummary

// V3Summary mirrors the InDeep Factory Gate Summary v3 wire contract.
//
// V3 is a strict superset of v2: every v2 field is present in v3, and
// v3 additionally carries:
//
//   - The top-level `counts` aggregate block (total/pass/fail/timeout/
//     skip/unavailable) and four parallel name lists (failed_names/
//     timeout_names/skipped_names/unavailable_names) so consumers can
//     reproduce the verdict set without scanning the full check array.
//
//   - Three top-level SHA-256 evidence bindings (registry_sha256,
//     events_sha256, transcript_sha256) that pin the document to the
//     exact registry, JSONL event stream, and raw transcript used to
//     generate it.
//
//   - Per-check evidence identifiers (id/order/execution_class/argv/
//     implementation) and the runner's deadline-exceeded flag.
//
//   - Both runner-classified and raw process exit codes
//     (canonical_exit_code/raw_process_exit_code) and the captured
//     termination signal so consumers can distinguish a graceful
//     non-zero exit from a runner-imposed timeout or a signal kill.
//
//   - Byte-exact stdout/stderr metrics and truncation flags so
//     consumers can detect bounded-writer truncation.
//
// Exit codes use WireInteger (nullable) so JSON null survives decoding;
// the duration field is a non-nullable WireInteger because the schema
// requires it.
type V3Summary struct {
	SchemaVersion       int       `json:"schema_version"`
	GeneratedAt         string    `json:"generated_at"`
	ScopeID             string    `json:"scope_id"`
	ScopeStatus         string    `json:"scope_status"`
	ScopeDisposition    string    `json:"scope_disposition"`
	ParentAct           string    `json:"parent_act"`
	ParentStatus        string    `json:"parent_status"`
	ParentDisposition   string    `json:"parent_disposition"`
	OverallStatus       string    `json:"overall_status"`
	OverallDisposition  string    `json:"overall_disposition"`
	ExecutionHeadOID    string    `json:"execution_head_oid"`
	ExecutionTreeOID    string    `json:"execution_tree_oid"`
	SubjectTreeOID      string    `json:"subject_tree_oid"`
	WorktreeCleanBefore bool      `json:"worktree_clean_before"`
	WorktreeCleanAfter  bool      `json:"worktree_clean_after"`
	RegistrySHA256      string    `json:"registry_sha256"`
	EventsSHA256        string    `json:"events_sha256"`
	TranscriptSHA256    string    `json:"transcript_sha256"`
	Counts              V3Counts  `json:"counts"`
	FailedNames         []string  `json:"failed_names"`
	TimeoutNames        []string  `json:"timeout_names"`
	SkippedNames        []string  `json:"skipped_names"`
	UnavailableNames    []string  `json:"unavailable_names"`
	Checks              []V3Check `json:"checks"`
}

// V3Counts is the canonical aggregate count block. The InDeep v3 schema
// declares these as the wire names total/pass/fail/timeout/skip/
// unavailable (all non-negative integers).
type V3Counts struct {
	Total       int `json:"total"`
	Pass        int `json:"pass"`
	Fail        int `json:"fail"`
	Timeout     int `json:"timeout"`
	Skip        int `json:"skip"`
	Unavailable int `json:"unavailable"`
}

// V3Check is the v3 per-check wire record.
//
// The v3 schema requires every field; no `omitempty` is used. WireInteger
// is used for non-nullable integer fields (order/duration_ms), and
// nullable pointers are used for exit codes so JSON null survives
// decoding (skip/null processes have no canonical exit code).
type V3Check struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Scope             string       `json:"scope"`
	Order             WireInteger  `json:"order"`
	ExecutionClass    string       `json:"execution_class"`
	Status            string       `json:"status"`
	Detail            string       `json:"detail"`
	Argv              []string     `json:"argv"`
	Implementation    string       `json:"implementation"`
	DurationMs        WireInteger  `json:"duration_ms"`
	DeadlineExceeded  bool         `json:"deadline_exceeded"`
	CanonicalExitCode *WireInteger `json:"canonical_exit_code"`
	RawProcessExit    *WireInteger `json:"raw_process_exit_code"`
	TerminationSignal *string      `json:"termination_signal"`
	StdoutSHA256      string       `json:"stdout_sha256"`
	StderrSHA256      string       `json:"stderr_sha256"`
	StdoutBytes       int64        `json:"stdout_bytes"`
	StderrBytes       int64        `json:"stderr_bytes"`
	StdoutTruncated   bool         `json:"stdout_truncated"`
	StderrTruncated   bool         `json:"stderr_truncated"`
}
