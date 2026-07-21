// Package gate provides opt-in factorize metrics collection with v3 contract.
package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// NewMetricsCollectionV3 creates a new metrics collection from environment variables.
// Returns nil if metrics collection is not enabled.
func NewMetricsCollectionV3(path, scenario, sequence string) (*MetricsCollectionV3, error) {
	if path == "" {
		return nil, nil
	}

	if scenario == "" {
		return nil, fmt.Errorf("LEAMAS_FACTORIZE_SCENARIO is required when LEAMAS_FACTORIZE_METRICS_FILE is set")
	}

	if !ValidScenarios[scenario] {
		return nil, fmt.Errorf("unknown scenario %q: must be one of controlled-go-cache-cold, controlled-warm, native-reference", scenario)
	}

	if sequence == "" {
		return nil, fmt.Errorf("LEAMAS_FACTORIZE_SEQUENCE is required when LEAMAS_FACTORIZE_METRICS_FILE is set")
	}

	seq, err := parsePositiveInt(sequence)
	if err != nil {
		return nil, fmt.Errorf("invalid sequence %q: %w", sequence, err)
	}

	mc := &MetricsCollectionV3{
		Path:      path,
		Scenario:  scenario,
		Sequence:  seq,
		StartTime: time.Now(),
		Host:      buildHostIdentity(),
	}

	return mc, nil
}

func parsePositiveInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("must be a positive integer")
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return 0, fmt.Errorf("must be a positive integer")
	}
	return n, nil
}

// buildHostIdentity captures the measurement environment.
func buildHostIdentity() HostIdentity {
	return HostIdentity{
		GoVersion:       runtime.Version(),
		GOOS:            runtime.GOOS,
		GOARCH:          runtime.GOARCH,
		GOMAXPROCS:      runtime.GOMAXPROCS(0),
		LogicalCPUCount: runtime.NumCPU(),
		TotalMemoryKB:   collectTotalMemory(),
		Kernel:          runtime.GOOS,
		OSRelease:       readOSRelease(),
	}
}

func readOSRelease() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if len(line) > 6 && line[:6] == "PRETTY" {
			return line
		}
	}
	return ""
}

// collectTotalMemory attempts to read total system memory in KB.
func collectTotalMemory() int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	const prefix = "MemTotal:"
	for _, line := range strings.Split(string(data), "\n") {
		if len(line) >= len(prefix) && line[:len(prefix)] == prefix {
			var kb int64
			for _, c := range line[len(prefix):] {
				if c >= '0' && c <= '9' {
					kb = kb*10 + int64(c-'0')
				}
			}
			return kb
		}
	}
	return 0
}

// SetSubjectIdentity populates the subject identity fields.
func (mc *MetricsCollectionV3) SetSubjectIdentity(headOID, treeOID, worktreeState, subjectDigest string) {
	mc.HeadOID = headOID
	mc.TreeOID = treeOID
	mc.WorktreeState = worktreeState
	mc.SubjectInputDigest = subjectDigest

	mc.RunID = fmt.Sprintf("%s:%s:%d", subjectDigest, mc.Scenario, mc.Sequence)
}

// AddCheckWithResources records metrics when resources are provided directly.
func (mc *MetricsCollectionV3) AddCheckWithResources(
	verifier Verifier,
	ordinal int,
	findings []checks.Finding,
	duration time.Duration,
	userDelta, systemDelta time.Duration,
	maxRSSKB int64,
	root string,
	env []string,
) error {
	if userDelta < 0 {
		return fmt.Errorf("negative user CPU delta for %s: %v", verifier.Name, userDelta)
	}
	if systemDelta < 0 {
		return fmt.Errorf("negative system CPU delta for %s: %v", verifier.Name, systemDelta)
	}

	status := "pass"
	exitCode := 0
	if len(findings) > 0 {
		status = "fail"
		exitCode = 1
	}

	fingerprint, err := executionFingerprintV3(verifier.Name, verifier.Execution, env)
	if err != nil {
		return fmt.Errorf("execution fingerprint for %s: %w", verifier.Name, err)
	}

	mc.Checks = append(mc.Checks, MetricsCheckV3{
		Ordinal:            ordinal,
		ID:                 verifier.Name,
		Status:             status,
		ExitCode:           exitCode,
		DurationNs:         duration.Nanoseconds(),
		Resources:          buildResourceObservation(userDelta, systemDelta, maxRSSKB),
		CommandFingerprint: fingerprint,
		Cache:              verifier.Cache,
	})
	return nil
}

