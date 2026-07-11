package doctrinecompiler

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Newline is the single documented newline convention.
const Newline = "\n"

// NormalizeTargetPath canonicalizes a repo-relative POSIX path.
//
// It rejects absolute paths, traversal segments, NUL bytes, and empty
// segments. Backslashes are not accepted. The returned string uses
// forward slashes only.
//
// ".." is rejected as a literal segment even when the cleaned result
// would resolve inside the root; we treat any traversal intent in the
// input as a programming error.
func NormalizeTargetPath(p string) (TargetPath, error) {
	if p == "" {
		return "", newError("validate", "path", "empty path")
	}
	if strings.ContainsRune(p, 0) {
		return "", newError("validate", "path", "path contains NUL byte")
	}
	if strings.ContainsRune(p, '\\') {
		return "", newError("validate", "path", "path contains backslash")
	}
	if filepath.IsAbs(p) {
		return "", newError("validate", "path", "absolute paths are forbidden: "+p)
	}
	// Reject any traversal or empty segment in the raw input.
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", newError("validate", "path", "traversal segment forbidden: "+p)
		}
		if seg == "" && p != "" {
			return "", newError("validate", "path", "empty path segment: "+p)
		}
	}
	cleaned := filepath.Clean(p)
	if cleaned == "." {
		return "", newError("validate", "path", "target root is not a valid projection entry")
	}
	if strings.HasPrefix(cleaned, "..") {
		return "", newError("validate", "path", "traversal segment forbidden: "+p)
	}
	// Reject any segment that is empty, ".", or ".." after split.
	parts := strings.Split(cleaned, "/")
	for _, seg := range parts {
		if !isPathSegmentSafe(seg) {
			return "", newError("validate", "path", "unsafe path segment: "+seg)
		}
	}
	return TargetPath(cleaned), nil
}

// ValidatePathUniqueness asserts that no two normalized paths collide.
func ValidatePathUniqueness(paths []TargetPath) error {
	seen := make(map[TargetPath]struct{}, len(paths))
	for _, p := range paths {
		if _, dup := seen[p]; dup {
			return newError("validate", "path", "duplicate normalized path: "+string(p))
		}
		seen[p] = struct{}{}
	}
	return nil
}

// Resolver resolves a target path against an absolute target root.
// It guarantees the resolved absolute path remains inside the root.
type Resolver struct {
	Root string
}

// NewResolver constructs a Resolver after canonicalizing the root.
func NewResolver(root string) (*Resolver, error) {
	if root == "" {
		return nil, newError("validate", "target", "empty target root")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, newError("validate", "target", fmt.Sprintf("abs root: %v", err))
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	return &Resolver{Root: abs}, nil
}

// Resolve maps a TargetPath to an absolute filesystem path inside Root.
func (r *Resolver) Resolve(p TargetPath) string {
	return filepath.Join(r.Root, filepath.FromSlash(string(p)))
}

// Contains reports whether abs is inside the resolver root.
// Both paths must be cleaned absolute paths.
func (r *Resolver) Contains(abs string) bool {
	rel, err := filepath.Rel(r.Root, abs)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, "..") || rel == ".." {
		return false
	}
	return true
}

// PathKind classifies the existing filesystem object at a path.
type PathKind int

const (
	PathMissing PathKind = iota
	PathRegularFile
	PathDirectory
	PathSymlink
	PathOther
)

// InspectPath returns the kind of object at p and, when applicable, the
// absolute path of the first symlink encountered while descending.
func (r *Resolver) InspectPath(p TargetPath) (kind PathKind, symlink string, err error) {
	rel := filepath.FromSlash(string(p))
	current := r.Root
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		if seg == "" || seg == "." {
			continue
		}
		next := filepath.Join(current, seg)
		lst, lerr := os.Lstat(next)
		if lerr != nil {
			if errors.Is(lerr, fs.ErrNotExist) {
				return PathMissing, "", nil
			}
			return PathMissing, "", lerr
		}
		mode := lst.Mode()
		if mode&os.ModeSymlink != 0 {
			target, terr := os.Readlink(next)
			if terr != nil {
				return PathSymlink, next, terr
			}
			if filepath.IsAbs(target) {
				return PathSymlink, next, nil
			}
			followed := filepath.Join(filepath.Dir(next), target)
			if !r.Contains(followed) {
				return PathSymlink, next, nil
			}
			return PathSymlink, next, nil
		}
		current = next
	}
	lst, err := os.Lstat(current)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return PathMissing, "", nil
		}
		return PathMissing, "", err
	}
	mode := lst.Mode()
	switch {
	case mode.IsRegular():
		return PathRegularFile, "", nil
	case mode.IsDir():
		return PathDirectory, "", nil
	default:
		return PathOther, "", nil
	}
}

// HasSymlinkEscape reports whether descending to p from Root traverses
// any symlink, and returns the absolute path of the first symlink
// encountered. A path that already exists as a symlink at the final
// component is also reported.
func (r *Resolver) HasSymlinkEscape(p TargetPath) (string, bool) {
	rel := filepath.FromSlash(string(p))
	current := r.Root
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		if seg == "" || seg == "." {
			continue
		}
		next := filepath.Join(current, seg)
		lst, err := os.Lstat(next)
		if err != nil {
			return "", false
		}
		if lst.Mode()&os.ModeSymlink != 0 {
			return next, true
		}
		current = next
	}
	// Final component symlink check.
	lst, err := os.Lstat(current)
	if err != nil {
		return "", false
	}
	if lst.Mode()&os.ModeSymlink != 0 {
		return current, true
	}
	return "", false
}

// deviceID returns the device ID of the filesystem containing path.
func deviceID(path string) (uint64, error) {
	var st syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil {
		return 0, err
	}
	return uint64(st.Dev), nil
}

// SameFilesystem reports whether root and target live on the same
// filesystem, which is required for atomic rename-based replacement.
func SameFilesystem(root, target string) (bool, error) {
	rDev, err := deviceID(root)
	if err != nil {
		return false, err
	}
	tDev, err := deviceID(target)
	if err != nil {
		return false, err
	}
	return rDev == tDev, nil
}
