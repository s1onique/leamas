//go:build unix || darwin || linux

package execution

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// readinessPollInterval is the poll interval used by waitForReadiness and
// waitForExpectedRoles. It is intentionally small so a readiness publication
// is observed within at most one interval of the file system update.
const readinessPollInterval = 10 * time.Millisecond

// newProcessVerifier creates a new process verifier for a test. The verifier
// owns a unique manifest file and a unique readiness directory. Both are
// removed by the returned cleanup.
func newProcessVerifier(t *testing.T) (*processVerifier, func()) {
	t.Helper()

	helperPath, err := getHelperPath()
	if err != nil {
		t.Fatalf("test helper not found: %v", err)
	}
	_ = helperPath

	manifest, err := os.CreateTemp("", "leamas-pid-manifest-*.jsonl")
	if err != nil {
		t.Fatalf("failed to create manifest file: %v", err)
	}
	if err := manifest.Close(); err != nil {
		t.Fatalf("failed to close manifest file: %v", err)
	}

	readyDir, err := os.MkdirTemp("", "leamas-pid-ready-")
	if err != nil {
		_ = os.Remove(manifest.Name())
		t.Fatalf("failed to create ready directory: %v", err)
	}

	v := &processVerifier{
		manifestFile: manifest.Name(),
		readyDir:     readyDir,
		t:            t,
	}
	cleanup := func() {
		// Best-effort removal. testdata/testhelper/main.go guarantees
		// deterministic readiness publication so leftover ready sentinel
		// files inside readyDir are diagnostic evidence of a leaked child.
		_ = os.Remove(v.manifestFile)
		_ = os.RemoveAll(v.readyDir)
	}
	// Use t.Cleanup so even a failing Fatal preserves cleanup ordering.
	t.Cleanup(cleanup)
	return v, cleanup
}

// ManifestFile returns the path to the PID manifest file.
func (v *processVerifier) ManifestFile() string {
	return v.manifestFile
}

// ReadyDir returns the path to the readiness directory. Tests must export
// this path via LEAMAS_EXEC_TEST_READY_DIR so the helper can publish
// per-process ready sentinel files.
func (v *processVerifier) ReadyDir() string {
	return v.readyDir
}

// parseManifest reads and parses the PID manifest.
func (v *processVerifier) parseManifest() ([]PIDRecord, error) {
	v.t.Helper()

	f, err := os.Open(v.manifestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer f.Close()

	records := make([]PIDRecord, 0)
	scanner := bufio.NewScanner(f)

	const maxLineLength = 1024
	buf := make([]byte, maxLineLength)
	scanner.Buffer(buf, maxLineLength)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec PIDRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("line %d: malformed JSON: %w", lineNum, err)
		}
		if rec.Role == "" || rec.PID == 0 {
			return nil, fmt.Errorf("line %d: missing required fields", lineNum)
		}
		records = append(records, rec)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	v.records = records
	return records, nil
}

// requireNonEmptyManifest asserts that the manifest contains at least one record.
func (v *processVerifier) requireNonEmptyManifest() {
	v.t.Helper()
	if len(v.records) == 0 {
		v.t.Fatal("manifest is empty - no PID records recorded")
	}
}

// requireExpectedRoles asserts that every role listed in
// expectedRolesForMode is present in the latest manifest. Roles missing
// from the manifest are recorded as test errors - a missing role is a
// proof failure, not a warning.
func (v *processVerifier) requireExpectedRoles(mode string) {
	v.t.Helper()

	expected, ok := expectedRolesForMode[mode]
	if !ok {
		return
	}

	recordedRoles := make(map[string]int)
	for _, rec := range v.records {
		recordedRoles[rec.Role]++
	}

	for _, role := range expected {
		if count := recordedRoles[role]; count == 0 {
			v.t.Errorf("expected role %q not found in manifest", role)
		}
	}
}

// requireSignalReadyForRoles asserts that every role listed in
// signalReadyForMode[mode] is recorded with SignalReady=true.
func (v *processVerifier) requireSignalReadyForRoles(mode string) {
	v.t.Helper()

	required, ok := signalReadyForMode[mode]
	if !ok {
		return
	}
	readyByRole := make(map[string]int)
	for _, rec := range v.records {
		if rec.SignalReady {
			readyByRole[rec.Role]++
		}
	}
	for _, role := range required {
		if readyByRole[role] == 0 {
			v.t.Errorf("expected role %q to carry signal_ready=true, got false", role)
		}
	}
}

// requireExactlyOnePGID asserts that all records share the same PGID.
func (v *processVerifier) requireExactlyOnePGID() int {
	v.t.Helper()
	if len(v.records) == 0 {
		v.t.Fatal("no records to check PGID")
	}

	pgid := v.records[0].PGID
	for _, rec := range v.records[1:] {
		if rec.PGID != pgid {
			v.t.Errorf("PGID mismatch: first=%d, got=%d", pgid, rec.PGID)
		}
	}
	return pgid
}

// getProcessGroups returns unique process group IDs from the manifest.
func (v *processVerifier) getProcessGroups() map[int]struct{} {
	groups := make(map[int]struct{})
	for _, rec := range v.records {
		groups[rec.PGID] = struct{}{}
	}
	return groups
}

// readinessObservation captures the per-attempt state observed by
// waitForReadiness. It is returned alongside an error so callers can produce
// actionable diagnostics when a readiness deadline expires.
type readinessObservation struct {
	ObservedRoles      []string
	ObservedReadyPIDs  []int
	MissingRoles       []string
	MissingReadyRoles  []string
	ParseErr           error
	RecordsCount       int
	ReadySentinelFiles []string
}