func buildResourceObservation(userDelta, systemDelta time.Duration, maxRSSKB int64) ResourceObservation {
	return ResourceObservation{
		Status:                    "available",
		Scope:                     "process-self-pre-post-delta",
		UserCPUNanosecondsDelta:   userDelta.Nanoseconds(),
		SystemCPUNanosecondsDelta: systemDelta.Nanoseconds(),
		ProcessMaxRSSKBAfter:      maxRSSKB,
		MemoryScope:               "process-high-water-after-check",
	}
}

// Finalize completes the metrics collection and writes the artifact.
// Returns error if validation fails or publication fails (fail-closed).
func (mc *MetricsCollectionV3) Finalize(failed bool) error {
	if err := mc.validateReconciliation(); err != nil {
		return fmt.Errorf("metrics reconciliation: %w", err)
	}

	doc := FactorizeMetricsV3{
		Schema:             MetricsSchema,
		GeneratedAt:        time.Now().UTC().Format(time.RFC3339Nano),
		HeadOID:            mc.HeadOID,
		TreeOID:            mc.TreeOID,
		WorktreeState:      mc.WorktreeState,
		SubjectInputDigest: mc.SubjectInputDigest,
		Scenario:           mc.Scenario,
		Sequence:           mc.Sequence,
		RunID:              mc.RunID,
		Host:               mc.Host,
		Checks:             mc.Checks,
		ChecksTotal:        len(mc.Checks),
		ChecksPassed:       countPassedV3(mc.Checks),
		ChecksFailed:       countFailedV3(mc.Checks),
		// Evidence is complete if all expected checks were recorded
		Complete: len(mc.Checks) > 0,
	}

	return PublishMetricsV3(mc.Path, &doc)
}

func (mc *MetricsCollectionV3) validateReconciliation() error {
	if len(mc.Checks) == 0 {
		return fmt.Errorf("no checks recorded")
	}

	ordinals := make(map[int]bool)
	for _, c := range mc.Checks {
		if ordinals[c.Ordinal] {
			return fmt.Errorf("duplicate ordinal %d", c.Ordinal)
		}
		ordinals[c.Ordinal] = true
	}

	ids := make(map[string]bool)
	for _, c := range mc.Checks {
		if ids[c.ID] {
			return fmt.Errorf("duplicate verifier ID %q", c.ID)
		}
		ids[c.ID] = true
	}

	for i := 1; i <= len(mc.Checks); i++ {
		if !ordinals[i] {
			return fmt.Errorf("missing ordinal %d", i)
		}
	}

	// Validate arithmetic: passed + failed = total
	passed := countPassedV3(mc.Checks)
	failed := countFailedV3(mc.Checks)
	if passed+failed != len(mc.Checks) {
		return fmt.Errorf("checks_failed + checks_passed (%d) != checks_total (%d)", passed+failed, len(mc.Checks))
	}

	return nil
}

func countPassedV3(checks []MetricsCheckV3) int {
	n := 0
	for _, c := range checks {
		if c.Status == "pass" {
			n++
		}
	}
	return n
}

func countFailedV3(checks []MetricsCheckV3) int {
	n := 0
	for _, c := range checks {
		if c.Status == "fail" {
			n++
		}
	}
	return n
}

// FingerprintError represents an error in computing a command fingerprint.
type FingerprintError struct {
	Reason string
}

func (e *FingerprintError) Error() string {
	return "command fingerprint error: " + e.Reason
}

// executionFingerprintV3 computes a digest of a verifier's execution definition.
func executionFingerprintV3(name string, exec ExecutionDefinition, env []string) (string, error) {
	if name == "" {
		return "", &FingerprintError{Reason: "verifier name is required"}
	}
	if exec.ImplementationID == "" {
		return "", &FingerprintError{Reason: "implementation ID is required"}
	}

	h := sha256.New()
	h.Write([]byte("factorize-v3"))
	h.Write([]byte{0})
	h.Write([]byte(name))
	h.Write([]byte{0})
	h.Write([]byte(exec.Kind))
	h.Write([]byte{0})
	h.Write([]byte(exec.ImplementationID))
	h.Write([]byte{0})

	present := make(map[string]string)
	for _, e := range env {
		if idx := strings.IndexByte(e, '='); idx >= 0 {
			present[e[:idx]] = e[idx+1:]
		}
	}

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
