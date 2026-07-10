// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// CycleDetector tracks active execution fingerprints to detect cycles.
type CycleDetector struct {
	mu           sync.Mutex
	active       map[string]struct{}
	fingerprints map[string]string // fingerprint -> request name for error messages
}

// NewCycleDetector creates a new cycle detector.
func NewCycleDetector() *CycleDetector {
	return &CycleDetector{
		active:       make(map[string]struct{}),
		fingerprints: make(map[string]string),
	}
}

// ComputeFingerprint computes a SHA256 fingerprint for cycle detection.
// The fingerprint includes executable, normalized args, working directory, and action name.
func ComputeFingerprint(exec string, args []string, dir, action string) string {
	h := sha256.New()

	// Include executable
	h.Write([]byte("exec:"))
	h.Write([]byte(exec))
	h.Write([]byte("\n"))

	// Include normalized arguments
	h.Write([]byte("args:"))
	for _, arg := range args {
		h.Write([]byte(arg))
		h.Write([]byte("\x00"))
	}
	h.Write([]byte("\n"))

	// Include working directory
	h.Write([]byte("dir:"))
	h.Write([]byte(dir))
	h.Write([]byte("\n"))

	// Include action name
	h.Write([]byte("action:"))
	h.Write([]byte(action))
	h.Write([]byte("\n"))

	return hex.EncodeToString(h.Sum(nil))[:16] // Use first 16 chars for readability
}

// CheckAndTrack checks if a fingerprint is already active and marks it as active if not.
// Returns an error if a cycle is detected.
func (d *CycleDetector) CheckAndTrack(fingerprint, name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.active[fingerprint]; exists {
		err := &ExecutionError{
			Code:    CodeExecutionCycleDetected,
			Message: "execution cycle detected",
		}
		err.Dimension = "fingerprint"
		err.Limit = fingerprint
		err.Observed = name
		return err
	}

	d.active[fingerprint] = struct{}{}
	d.fingerprints[fingerprint] = name
	return nil
}

// Untrack removes a fingerprint from the active set.
func (d *CycleDetector) Untrack(fingerprint string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.active, fingerprint)
	delete(d.fingerprints, fingerprint)
}

// ActiveCount returns the number of active fingerprints.
func (d *CycleDetector) ActiveCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.active)
}

// IsActive returns true if the fingerprint is currently active.
func (d *CycleDetector) IsActive(fingerprint string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, exists := d.active[fingerprint]
	return exists
}