// waitForReadiness polls the manifest until every role listed in
// expectedRolesForMode is recorded AND every role listed in
// signalReadyForMode[mode] is recorded with SignalReady=true.
//
// The fsynced manifest record is the SOLE AUTHORITY for role and
// signal-readiness evidence. The per-PID `<pid>.ready` sentinel files
// the helper emits via publishReady are DIAGNOSTIC only and are NOT
// consulted by this function. Stage-specific hand-off sentinels
// (e.g. `descriptor-ready.wait`, `parent-exit-imminent.<pid>`,
// `<pid>.output-flood-ready`) are test-specific handoffs whose
// observability is the responsibility of the calling test; they
// are also NOT consulted by this function.
//
// Returns nil when all conditions hold, or an error describing the
// last observation if the deadline elapses first.
//
// The function is deterministic with respect to its inputs and never
// invokes time.Sleep outside its bounded deadline. The poll interval
// is readinessPollInterval.
func (v *processVerifier) waitForReadiness(mode string, deadline time.Time) error {
	v.t.Helper()

	expected, expectedOK := expectedRolesForMode[mode]
	signalReady, signalReadyOK := signalReadyForMode[mode]
	var lastErr error

	for {
		obs, parseErr := v.observeReadiness(mode, expected, expectedOK,
			signalReady, signalReadyOK)
		// Preserve the most recent parse error for diagnostics.
		if parseErr != nil {
			lastErr = parseErr
		}
		if len(obs.MissingRoles) == 0 &&
			len(obs.MissingReadyRoles) == 0 &&
			parseErr == nil {
			v.records = v.collectRecords()
			return nil
		}
		if time.Now().After(deadline) {
			// Build the most informative error possible.
			switch {
			case lastErr != nil:
				return fmt.Errorf(
					"readiness deadline exceeded (last parse error: %v)\n"+
						"  observed roles=%v\n"+
						"  ready pids=%v\n"+
						"  missing roles=%v\n"+
						"  missing ready roles=%v",
					lastErr,
					obs.ObservedRoles,
					obs.ObservedReadyPIDs,
					obs.MissingRoles,
					obs.MissingReadyRoles)
			case len(obs.MissingRoles) > 0 || len(obs.MissingReadyRoles) > 0:
				return fmt.Errorf(
					"readiness deadline exceeded\n"+
						"  observed roles=%v\n"+
						"  ready pids=%v\n"+
						"  missing roles=%v\n"+
						"  missing ready roles=%v\n"+
						"  ready sentinel files=%v",
					obs.ObservedRoles,
					obs.ObservedReadyPIDs,
					obs.MissingRoles,
					obs.MissingReadyRoles,
					obs.ReadySentinelFiles)
			default:
				return fmt.Errorf(
					"readiness deadline exceeded (no observation captured)")
			}
		}
		time.Sleep(readinessPollInterval)
	}
}

// observeReadiness takes a single snapshot of the manifest and readiness
// directory. It is separated from waitForReadiness so other call sites
// (final-state assertions) can reuse the same evaluation logic.
func (v *processVerifier) observeReadiness(mode string,
	expected []string, expectedOK bool,
	signalReady []string, signalReadyOK bool,
) (readinessObservation, error) {
	obs := readinessObservation{}

	records, parseErr := v.parseManifest()
	if parseErr != nil {
		obs.ParseErr = parseErr
		return obs, parseErr
	}
	obs.RecordsCount = len(records)

	// Track observed roles (preserving first observation order).
	seen := make(map[string]bool)
	for _, rec := range records {
		if !seen[rec.Role] {
			seen[rec.Role] = true
			obs.ObservedRoles = append(obs.ObservedRoles, rec.Role)
		}
		if rec.SignalReady {
			obs.ObservedReadyPIDs = append(obs.ObservedReadyPIDs, rec.PID)
		}
	}
	obs.ObservedReadyPIDs = sortAndDedup(obs.ObservedReadyPIDs)

	if expectedOK {
		expectedSet := make(map[string]bool, len(expected))
		for _, role := range expected {
			expectedSet[role] = true
		}
		observedSet := make(map[string]bool, len(seen))
		for role := range seen {
			observedSet[role] = true
		}
		for role := range expectedSet {
			if !observedSet[role] {
				obs.MissingRoles = append(obs.MissingRoles, role)
			}
		}
		sort.Strings(obs.MissingRoles)
	}

	if signalReadyOK {
		readyByRole := make(map[string]int)
		for _, rec := range records {
			if rec.SignalReady {
				readyByRole[rec.Role]++
			}
		}
		for _, role := range signalReady {
			if readyByRole[role] == 0 {
				obs.MissingReadyRoles = append(obs.MissingReadyRoles, role)
			}
		}
		sort.Strings(obs.MissingReadyRoles)
	}

	// Snapshot ready sentinel files for diagnostics. We do not require the
	// ready sentinel file presence to consider readiness satisfied because
	// SignalReady in the manifest is the explicit evidence the test relies
	// on. The directory contents still surface auxiliary diagnostics.
	entries, err := os.ReadDir(v.readyDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			obs.ReadySentinelFiles = append(obs.ReadySentinelFiles,
				filepath.Join(v.readyDir, e.Name()))
		}
		sort.Strings(obs.ReadySentinelFiles)
	}

	return obs, nil
}

// collectRecords returns the latest parsed records for caching on the
// verifier. waitForReadiness updates v.records on successful completion so
// callers can use requireNonEmptyManifest and requireExpectedRoles without
// re-parsing.
func (v *processVerifier) collectRecords() []PIDRecord {
	if v.records != nil {
		return v.records
	}
	records, _ := v.parseManifest()
	return records
}

func sortAndDedup(ints []int) []int {
	if len(ints) <= 1 {
		return ints
	}
	seen := make(map[int]bool)
	out := ints[:0]
	for _, n := range ints {
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}
