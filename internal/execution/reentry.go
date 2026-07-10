// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// Re-entry protection environment variable names.
const (
	EnvRootID     = "LEAMAS_EXEC_ROOT_ID"
	EnvParentPID  = "LEAMAS_EXEC_PARENT_PID"
	EnvGeneration = "LEAMAS_EXEC_GENERATION"
)

// ReentryPolicy defines the policy for handling nested execution.
type ReentryPolicy int

const (
	// ReentryPolicyReject rejects all nested execution attempts.
	ReentryPolicyReject ReentryPolicy = iota
	// ReentryPolicyAllow allows nested execution (for testing only).
	ReentryPolicyAllow
)

// checkReentry checks if Leamas is already running and returns an error if so.
// This is the emergency fuse that prevents process explosions.
func checkReentry(policy ReentryPolicy) error {
	if policy == ReentryPolicyAllow {
		return nil
	}

	rootID := os.Getenv(EnvRootID)
	if rootID != "" {
		// We're inside a Leamas execution
		parentPID := os.Getenv(EnvParentPID)
		generation := os.Getenv(EnvGeneration)
		return fmt.Errorf("%w: root_id=%s, parent_pid=%s, generation=%s",
			ErrNestedLeamasExecution, rootID, parentPID, generation)
	}

	return nil
}

// ExecutionRoot holds state for the root execution.
type ExecutionRoot struct {
	ID         string // Unique root execution ID
	ParentPID  int    // Parent process PID of the root Leamas
	Generation uint32 // Execution generation (0 = root)
	SelfPath   string // Resolved path to current executable
}

// NewExecutionRoot creates a new root execution context.
// Must be called at startup before any command execution.
func NewExecutionRoot() (*ExecutionRoot, error) {
	// Check for nested execution
	if err := checkReentry(ReentryPolicyReject); err != nil {
		return nil, err
	}

	// Resolve our own executable path to detect direct self-execution
	selfPath, err := resolveSelfExecutable()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve self executable: %w", err)
	}

	root := &ExecutionRoot{
		ID:         generateRootID(),
		ParentPID:  os.Getpid(), // Root's parent is the original parent
		Generation: 0,
		SelfPath:   selfPath,
	}

	// Set environment variables for child processes
	// These will be inherited by all child processes
	os.Setenv(EnvRootID, root.ID)
	os.Setenv(EnvParentPID, fmt.Sprintf("%d", os.Getpid())) // Leamas PID, not its parent
	os.Setenv(EnvGeneration, fmt.Sprintf("%d", root.Generation))

	return root, nil
}

// NewTestExecutionRoot creates a root execution for testing.
// This bypasses the re-entry check for unit tests.
func NewTestExecutionRoot() *ExecutionRoot {
	return &ExecutionRoot{
		ID:         generateRootID(),
		ParentPID:  os.Getpid(),
		Generation: 0,
		SelfPath:   os.Args[0],
	}
}

// ForChild returns a copy configured for a child invocation.
// The generation is incremented for the child.
func (r *ExecutionRoot) ForChild() *ExecutionRoot {
	// Check for overflow
	gen := r.Generation + 1
	if gen < r.Generation {
		gen = ^uint32(0) // Max uint32 on overflow
	}
	return &ExecutionRoot{
		ID:         r.ID,
		ParentPID:  r.ParentPID,
		Generation: gen,
		SelfPath:   r.SelfPath,
	}
}

// IsSelfExecutable checks if the given path resolves to the current Leamas binary.
// This prevents executing ourselves directly.
func (r *ExecutionRoot) IsSelfExecutable(path string) bool {
	resolved, err := resolveExecutable(path)
	if err != nil {
		return false
	}
	return resolved == r.SelfPath
}

// resolveSelfExecutable resolves the current process's executable path.
func resolveSelfExecutable() (string, error) {
	// Get the executable path
	exePath, err := os.Executable()
	if err != nil {
		// Fallback: try reading /proc/self/exe on Linux
		exePath = "/proc/self/exe"
	}

	return resolveExecutable(exePath)
}

// resolveExecutable resolves a path to its canonical form.
// It resolves symlinks and returns the absolute path.
func resolveExecutable(path string) (string, error) {
	// Make it absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Clean the path
	cleanPath := filepath.Clean(absPath)

	// Try to resolve symlinks
	realPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		// If symlink resolution fails, use the clean path
		return cleanPath, nil
	}

	return realPath, nil
}

// generateRootID generates a unique root execution ID.
// Uses PID + timestamp + cryptographic randomness for uniqueness.
func generateRootID() string {
	// Get PID and timestamp
	pid := int64(os.Getpid())

	// Generate random bytes for uniqueness
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to less unique ID
		return fmt.Sprintf("leamas-%d-%d", pid, os.Getpid())
	}
	randomHex := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("leamas-%d-%s", pid, randomHex)
}
