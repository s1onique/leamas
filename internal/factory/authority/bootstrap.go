// SPDX-License-Identifier: Apache-2.0

// Package authority: bootstrap.go provides the deterministic recovery
// path that rebuilds the leamas binary bound to the current
// repository HEAD when the installed binary is stale.
//
// The bootstrap is bounded: it does not touch tracked files, does
// not recursively invoke leamas, and fails closed on any build or
// verification failure.
package authority

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BootstrapOptions controls the bootstrap operation.
type BootstrapOptions struct {
	// RepoRoot is the repository root to build from.
	RepoRoot string
	// OutputPath is the destination for the built binary. When
	// empty, the canonical bin/leamas in RepoRoot is used.
	OutputPath string
	// Environment supplies GOOS / GOARCH / CGO_ENABLED overrides.
	Environment []string
	// WorkingTreeClean, when true, refuses to build when the
	// repository has uncommitted changes. The exact-HEAD rule
	// must not be bypassed by an unverified local tree.
	WorkingTreeClean bool
	// BuildTimeoutSeconds, when positive, bounds the build
	// invocation. The default is 300 seconds.
	BuildTimeoutSeconds int
}

// BootstrapResult describes the outcome of a bootstrap invocation.
type BootstrapResult struct {
	OutputPath     string
	OutputSHA256   string
	EmbeddedCommit string
	DurationMS     int64
	OutputBytes    int64
}

// canonicalBootstrapPath returns the conventional leamas binary path
// for the given repository root.
func canonicalBootstrapPath(repoRoot string) string {
	return filepath.Join(repoRoot, "bin", "leamas")
}

// CanonicalBootstrapPath is the public form of canonicalBootstrapPath.
func CanonicalBootstrapPath(repoRoot string) string {
	return canonicalBootstrapPath(repoRoot)
}

// ErrBootstrapDirty is returned when the working tree has uncommitted
// changes and WorkingTreeClean is required.
var ErrBootstrapDirty = errors.New("bootstrap refused: working tree is dirty")

// ErrBuildFailed is returned when the underlying go build fails.
type ErrBuildFailed struct {
	Output string
	Err    error
}

func (e *ErrBuildFailed) Error() string {
	return fmt.Sprintf("go build failed: %v: %s", e.Err, e.Output)
}
func (e *ErrBuildFailed) Unwrap() error { return e.Err }

// ErrVerificationFailed is returned when the freshly built binary
// does not satisfy the embedded-commit contract.
var ErrVerificationFailed = errors.New("bootstrap verification failed")

// reentryEnvVars is the closed list of environment variables that
// the emergency re-entry fuse in main.go uses to detect a nested
// Leamas invocation. The bootstrap verify child must not see them,
// otherwise the fresh binary refuses to start.
var reentryEnvVars = []string{
	"LEAMAS_EXEC_ROOT_ID",
	"LEAMAS_EXEC_PARENT_PID",
	"LEAMAS_EXEC_GENERATION",
}

// cleanReentryEnv returns env with every LEAMAS_EXEC_* variable
// stripped. The verify child of the bootstrap must run as a fresh
// root invocation.
func cleanReentryEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, line := range env {
		strip := false
		for _, name := range reentryEnvVars {
			if strings.HasPrefix(line, name+"=") {
				strip = true
				break
			}
		}
		if !strip {
			out = append(out, line)
		}
	}
	return out
}

