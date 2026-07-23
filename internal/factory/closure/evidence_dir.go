package closure

import (
	"fmt"
	"os"
	"path/filepath"
)

func prepareEvidenceDirectory(repositoryRoot, evidenceDirectory string) (string, error) {
	if evidenceDirectory == "" || !filepath.IsAbs(evidenceDirectory) {
		return "", fmt.Errorf("evidence directory must be an absolute path")
	}
	resolvedRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	resolvedRoot, err = filepath.Abs(resolvedRoot)
	if err != nil {
		return "", fmt.Errorf("make repository root absolute: %w", err)
	}
	if err := os.MkdirAll(evidenceDirectory, 0o700); err != nil {
		return "", fmt.Errorf("create evidence directory: %w", err)
	}
	resolvedEvidence, err := filepath.EvalSymlinks(evidenceDirectory)
	if err != nil {
		return "", fmt.Errorf("resolve evidence directory: %w", err)
	}
	resolvedEvidence, err = filepath.Abs(resolvedEvidence)
	if err != nil {
		return "", fmt.Errorf("make evidence directory absolute: %w", err)
	}
	inside, err := pathInside(resolvedEvidence, resolvedRoot)
	if err != nil {
		return "", err
	}
	if inside {
		return "", fmt.Errorf("evidence directory must resolve outside the Git worktree")
	}
	info, err := os.Stat(resolvedEvidence)
	if err != nil {
		return "", fmt.Errorf("stat evidence directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("evidence path is not a directory")
	}
	return resolvedEvidence, nil
}

func pathInside(path, root string) (bool, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false, fmt.Errorf("compare paths: %w", err)
	}
	return relative == "." || relative != ".." && !startsWithParent(relative), nil
}

func startsWithParent(relative string) bool {
	return len(relative) > 3 && relative[:3] == ".."+string(filepath.Separator)
}
