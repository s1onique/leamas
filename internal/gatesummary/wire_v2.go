package gatesummary

// V2Summary mirrors the frozen v2 wire contract described in
// gate-summary-v2-spec.md. Optional fields use pointer types when
// absent-vs-present matters; exit_code uses a nullable WireInteger
// so JSON null survives decoding.
type V2Summary struct {
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
	Checks              []V2Check `json:"checks"`
}

// V2Check is the v2 per-check wire record.
type V2Check struct {
	Name             string       `json:"name"`
	Scope            string       `json:"scope"`
	Status           string       `json:"status"`
	Evidence         string       `json:"evidence"`
	Detail           string       `json:"detail"`
	Extras           V2Extras     `json:"extras"`
	Total            *WireInteger `json:"total,omitempty"`
	PassCount        *WireInteger `json:"pass_count,omitempty"`
	FailCount        *WireInteger `json:"fail_count,omitempty"`
	SkipCount        *WireInteger `json:"skip_count,omitempty"`
	UnavailableCount *WireInteger `json:"unavailable_count,omitempty"`
}

// V2Extras carries per-check process execution evidence.
type V2Extras struct {
	Argv         []string     `json:"argv"`
	ExitCode     *WireInteger `json:"exit_code"`
	DurationMs   WireInteger  `json:"duration_ms"`
	StdoutSHA256 string       `json:"stdout_sha256"`
	StderrSHA256 string       `json:"stderr_sha256"`
}
