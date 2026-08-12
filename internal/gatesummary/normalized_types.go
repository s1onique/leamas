package gatesummary

// GateStatus is the normalized machine-gate status for checks and overall.
// This is a strict subset vocabulary used for both per-check status and
// aggregate overall status in the normalized model.
type GateStatus string

const (
	GatePass        GateStatus = "pass"
	GateFail        GateStatus = "fail"
	GateSkip        GateStatus = "skip"
	GateUnavailable GateStatus = "unavailable"
	// GateTimeout signals a runner-imposed deadline expiry.
	// Distinct from GateFail because the underlying process
	// status may have been a voluntary exit. v3 introduces
	// timeout as a first-class per-check status; v1 and v2
	// do not use it.
	GateTimeout GateStatus = "timeout"
)

// LifecycleStatus is the normalized scope/parent lifecycle status.
// Wire form is uppercase; normalized form is lowercase.
type LifecycleStatus string

const (
	LifecycleOpen    LifecycleStatus = "open"
	LifecyclePartial LifecycleStatus = "partial"
	LifecycleClosed  LifecycleStatus = "closed"
)

// normalizeLifecycle converts uppercase wire form to normalized lowercase.
// Returns zero value for unexpected input (should not occur for valid decoded v2/v3).
func normalizeLifecycle(wire string) LifecycleStatus {
	switch wire {
	case "OPEN":
		return LifecycleOpen
	case "PARTIAL":
		return LifecyclePartial
	case "CLOSED":
		return LifecycleClosed
	}
	return ""
}

// wireToGateStatus converts wire string to GateStatus.
// Wire values are already validated by the decoder schema.
func wireToGateStatus(wire string) GateStatus {
	switch wire {
	case "pass":
		return GatePass
	case "fail":
		return GateFail
	case "timeout":
		return GateTimeout
	case "skip":
		return GateSkip
	case "unavailable":
		return GateUnavailable
	}
	return ""
}

// Scope represents the bounded child scope in v2/v3.
type Scope struct {
	ID          string
	Status      LifecycleStatus
	Disposition string
}

// Parent represents the parent ACT in v2/v3.
type Parent struct {
	Act         string
	Status      LifecycleStatus
	Disposition string
	Root        bool
}

// Overall represents the aggregate machine-gate status.
type Overall struct {
	Status      GateStatus
	Disposition *string
}

// ExecutionBinding represents Git execution identity.
type ExecutionBinding struct {
	HeadOID    string
	TreeOID    string
	SubjectOID string
}

// WorktreeState represents worktree cleanliness.
type WorktreeState struct {
	CleanBefore bool
	CleanAfter  bool
}

// EvidenceHashes binds a v3 summary to the exact evidence files used to
// generate it: the registry, the JSONL event stream, and the raw
// transcript. Empty values mean the producer did not bind a particular
// stream (typically because it never produced one); v3 consumers can
// fall back to other verification paths when a hash is empty.
type EvidenceHashes struct {
	RegistrySHA256   string
	EventsSHA256     string
	TranscriptSHA256 string
}

// Counts is the aggregate verdict set carried by every v3 document at
// the top level. The counts are authoritative; the v3 semantic
// validator re-derives them from Checks and rejects any mismatch.
type Counts struct {
	Total       int
	Pass        int
	Fail        int
	Timeout     int
	Skip        int
	Unavailable int
}

// CheckExecution represents per-check process execution evidence.
//
// V2 wire forms populate Argv/ExitCode/StdoutSHA256/StderrSHA256. V3
// additionally carries byte-exact stdout/stderr metrics and truncation
// flags (StdoutBytes/StderrBytes/StdoutTruncated/StderrTruncated),
// the runner-classified deadline-exceeded flag, the runner's raw
// process exit code, the captured termination signal, and the canonical
// argv sha256 implementation binding.
type CheckExecution struct {
	Argv              []string
	ExitCode          *Integer
	RawExitCode       *Integer
	StdoutSHA256      string
	StderrSHA256      string
	StdoutBytes       int64
	StderrBytes       int64
	StdoutTruncated   bool
	StderrTruncated   bool
	DeadlineExceeded  bool
	TerminationSignal *string
	Implementation    string
}

// TestTotals represents optional per-check test arithmetic.
type TestTotals struct {
	Total       Integer
	Pass        Integer
	Fail        Integer
	Skip        Integer
	Unavailable Integer
}

// Check represents a normalized check entry.
//
// V3 wires additionally carry ID/Order/ExecutionClass on the check;
// v1/v2 wires leave these nil.
type Check struct {
	ID             *string
	Name           string
	Scope          *string
	Order          *Integer
	ExecutionClass *string
	Status         GateStatus
	Evidence       *string
	Detail         *string
	DurationMs     *Integer
	Execution      *CheckExecution
	Totals         *TestTotals
}

// Summary is the common normalized domain model for v1, v2, and v3.
// All slices and pointers are newly owned; no aliasing with decoder state.
//
// Counts, EvidenceHashes, the per-check ID/Order/ExecutionClass
// fields, and the four parallel name arrays are only populated for
// v3 documents. The semantic validator (`validateV3NameLists`)
// cross-checks the name arrays against the actual per-check
// statuses.
type Summary struct {
	SchemaVersion   Version
	GeneratedAt     string

	Tool            *string
	Scope           *Scope
	Parent          *Parent
	Overall         Overall
	Execution       *ExecutionBinding
	Worktree        *WorktreeState
	EvidenceHashes  *EvidenceHashes
	Counts          Counts
	FailedNames     []string
	TimeoutNames    []string
	SkippedNames    []string
	UnavailableNames []string

	Checks []Check
}

// Valid reports whether the summary has a known schema version.
func (s Summary) Valid() bool {
	return s.SchemaVersion == Version1 ||
		s.SchemaVersion == Version2 ||
		s.SchemaVersion == Version3
}
