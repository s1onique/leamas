// Package gate provides subject identity collection for metrics.
package gate

import (
	"context"
	"strings"

	"github.com/s1onique/leamas/internal/execution"
)

// runGitWithContext runs a git command using the bounded execution gateway.
func runGitWithContext(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := execution.RunGit(ctx, dir, args...)
	if err != nil {
		return "", err
	}
	return out, nil
}

// runGit runs a git command with no context cancellation.
func runGit(dir string, args ...string) (string, error) {
	out, err := execution.RunGitSimple(dir, args...)
	if err != nil {
		return "", err
	}
	return out, nil
}

// getHEADPaths returns tracked file paths from HEAD.
func getHEADPaths(root string) ([]string, error) {
	out, err := runGit(root, "ls-tree", "-rz", "--name-only", "HEAD")
	if err != nil {
		return nil, err
	}
	paths := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
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
	out, err := runGit(root, "ls-files", "--stage", "-z")
	if err != nil {
		return nil, err
	}
	entries := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
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
	out, err := runGit(root, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	paths := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != "" {
			result = append(result, p)
		}
	}
	return result, nil
}
