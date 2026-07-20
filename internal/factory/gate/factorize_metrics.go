// Package gate provides opt-in factorize metrics collection.
package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// MetricsSchema is the schema identifier for factorize metrics.
const MetricsSchema = "factorize-performance-v2"

// FingerprintError represents an error in computing a command fingerprint.
type FingerprintError struct {
	Reason string
}

func (e *FingerprintError) Error() string {
	return "command fingerprint error: " + e.Reason
}

// executionFingerprint computes a digest of a verifier's execution definition.
// It canonicalizes declared environment keys with presence/absence markers.
func executionFingerprint(name string, exec ExecutionDefinition, env []string) (string, error) {
	if name == "" {
		return "", &FingerprintError{Reason: "verifier name is required"}
	}
	if exec.ImplementationID == "" {
		return "", &FingerprintError{Reason: "implementation ID is required"}
	}

	h := sha256.New()
	h.Write([]byte("factorize-v2"))
	h.Write([]byte{0})
	h.Write([]byte(name))
	h.Write([]byte{0})
	h.Write([]byte(exec.Kind))
	h.Write([]byte{0})
	h.Write([]byte(exec.ImplementationID))
	h.Write([]byte{0})

	// Build a map of present environment values
	present := make(map[string]string)
	for _, e := range env {
		if idx := strings.IndexByte(e, '='); idx >= 0 {
			present[e[:idx]] = e[idx+1:]
		}
	}

	// Hash declared environment keys with canonical ordering and presence markers
	sortedKeys := make([]string, len(exec.EnvVars))
	copy(sortedKeys, exec.EnvVars)
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		h.Write([]byte(key))
		h.Write([]byte{0})
		if val, ok := present[key]; ok {
			h.Write([]byte("present"))
			h.Write([]byte{0})
			h.Write([]byte(val))
			h.Write([]byte{0})
		} else {
			h.Write([]byte("absent"))
			h.Write([]byte{0})
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
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
		GoVersion:            runtime.Version(),
		GoOS:                 runtime.GOOS,
		GoArch:               runtime.GOARCH,
		GoMaxProcs:           runtime.GOMAXPROCS(0),
		LogicalCPUCount:      runtime.NumCPU(),
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

// AddCheck records metrics for a single verifier using authoritative verifier metadata.
func (mc *MetricsCollection) AddCheck(
	verifier Verifier,
	ordinal int,
	findings []checks.Finding,
	duration time.Duration,
	rusage rusageMetrics,
	root string,
	env []string,
) error {
	status := "pass"
	exitCode := 0
	if len(findings) > 0 {
		status = "fail"
		exitCode = 1
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

	fingerprint, err := executionFingerprint(verifier.Name, verifier.Execution, env)
	if err != nil {
		return fmt.Errorf("execution fingerprint for %s: %w", verifier.Name, err)
	}

	mc.Checks = append(mc.Checks, MetricsCheck{
		Ordinal:            ordinal,
		ID:                 verifier.Name,
		Status:             status,
		ExitCode:           exitCode,
		DurationNs:         duration.Nanoseconds(),
		UserCPUNs:          userCPU,
		SystemCPUNs:        systemCPU,
		MaxRSSBytes:        maxRSS,
		ResourceScope:      "verifier",
		CommandFingerprint: fingerprint,
		Cache:              verifier.Cache,
	})
	return nil
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
			Sequence:      sequence,
			StartedAt:     mc.StartTime.UTC().Format(time.RFC3339),
			Status:        status,
			ExitCode:      exitCode,
			DurationNs:    totalDuration.Nanoseconds(),
			UserCPUNs:     userCPU,
			SystemCPUNs:   systemCPU,
			MaxRSSBytes:   maxRSS,
			ResourceScope: "full-run",
		},
		Checks: mc.Checks,
	}
	return writeMetrics(mc.Path, m)
}
