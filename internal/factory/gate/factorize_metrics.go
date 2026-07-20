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

// FingerprintError represents an error in computing a command fingerprint.
type FingerprintError struct {
	Reason string
}

func (e *FingerprintError) Error() string {
	return "command fingerprint error: " + e.Reason
}

// commandFingerprint computes a digest of a verifier's execution definition.
// It binds the verifier ID, argv, and execution-relevant environment.
// Returns an error if the execution definition is incomplete.
// The fingerprint is invariant under checkout relocation.
func commandFingerprint(name string, root string, argv []string, env []string, execPath string) (string, error) {
	if name == "" {
		return "", &FingerprintError{Reason: "verifier name is required"}
	}
	if len(argv) == 0 {
		return "", &FingerprintError{Reason: "argv is required"}
	}

	h := sha256.New()
	h.Write([]byte("factorize-v2"))
	h.Write([]byte{0})
	h.Write([]byte(name))
	h.Write([]byte{0})

	for _, arg := range argv {
		h.Write([]byte(arg))
		h.Write([]byte{0})
	}
	h.Write([]byte{0})

	var relevant []string
	for _, e := range env {
		if hasExecEnvPrefix(e) && isExecRelevantEnv(e) {
			relevant = append(relevant, e)
		}
	}
	sort.Strings(relevant)
	for _, e := range relevant {
		h.Write([]byte(e))
		h.Write([]byte{0})
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// hasExecEnvPrefix returns true if the env var starts with a known prefix.
func hasExecEnvPrefix(env string) bool {
	execPrefixes := []string{
		"GO", "CGO_", "GOPROXY", "GOSUMDB", "GOPRIVATE", "PATH",
	}
	for _, prefix := range execPrefixes {
		if strings.HasPrefix(env, prefix) {
			return true
		}
	}
	return false
}

// isExecRelevantEnv returns true if the env var affects verifier execution.
func isExecRelevantEnv(env string) bool {
	excluded := map[string]bool{
		"LEAMAS_FACTORIZE_METRICS_FILE": true,
		"LEAMAS_FACTORIZE_SCENARIO":    true,
		"LEAMAS_FACTORIZE_SEQUENCE":    true,
	}

	keyEnd := strings.IndexByte(env, '=')
	if keyEnd <= 0 {
		return false
	}
	key := env[:keyEnd]

	if excluded[key] {
		return false
	}

	execRelevant := map[string]bool{
		"GOFLAGS":     true,
		"GOCACHE":     true,
		"GOENV":       true,
		"GOTOOLCHAIN": true,
		"GOMAXPROCS":  true,
		"CGO_ENABLED": true,
		"GOOS":        true,
		"GOARCH":      true,
		"GOPROXY":     true,
		"GONOSUMDB":   true,
		"GOSUMDB":     true,
		"GOPRIVATE":   true,
		"PATH":        true,
	}

	return execRelevant[key]
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

// AddCheck records metrics for a single verifier.
// Returns an error if the command fingerprint cannot be computed.
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

	fingerprint, err := commandFingerprint(name, root, argv, env, execPath)
	if err != nil {
		return fmt.Errorf("command fingerprint for %s: %w", name, err)
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
		CommandFingerprint: fingerprint,
		CacheObservation:   cacheObservation,
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
