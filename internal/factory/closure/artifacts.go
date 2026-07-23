package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type artifactHashHook func(stage string)

func collectRepositoryArtifacts(repositoryRoot string, planned []PlanArtifact) []ArtifactResult {
	results := make([]ArtifactResult, 0, len(planned))
	for _, artifact := range planned {
		result, err := hashRepositoryArtifact(repositoryRoot, artifact, nil)
		if err == nil {
			results = append(results, result)
			continue
		}
		status := ArtifactStatusFail
		if errors.Is(err, os.ErrNotExist) {
			status = ArtifactStatusMissing
		}
		diagnostic := strings.ReplaceAll(err.Error(), repositoryRoot, ".")
		results = append(results, ArtifactResult{
			ArtifactID: artifact.ID,
			Path:       artifact.Path,
			Required:   *artifact.Required,
			MediaType:  artifact.MediaType,
			Status:     status,
			Diagnostic: sanitizeDiagnostic(diagnostic),
		})
	}
	return results
}

func hashRepositoryArtifact(repositoryRoot string, planned PlanArtifact, hook artifactHashHook) (ArtifactResult, error) {
	if err := validateRepositoryRelativePath(planned.Path, false); err != nil {
		return ArtifactResult{}, fmt.Errorf("artifact path must remain inside repository: %w", err)
	}
	root, err := os.OpenRoot(repositoryRoot)
	if err != nil {
		return ArtifactResult{}, fmt.Errorf("open repository root: %w", err)
	}
	defer root.Close()
	relative := filepath.FromSlash(planned.Path)
	if err := rejectSymlinkComponents(root, relative); err != nil {
		return ArtifactResult{}, err
	}
	file, err := root.Open(relative)
	if err != nil {
		return ArtifactResult{}, fmt.Errorf("open artifact: %w", err)
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil {
		return ArtifactResult{}, fmt.Errorf("stat opened artifact: %w", err)
	}
	if !before.Mode().IsRegular() {
		return ArtifactResult{}, fmt.Errorf("artifact is not a regular file")
	}
	if before.Size() > planned.MaxBytes {
		return ArtifactResult{}, fmt.Errorf("artifact exceeds configured maximum of %d bytes", planned.MaxBytes)
	}
	firstHash, firstCount, err := hashOpenArtifact(file, planned.MaxBytes)
	if err != nil {
		return ArtifactResult{}, err
	}
	if hook != nil {
		hook("between_hashes")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ArtifactResult{}, fmt.Errorf("rewind artifact: %w", err)
	}
	secondHash, secondCount, err := hashOpenArtifact(file, planned.MaxBytes)
	if err != nil {
		return ArtifactResult{}, err
	}
	after, err := file.Stat()
	if err != nil {
		return ArtifactResult{}, fmt.Errorf("restat opened artifact: %w", err)
	}
	if err := rejectSymlinkComponents(root, relative); err != nil {
		return ArtifactResult{}, fmt.Errorf("artifact path changed during hashing: %w", err)
	}
	pathInfo, err := root.Stat(relative)
	if err != nil {
		return ArtifactResult{}, fmt.Errorf("restat artifact path: %w", err)
	}
	if firstHash != secondHash || firstCount != secondCount || before.Size() != after.Size() ||
		!before.ModTime().Equal(after.ModTime()) || !os.SameFile(before, after) || !os.SameFile(after, pathInfo) {
		return ArtifactResult{}, fmt.Errorf("artifact changed during hashing")
	}
	return ArtifactResult{
		ArtifactID: planned.ID,
		Path:       planned.Path,
		Required:   *planned.Required,
		MediaType:  planned.MediaType,
		Status:     ArtifactStatusPass,
		SHA256:     firstHash,
		ByteCount:  firstCount,
	}, nil
}

func hashOpenArtifact(file *os.File, maximum int64) (string, int64, error) {
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(file, maximum+1))
	if err != nil {
		return "", 0, fmt.Errorf("hash artifact: %w", err)
	}
	if count > maximum {
		return "", 0, fmt.Errorf("artifact exceeds configured maximum of %d bytes", maximum)
	}
	return hex.EncodeToString(hash.Sum(nil)), count, nil
}

func rejectSymlinkComponents(root *os.Root, relative string) error {
	components := strings.FieldsFunc(filepath.Clean(relative), func(r rune) bool {
		return r == '/' || r == '\\'
	})
	current := ""
	for _, component := range components {
		current = filepath.Join(current, component)
		info, err := root.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect artifact path component %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("artifact path component %q is a symlink", current)
		}
	}
	return nil
}

func sanitizeDiagnostic(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	if len(value) > 240 {
		value = value[:240]
	}
	return value
}
