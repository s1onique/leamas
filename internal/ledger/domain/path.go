package domain

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
)

// ErrPathTraversal is returned when a repository-relative path escapes the
// repository root.
var ErrPathTraversal = errors.New("path escapes repository root")

// ErrInvalidDigest is returned when a SHA-256 digest is malformed.
var ErrInvalidDigest = errors.New("invalid SHA-256 digest")

// sha256Pattern is the canonical lowercase hexadecimal SHA-256 pattern.
var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ValidateSHA256 validates a lowercase hexadecimal SHA-256 digest.
func ValidateSHA256(digest string) error {
	if digest == "" {
		return fmt.Errorf("%w: digest must not be empty", ErrInvalidDigest)
	}
	if !sha256Pattern.MatchString(digest) {
		return fmt.Errorf("%w: %q", ErrInvalidDigest, digest)
	}
	return nil
}

// ValidateRelativePath validates a repository-relative artifact path.
//
// The path must be relative, must not escape the repository root and must
// not contain a Windows drive prefix or a backslash separator.
func ValidateRelativePath(rel string) error {
	if rel == "" {
		return fmt.Errorf("%w: path must not be empty", ErrPathTraversal)
	}
	if strings.ContainsRune(rel, '\\') {
		return fmt.Errorf("%w: %q must not contain a backslash separator", ErrPathTraversal, rel)
	}
	if strings.Contains(rel, "\x00") {
		return fmt.Errorf("%w: %q must not contain a NUL byte", ErrPathTraversal, rel)
	}
	if path.IsAbs(rel) {
		return fmt.Errorf("%w: %q must be relative", ErrPathTraversal, rel)
	}
	if len(rel) >= 2 && rel[1] == ':' {
		return fmt.Errorf("%w: %q must not carry a drive prefix", ErrPathTraversal, rel)
	}
	cleaned := path.Clean(rel)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("%w: %q", ErrPathTraversal, rel)
	}
	if cleaned == "." {
		return fmt.Errorf("%w: %q does not name a file", ErrPathTraversal, rel)
	}
	return nil
}