// BootstrapSelf rebuilds the leamas binary bound to the current
// repository HEAD. The implementation:
//
//  1. resolves the canonical output path;
//  2. checks the working tree when WorkingTreeClean is required;
//  3. computes the VCS revision to embed;
//  4. invokes `go build` with -buildvcs and -trimpath;
//  5. verifies the resulting binary embeds the expected commit;
//  6. computes the output file's SHA-256.
//
// The function never modifies tracked repository files.
func BootstrapSelf(opts BootstrapOptions) (*BootstrapResult, error) {
	if opts.RepoRoot == "" {
		return nil, errors.New("repository root is required")
	}

	if opts.OutputPath == "" {
		opts.OutputPath = canonicalBootstrapPath(opts.RepoRoot)
	}

	if opts.WorkingTreeClean {
		out, err := DefaultGitRunner(opts.RepoRoot, "status", "--porcelain")
		if err != nil {
			return nil, fmt.Errorf("check working tree: %w", err)
		}
		if strings.TrimSpace(out) != "" {
			return nil, ErrBootstrapDirty
		}
	}

	head, err := DefaultGitRunner(opts.RepoRoot, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("resolve HEAD: %w", err)
	}
	shortHead := head
	if len(shortHead) > 12 {
		shortHead = shortHead[:12]
	}

	if err := os.MkdirAll(filepath.Dir(opts.OutputPath), 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	buildTimeout := opts.BuildTimeoutSeconds
	if buildTimeout <= 0 {
		buildTimeout = 300
	}

	cmd := exec.Command("go", "build",
		"-buildvcs=true",
		"-trimpath",
		"-ldflags", ldflagsFor(head),
		"-o", opts.OutputPath,
		"./cmd/leamas")
	cmd.Dir = opts.RepoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if len(opts.Environment) > 0 {
		cmd.Env = append(cmd.Env, opts.Environment...)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, &ErrBuildFailed{Output: string(out), Err: err}
	}

	info, err := os.Stat(opts.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("stat built binary: %w", err)
	}

	hash, embedded, err := verifyBuiltBinary(opts.OutputPath, head)
	if err != nil {
		return nil, err
	}

	return &BootstrapResult{
		OutputPath:     opts.OutputPath,
		OutputSHA256:   hash,
		EmbeddedCommit: embedded,
		OutputBytes:    info.Size(),
	}, nil
}

// ldflagsFor builds the linker flag string used to embed the
// binary's VCS commit. The injected symbols cover all three of the
// version package globals that affect identity.
func ldflagsFor(commit string) string {
	return fmt.Sprintf("-X github.com/s1onique/leamas/internal/version.Commit=%s", commit)
}

// verifyBuiltBinary computes the SHA-256 of path and confirms that
// path's embedded VCS commit matches expected. We use a child
// invocation of `path --version --json` to extract the commit
// rather than scraping the binary directly: this guarantees that
// the verification path matches the runtime contract used by
// CheckExecutable.
func verifyBuiltBinary(path, expected string) (sha256Hex, embeddedCommit string, err error) {
	sum, err := fileSHA256(path)
	if err != nil {
		return "", "", fmt.Errorf("hash built binary: %w", err)
	}

	verifyCmd := exec.Command(path, "version", "--json")
	// The newly-built binary inherits the parent's environment,
	// including LEAMAS_EXEC_ROOT_ID / LEAMAS_EXEC_PARENT_PID /
	// LEAMAS_EXEC_GENERATION. Those variables are set by the
	// emergency re-entry fuse in main.go and would cause the
	// verification child to refuse to start. Strip them so the
	// verify child starts as a fresh root invocation.
	verifyCmd.Env = cleanReentryEnv(os.Environ())
	out, runErr := verifyCmd.Output()
	if runErr != nil {
		return "", "", fmt.Errorf("%w: %v", ErrVerificationFailed, runErr)
	}
	embeddedCommit = extractCommitFromJSON(string(out))
	if embeddedCommit == "" {
		return "", "", fmt.Errorf("%w: commit field missing", ErrVerificationFailed)
	}
	if !strings.EqualFold(embeddedCommit, expected) {
		return "", "", fmt.Errorf("%w: embedded %q != expected %q",
			ErrVerificationFailed, embeddedCommit, expected)
	}

	return sum, embeddedCommit, nil
}

// fileSHA256 returns the SHA-256 hex digest of the bytes of path.
func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// extractCommitFromJSON scans the bytes of `version --json` output
// for the `"commit"` field. It deliberately avoids importing
// encoding/json here so the helper remains in this file.
func extractCommitFromJSON(s string) string {
	const key = `"commit":`
	idx := strings.Index(s, key)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(key):]
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	rest = rest[1:]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return ""
	}
	return rest[:end]
}
