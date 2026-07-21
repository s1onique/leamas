// Package gate provides subject identity collection for metrics.
package gate

import (
	"context"
	"fmt"
	"strings"

	"github.com/s1onique/leamas/internal/execution"
)

// runGitWithContext runs a git command using the bounded execution gateway.
func runGitWithContext(ctx context.Context, dir string, args ...string) (string, error) {
	result, err := execution.RunGit(ctx, dir, args...)
	if err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", result.Error
	}
	return string(result.Stdout), nil
}

// runGit runs a git command with default timeout.
func runGit(dir string, args ...string) (string, error) {
	result, err := execution.RunGit(context.Background(), dir, args...)
	if err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", result.Error
	}
	return string(result.Stdout), nil
}

// runGitWithOutput runs a git command and returns raw output bytes.
func runGitWithOutput(dir string, args ...string) ([]byte, error) {
	result, err := execution.RunGit(context.Background(), dir, args...)
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return result.Stdout, nil
}

// runGitOrBail runs git and wraps errors for the factorize context.
func runGitOrBail(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := runGitWithContext(ctx, dir, args...)
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return out, nil
}

// getHEADPaths returns tracked file paths from HEAD.
func getHEADPaths(root string) ([]string, error) {
	out, err := runGitWithOutput(root, "ls-tree", "-rz", "--name-only", "HEAD")
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
func getIndexPaths(root string) ([]string, error) {
	out, err := runGitWithOutput(root, "ls-files", "--stage", "-z")
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
func getNonignoredUntrackedPaths(root string) ([]string, error) {
	out, err := runGitWithOutput(root, "ls-files", "--others", "--exclude-standard", "-z")
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
