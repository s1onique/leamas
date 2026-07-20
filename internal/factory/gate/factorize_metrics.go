// Package gate provides opt-in factorize metrics collection.
//
// This module adds machine-readable per-verifier metrics to the factorize
// command when LEAMAS_FACTORIZE_METRICS_FILE is set. Metrics are written
// atomically to avoid partial artifacts. When the environment variable is
// absent, behaviour is identical to the existing factorize implementation.
package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/s1onique/leamas/internal/factory/checks"
)

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

// commandFingerprint computes a digest of a verifier's execution definition.
// It binds the verifier ID, executable identity, argv, and selected environment.
func commandFingerprint(name string, root string, argv []string, env []string, execPath string) string {
	h := sha256.New()
	h.Write([]byte("factorize-v1"))
	h.Write([]byte{0})
	h.Write([]byte(name))
	h.Write([]byte{0})
	// Bind executable identity
	h.Write([]byte(execPath))
	h.Write([]byte{0})
	// Bind argv
	for _, arg := range argv {
		h.Write([]byte(arg))
		h.Write([]byte{0})
	}
	h.Write([]byte{0}) // argv terminator
	// Bind selected environment (only LEAMAS_* vars that affect execution)
	for _, e := range env {
		if len(e) > 8 && e[:8] == "LEAMAS_" {
			h.Write([]byte(e))
			h.Write([]byte{0})
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// collectRusage collects resource usage for the current process.
func collectRusage() rusageMetrics {
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err != nil {
		return rusageMetrics{}
	}
	return rusageMetrics{
		userCPU:   rusage.Utime.Nano(),
		systemCPU: rusage.Stime.Nano(),
		maxRSS:    int64(rusage.Maxrss) * 1024,
	}
}

// buildEnvironment captures the measurement environment.
func buildEnvironment() MetricsEnvironment {
	env := MetricsEnvironment{
		GoVersion:           runtime.Version(),
		GoOS:                runtime.GOOS,
		GoArch:              runtime.GOARCH,
		GoMaxProcs:          runtime.GOMAXPROCS(0),
		LogicalCPUCount:     runtime.NumCPU(),
		MeasurementHostClass: "development-workstation",
	}
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range splitLines(string(data)) {
			if len(line) > 6 && line[:6] == "PRETTY" {
				env.OSRelease = line
				break
			}
		}
	}
	return env
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// metricsFilePath returns the configured metrics file path or empty string.
func metricsFilePath() string {
	return os.Getenv("LEAMAS_FACTORIZE_METRICS_FILE")
}

// shouldCollectMetrics returns true if metrics collection is enabled.
func shouldCollectMetrics() bool {
	return metricsFilePath() != ""
}

// writeMetrics atomically writes metrics to the specified path.
func writeMetrics(path string, m *FactorizeMetrics) error {
	if path == "" {
		return fmt.Errorf("metrics file path is empty")
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp metrics file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	return nil
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

// AddCheck records metrics for a single verifier.
func (mc *MetricsCollection) AddCheck(
	name string,
	ordinal int,
	findings []checks.Finding,
	duration time.Duration,
	rusage rusageMetrics,
	root string,
	cacheObservation string,
	argv []string,
	env []string,
	execPath string,
) {
	var status string
	var exitCode int
	if len(findings) > 0 {
		status = "fail"
		exitCode = 1
	} else {
		status = "pass"
		exitCode = 0
	}
	var userCPU, systemCPU, maxRSS *int64
	if rusage.userCPU > 0 {
		v := rusage.userCPU
		userCPU = &v
	}
	if rusage.systemCPU > 0 {
		v := rusage.systemCPU
		systemCPU = &v
	}
	if rusage.maxRSS > 0 {
		v := rusage.maxRSS
		maxRSS = &v
	}
	mc.Checks = append(mc.Checks, MetricsCheck{
		Ordinal:            ordinal,
		ID:                 name,
		Status:             status,
		ExitCode:           exitCode,
		DurationNs:         duration.Nanoseconds(),
		UserCPUNs:          userCPU,
		SystemCPUNs:        systemCPU,
		MaxRSSBytes:        maxRSS,
		ResourceScope:      "verifier",
		CommandFingerprint: commandFingerprint(name, root, argv, env, execPath),
		CacheObservation:   cacheObservation,
	})
}

// FinalizeRun completes the metrics collection and writes the artifact.
func (mc *MetricsCollection) FinalizeRun(
	status string,
	exitCode int,
	totalDuration time.Duration,
	rusage rusageMetrics,
	subject MetricsSubject,
	scenario string,
	sequence int,
) error {
	var userCPU, systemCPU, maxRSS *int64
	if rusage.userCPU > 0 {
		v := rusage.userCPU
		userCPU = &v
	}
	if rusage.systemCPU > 0 {
		v := rusage.systemCPU
		systemCPU = &v
	}
	if rusage.maxRSS > 0 {
		v := rusage.maxRSS
		maxRSS = &v
	}
	m := &FactorizeMetrics{
		Schema:      MetricsSchema,
		Subject:     subject,
		Environment: buildEnvironment(),
		Run: MetricsRun{
			Scenario:      scenario,
			Sequence:     sequence,
			StartedAt:    mc.StartTime.UTC().Format(time.RFC3339),
			Status:       status,
			ExitCode:     exitCode,
			DurationNs:   totalDuration.Nanoseconds(),
			UserCPUNs:    userCPU,
			SystemCPUNs:  systemCPU,
			MaxRSSBytes:  maxRSS,
			ResourceScope: "full-run",
		},
		Checks: mc.Checks,
	}
	return writeMetrics(mc.Path, m)
}
