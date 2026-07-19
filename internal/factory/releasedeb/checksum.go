package releasedeb

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// WriteChecksum writes a basename-only SHA256SUMS entry for the Debian asset.
func WriteChecksum(packagePath, outputPath string) error {
	if packagePath == "" || outputPath == "" {
		return fmt.Errorf("checksum paths must not be empty")
	}
	info, err := os.Stat(packagePath)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("Debian package does not exist as a regular file: %s", packagePath)
	}
	sum, err := checksum(packagePath)
	if err != nil {
		return fmt.Errorf("hash Debian package: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(fmt.Sprintf("%s  %s\n", sum, filepath.Base(packagePath))), 0644); err != nil {
		return fmt.Errorf("write checksum file: %w", err)
	}
	return nil
}

// VerifyChecksum delegates final checksum-file parsing to sha256sum while
// retaining the release workflow's exact asset path contract.
func VerifyChecksum(ctx context.Context, packagePath, checksumPath string, out io.Writer) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("read checksum file: %w", err)
	}
	fields := strings.Fields(string(data))
	if len(fields) != 2 || fields[1] != filepath.Base(packagePath) {
		return fmt.Errorf("checksum file must name the package basename %q", filepath.Base(packagePath))
	}
	return printCommand(ctx, out, filepath.Dir(checksumPath), "sha256sum", "--check", filepath.Base(checksumPath))
}
