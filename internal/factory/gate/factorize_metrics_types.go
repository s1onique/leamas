// Package gate provides opt-in factorize metrics collection with v3 contract.
package gate

import "time"

// MetricsSchema is the schema identifier for factorize metrics v3.
const MetricsSchema = "factorize-performance-v3"

// ResourceSnapshot represents a point-in-time resource observation.
type ResourceSnapshot struct {
	UserCPU         time.Duration
	SystemCPU       time.Duration
	ProcessMaxRSSKB int64
}

// HostIdentity captures the measurement host characteristics.
type HostIdentity struct {
	GoVersion       string `json:"go_version"`
	GOOS            string `json:"goos"`
	GOARCH          string `json:"goarch"`
	GOMAXPROCS      int    `json:"gomaxprocs"`
	LogicalCPUCount int    `json:"logical_cpu_count"`
	TotalMemoryKB   int64  `json:"total_memory_kb"`
	Kernel          string `json:"kernel"`
	OSRelease       string `json:"os_release"`
}

// ResourceObservation represents resource usage for a single check.
type ResourceObservation struct {
	Status                    string `json:"status"`
	Scope                     string `json:"scope"`
	UserCPUNanosecondsDelta   int64  `json:"user_cpu_nanoseconds_delta"`
	SystemCPUNanosecondsDelta int64  `json:"system_cpu_nanoseconds_delta"`
	ProcessMaxRSSKBAfter      int64  `json:"process_max_rss_kb_after"`
	MemoryScope               string `json:"memory_scope"`
	UnavailableReason         string `json:"unavailable_reason,omitempty"`
}

// MetricsCheckV3 represents a single verifier check result in v3 format.
type MetricsCheckV3 struct {
	Ordinal            int                 `json:"ordinal"`
	ID                 string              `json:"id"`
	Status             string              `json:"status"`
	ExitCode           int                 `json:"exit_code"`
	DurationNs         int64               `json:"duration_ns"`
	Resources          ResourceObservation `json:"resources"`
	CommandFingerprint string              `json:"command_fingerprint"`
	Cache              CacheSemantics      `json:"cache"`
}

// FactorizeMetricsV3 is the top-level metrics document for v3.
type FactorizeMetricsV3 struct {
	Schema             string           `json:"schema"`
	GeneratedAt        string           `json:"generated_at"`
	HeadOID            string           `json:"head_oid"`
	TreeOID            string           `json:"tree_oid"`
	WorktreeState      string           `json:"worktree_state"`
	SubjectInputDigest string           `json:"subject_input_digest"`
	Scenario           string           `json:"scenario"`
	Sequence           int              `json:"sequence"`
	RunID              string           `json:"run_id"`
	Host               HostIdentity     `json:"host"`
	Checks             []MetricsCheckV3 `json:"checks"`
	ChecksTotal        int              `json:"checks_total"`
	ChecksPassed       int              `json:"checks_passed"`
	ChecksFailed       int              `json:"checks_failed"`
	Complete           bool             `json:"complete"`
}

// ValidScenarios contains the allowed scenario values.
var ValidScenarios = map[string]bool{
	"controlled-go-cache-cold": true,
	"controlled-warm":          true,
	"native-reference":         true,
}

// MetricsCollectionV3 holds metrics for a single factorize run.
type MetricsCollectionV3 struct {
	Checks    []MetricsCheckV3
	StartTime time.Time
	Path      string

	// Identity fields populated from environment
	Scenario string
	Sequence int
	RunID    string

	// Subject identity
	HeadOID            string
	TreeOID            string
	WorktreeState      string
	SubjectInputDigest string

	// Host identity captured at start
	Host HostIdentity

	// Error accumulator - if set, publication will fail
	err error
}
