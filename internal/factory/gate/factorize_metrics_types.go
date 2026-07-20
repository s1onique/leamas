// Package gate provides opt-in factorize metrics collection.
package gate

import "time"

// MetricsSchema is the schema identifier for factorize metrics.
const MetricsSchema = "factorize-performance-v1"

// MetricsEnvironment represents the measurement environment.
type MetricsEnvironment struct {
	GoVersion           string `json:"go_version"`
	GoOS                string `json:"goos"`
	GoArch              string `json:"goarch"`
	GoMaxProcs          int    `json:"gomaxprocs"`
	LogicalCPUCount     int    `json:"logical_cpu_count"`
	OSRelease           string `json:"os_release"`
	MeasurementHostClass string `json:"measurement_host_class"`
}

// MetricsRun represents a single factorize run.
type MetricsRun struct {
	Scenario      string `json:"scenario"`
	Sequence      int    `json:"sequence"`
	StartedAt     string `json:"started_at"`
	Status        string `json:"status"`
	ExitCode      int    `json:"exit_code"`
	DurationNs    int64  `json:"duration_ns"`
	UserCPUNs     *int64 `json:"user_cpu_ns"`
	SystemCPUNs   *int64 `json:"system_cpu_ns"`
	MaxRSSBytes   *int64 `json:"max_rss_bytes"`
	ResourceScope string `json:"resource_scope"`
}

// MetricsCheck represents a single verifier check result.
type MetricsCheck struct {
	Ordinal            int    `json:"ordinal"`
	ID                 string `json:"id"`
	Status             string `json:"status"`
	ExitCode           int    `json:"exit_code"`
	DurationNs         int64  `json:"duration_ns"`
	UserCPUNs          *int64 `json:"user_cpu_ns"`
	SystemCPUNs        *int64 `json:"system_cpu_ns"`
	MaxRSSBytes        *int64 `json:"max_rss_bytes"`
	ResourceScope      string `json:"resource_scope"`
	CommandFingerprint string `json:"command_fingerprint"`
	CacheObservation   string `json:"cache_observation"`
}

// FactorizeMetrics is the top-level metrics document.
type FactorizeMetrics struct {
	Schema      string           `json:"schema"`
	Subject     MetricsSubject   `json:"subject"`
	Environment MetricsEnvironment `json:"environment"`
	Run         MetricsRun      `json:"run"`
	Checks      []MetricsCheck  `json:"checks"`
}

// MetricsSubject identifies the measurement subject.
type MetricsSubject struct {
	HeadOID           string `json:"head_oid"`
	TreeOID          string `json:"tree_oid"`
	WorktreeState    string `json:"worktree_state"`
	SubjectInputDigest string `json:"subject_input_digest"`
}

// rusageMetrics holds resource usage data.
type rusageMetrics struct {
	userCPU   int64
	systemCPU int64
	maxRSS    int64
}

// MetricsCollection holds metrics for a single factorize run.
type MetricsCollection struct {
	Checks    []MetricsCheck
	StartTime time.Time
	Path      string
}

// StartRun initializes a new metrics collection for a run.
func (mc *MetricsCollection) StartRun() {
	mc.Checks = nil
	mc.StartTime = time.Now()
}
