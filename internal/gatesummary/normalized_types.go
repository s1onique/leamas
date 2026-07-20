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
// Returns zero value for unexpected input (should not occur for valid decoded v2).
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
	case "skip":
		return GateSkip
	case "unavailable":
		return GateUnavailable
	}
	return ""
}

// Scope represents the bounded child scope in v2.
type Scope struct {
	ID          string
	Status      LifecycleStatus
	Disposition string
}

// Parent represents the parent ACT in v2.
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

// CheckExecution represents per-check process execution evidence.
type CheckExecution struct {
	Argv         []string
	ExitCode     *Integer
	StdoutSHA256 string
	StderrSHA256 string
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
type Check struct {
	Name       string
	Scope      *string
	Status     GateStatus
	Evidence   *string
	Detail     *string
	DurationMs *Integer
	Execution  *CheckExecution
	Totals     *TestTotals
}

// Summary is the common normalized domain model for both v1 and v2.
// All slices and pointers are newly owned; no aliasing with decoder state.
type Summary struct {
	SchemaVersion Version
	GeneratedAt   string

	Tool      *string
	Scope     *Scope
	Parent    *Parent
	Overall   Overall
	Execution *ExecutionBinding
	Worktree  *WorktreeState

	Checks []Check
}

// Valid reports whether the summary has a known schema version.
func (s Summary) Valid() bool {
	return s.SchemaVersion == Version1 || s.SchemaVersion == Version2
}
