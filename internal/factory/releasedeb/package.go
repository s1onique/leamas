package releasedeb

import (
	"context"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

func commandOutput(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	return commandOutputWithPath(ctx, dir, "", name, args...)
}

func commandOutputWithPath(ctx context.Context, dir, path, name string, args ...string) ([]byte, error) {
	budget := execution.DefaultBudget().WithTimeout(10 * time.Minute).WithMaxOutputBytes(16 << 20)
	executor, err := execution.NewExecutor(budget, nil)
	if err != nil {
		return nil, fmt.Errorf("create execution gateway: %w", err)
	}
	defer executor.Close()
	env := []string{
		execution.EnvRootID + "=",
		execution.EnvParentPID + "=",
		execution.EnvGeneration + "=",
	}
	if path != "" {
		env = append(env, "PATH="+path)
	}
	result := executor.Execute(ctx, &execution.Request{
		Name:      name,
		Args:      append([]string{name}, args...),
		Dir:       dir,
		Env:       env,
		Timeout:   10 * time.Minute,
		OutputCap: 16 << 20,
	})
	output := append(append([]byte(nil), result.Stdout...), result.Stderr...)
	if result.Failed() {
		return output, fmt.Errorf("%s %s: command failed\n%s", name, strings.Join(args, " "), output)
	}
	return output, nil
}

func printCommand(ctx context.Context, out io.Writer, dir, name string, args ...string) error {
	output, err := commandOutput(ctx, dir, name, args...)
	if len(output) != 0 {
		_, _ = out.Write(output)
	}
	return err
}

// Inspect verifies the Debian control fields and the data payload shape.
func (c Config) Inspect(ctx context.Context, out io.Writer) error {
	if c.PackagePath == "" {
		return fmt.Errorf("package path is empty")
	}
	if info, err := os.Stat(c.PackagePath); err != nil {
		return fmt.Errorf("package does not exist: %s: %w", c.PackagePath, err)
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("package is not a regular file: %s", c.PackagePath)
	}

	fields := []string{"Package", "Version", "Architecture", "Section", "Priority"}
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		output, err := commandOutput(ctx, "", "dpkg-deb", "--field", c.PackagePath, field)
		if err != nil {
			return err
		}
		value := strings.TrimSpace(string(output))
		values[field] = value
		fmt.Fprintf(out, "%s: %s\n", field, value)
	}
	if values["Package"] != "leamas" {
		return fmt.Errorf("Package mismatch: got %q, want %q", values["Package"], "leamas")
	}
	if values["Version"] != c.expectedPackageVersion() {
		return fmt.Errorf("Version mismatch: got %q, want %q", values["Version"], c.expectedPackageVersion())
	}
	if values["Architecture"] != ExpectedArchitecture {
		return fmt.Errorf("Architecture mismatch: got %q, want %q", values["Architecture"], ExpectedArchitecture)
	}
	if values["Section"] != "devel" {
		return fmt.Errorf("Section mismatch: got %q, want %q", values["Section"], "devel")
	}
	if values["Priority"] != "optional" {
		return fmt.Errorf("Priority mismatch: got %q, want %q", values["Priority"], "optional")
	}

	contents, err := commandOutput(ctx, "", "dpkg-deb", "--contents", c.PackagePath)
	if err != nil {
		return err
	}
	_, _ = out.Write(contents)
	return validatePayload(string(contents))
}

func validatePayload(contents string) error {
	executables := make([]string, 0, 1)
	for _, line := range strings.Split(strings.TrimSpace(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasPrefix(fields[0], "-") {
			continue
		}
		path := strings.TrimPrefix(fields[len(fields)-1], "./")
		if strings.Contains(fields[0], "x") {
			executables = append(executables, path)
		}
	}
	if len(executables) != 1 || executables[0] != "usr/bin/leamas" {
		return fmt.Errorf("payload must contain exactly one executable /usr/bin/leamas; executable files: %v", executables)
	}
	return nil
}

// Verify performs inspection, Lintian validation, extraction, execution,
// static-binary validation, and byte identity checking.
func (c Config) Verify(ctx context.Context, out io.Writer) error {
	if err := requireRegularExecutable(c.ReleaseBinary, "canonical release binary"); err != nil {
		return err
	}
	if err := c.Inspect(ctx, out); err != nil {
		return err
	}
	if err := printCommand(ctx, out, "", "dpkg-deb", "--info", c.PackagePath); err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "leamas-deb-")
	if err != nil {
		return fmt.Errorf("create extraction directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	if err := printCommand(ctx, out, "", "dpkg-deb", "--extract", c.PackagePath, tmpDir); err != nil {
		return err
	}
	extracted := filepath.Join(tmpDir, "usr", "bin", "leamas")
	if err := requireRegularExecutable(extracted, "extracted binary"); err != nil {
		return err
	}
	canonicalSum, err := checksum(c.ReleaseBinary)
	if err != nil {
		return fmt.Errorf("hash canonical release binary: %w", err)
	}
	extractedSum, err := checksum(extracted)
	if err != nil {
		return fmt.Errorf("hash extracted binary: %w", err)
	}
	if canonicalSum != extractedSum {
		return fmt.Errorf("extracted binary SHA-256 differs from canonical release binary: canonical %s, extracted %s", canonicalSum, extractedSum)
	}
	if err := printCommand(ctx, out, "", "lintian", "--fail-on", "error", c.PackagePath); err != nil {
		return err
	}
	if err := verifyStaticAMD64(extracted); err != nil {
		return err
	}
	if err := verifyBinaryVersion(ctx, extracted, c.Version, c.Commit); err != nil {
		return err
	}
	fmt.Fprintf(out, "extracted binary SHA-256: %s\n", extractedSum)
	return nil
}

func verifyBinaryVersion(ctx context.Context, binary, expectedVersion, expectedCommit string) error {
	output, err := commandOutput(ctx, "", binary, "version")
	if err != nil {
		return fmt.Errorf("run extracted binary version: %w", err)
	}
	values := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		key, value, ok := strings.Cut(line, ": ")
		if ok {
			values[key] = value
		}
	}
	if values["version"] != expectedVersion {
		return fmt.Errorf("extracted binary version mismatch: got %q, want %q", values["version"], expectedVersion)
	}
	if expectedCommit != "" && values["commit"] != expectedCommit {
		return fmt.Errorf("extracted binary commit mismatch: got %q, want %q", values["commit"], expectedCommit)
	}
	if values["build_time"] == "" || values["build_time"] == "unknown" {
		return fmt.Errorf("extracted binary has no build-time stamp")
	}
	return nil
}

func verifyStaticAMD64(path string) error {
	file, err := elf.Open(path)
	if err != nil {
		return fmt.Errorf("open extracted binary as ELF: %w", err)
	}
	defer file.Close()
	if file.Class != elf.ELFCLASS64 || file.Machine != elf.EM_X86_64 {
		return fmt.Errorf("extracted binary is not Linux amd64 ELF")
	}
	for _, program := range file.Progs {
		if program.Type == elf.PT_INTERP {
			return fmt.Errorf("extracted binary is dynamically linked")
		}
	}
	libraries, err := file.ImportedLibraries()
	if err != nil {
		return fmt.Errorf("inspect extracted binary dynamic libraries: %w", err)
	}
	if len(libraries) != 0 {
		return fmt.Errorf("extracted binary imports dynamic libraries: %v", libraries)
	}
	return nil
}
