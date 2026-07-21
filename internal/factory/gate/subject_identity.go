// Package gate provides subject identity collection for metrics.
package gate

import (
	"context"
	"strings"

	"github.com/s1onique/leamas/internal/execution"
)

// runGit runs a git command using the bounded execution gateway.
// It propagates the caller context for cancellation support.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	result, err := execution.RunGit(ctx, dir, args...)
	if err != nil {
		return "", err
	}
	return string(result.Stdout), nil
}

// runGitWithOutput runs a git command and returns raw output bytes.
func runGitWithOutput(ctx context.Context, dir string, args ...string) ([]byte, error) {
	result, err := execution.RunGit(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	return result.Stdout, nil
}

// getHEADPaths returns tracked file paths from HEAD.
func getHEADPaths(ctx context.Context, root string) ([]string, error) {
	out, err := runGitWithOutput(ctx, root, "ls-tree", "-z", "--name-only", "HEAD")
	if err != nil {
		return nil, err
	}
	// Preserve NUL-delimited output exactly
	paths := strings.Split(string(out), "\x00")
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != "" {
			result = append(result, p)
		}
	}
	return result, nil
}

// getIndexPaths returns file paths from the index.
func getIndexPaths(ctx context.Context, root string) ([]string, error) {
	out, err := runGitWithOutput(ctx, root, "ls-files", "--stage", "-z")
	if err != nil {
		return nil, err
	}
	// Preserve NUL-delimited output exactly
	entries := strings.Split(string(out), "\x00")
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		result = append(result, parts[1])
	}
	return result, nil
}

// getNonignoredUntrackedPaths returns non-ignored untracked file paths.
func getNonignoredUntrackedPaths(ctx context.Context, root string) ([]string, error) {
	out, err := runGitWithOutput(ctx, root, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	// Preserve NUL-delimited output exactly
	paths := strings.Split(string(out), "\x00")
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != "" {
			result = append(result, p)
		}
	}
	return result, nil
}
