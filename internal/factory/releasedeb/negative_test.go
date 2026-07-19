package releasedeb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runMake(t *testing.T, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	commandArgs := append([]string{"-C", repositoryRoot(t), "--no-print-directory"}, args...)
	output, err := commandOutput(ctx, repositoryRoot(t), "make", commandArgs...)
	if ctx.Err() != nil {
		t.Fatalf("make timed out: %v\n%s", ctx.Err(), output)
	}
	return string(output), err
}

func TestPackageDebRejectsInvalidInputs(t *testing.T) {
	cases := []struct {
		name     string
		override []string
		want     string
	}{
		{name: "empty version", override: []string{"VERSION="}, want: "VERSION must be a strict stable SemVer"},
		{name: "development version", override: []string{"VERSION=dev"}, want: "VERSION must be a strict stable SemVer"},
		{name: "unknown version", override: []string{"VERSION=unknown"}, want: "VERSION must be a strict stable SemVer"},
		{name: "malformed version", override: []string{"VERSION=1.2"}, want: "VERSION must be a strict stable SemVer"},
		{name: "darwin target", override: []string{"GOOS=darwin"}, want: "GOOS must be linux"},
		{name: "arm target", override: []string{"GOARCH=arm64"}, want: "GOARCH must be amd64"},
		{name: "missing license", override: []string{"LICENSE_FILE=/does/not/exist"}, want: "license file does not exist"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dist := t.TempDir()
			args := []string{
				"package-deb",
				"VERSION=0.1.0",
				"GOOS=linux",
				"GOARCH=amd64",
				"LICENSE_FILE=" + filepath.Join(repositoryRoot(t), "LICENSE"),
				"DIST_DIR=" + dist,
			}
			args = append(args, tc.override...)
			output, err := runMake(t, args...)
			if err == nil {
				t.Fatalf("package-deb unexpectedly succeeded:\n%s", output)
			}
			if !strings.Contains(output, tc.want) {
				t.Fatalf("package-deb error did not contain %q:\n%s", tc.want, output)
			}
			if entries, readErr := os.ReadDir(dist); readErr == nil && len(entries) != 0 {
				t.Fatalf("invalid package request left artifacts in %s: %v", dist, entries)
			}
		})
	}
}

func TestReleaseStampVerifyRejectsWrongEmbeddedVersion(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "leamas")
	content := "#!/bin/sh\nprintf '%s\\n' 'version: 9.9.9' 'commit: testcommit' 'build_time: 2026-07-19T00:00:00Z'\n"
	if err := os.WriteFile(binary, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	output, err := runMake(t,
		"release-stamp-verify",
		"VERSION=0.1.0",
		"ARTIFACT_DIR="+dir,
		"COMMIT=testcommit",
	)
	if err == nil {
		t.Fatalf("release-stamp-verify unexpectedly accepted wrong version:\n%s", output)
	}
	if !strings.Contains(output, "reports version 9.9.9, expected 0.1.0") {
		t.Fatalf("wrong-version diagnostic missing:\n%s", output)
	}
}

func TestPackageDebVerifyRejectsAbsentCanonicalBinary(t *testing.T) {
	if !commandAvailable("dpkg-deb") {
		t.Skip("dpkg-deb is required for Debian package contract tests")
	}
	packagePath := buildDebFixture(t, "amd64", []byte("#!/bin/sh\nexit 0\n"), true)
	missingBinary := filepath.Join(t.TempDir(), "missing-leamas")
	output, err := runMake(t,
		"package-deb-verify",
		"VERSION=0.1.0",
		"DEB_PACKAGE="+packagePath,
		"RELEASE_BINARY="+missingBinary,
	)
	if err == nil {
		t.Fatalf("package-deb-verify accepted an absent canonical binary:\n%s", output)
	}
	if !strings.Contains(output, "canonical release binary does not exist") {
		t.Fatalf("missing-binary diagnostic missing:\n%s", output)
	}
}

func TestPackageDebInspectRejectsWrongArchitecture(t *testing.T) {
	if !commandAvailable("dpkg-deb") {
		t.Skip("dpkg-deb is required for Debian package contract tests")
	}
	packagePath := buildDebFixture(t, "arm64", []byte("#!/bin/sh\nexit 0\n"), true)
	output, err := runMake(t,
		"package-deb-inspect",
		"VERSION=0.1.0",
		"DEB_PACKAGE="+packagePath,
	)
	if err == nil {
		t.Fatalf("package-deb-inspect accepted an arm64 package:\n%s", output)
	}
	if !strings.Contains(output, "Architecture mismatch") {
		t.Fatalf("wrong-architecture diagnostic missing:\n%s", output)
	}
}

func TestPackageDebInspectRejectsMissingApplicationPayload(t *testing.T) {
	if !commandAvailable("dpkg-deb") {
		t.Skip("dpkg-deb is required for Debian package contract tests")
	}
	packagePath := buildDebFixture(t, "amd64", nil, false)
	output, err := runMake(t,
		"package-deb-inspect",
		"VERSION=0.1.0",
		"DEB_PACKAGE="+packagePath,
	)
	if err == nil {
		t.Fatalf("package-deb-inspect accepted a package without /usr/bin/leamas:\n%s", output)
	}
	if !strings.Contains(output, "exactly one executable /usr/bin/leamas") {
		t.Fatalf("missing-payload diagnostic missing:\n%s", output)
	}
}

func TestPackageDebVerifyRejectsDifferentExtractedBinary(t *testing.T) {
	if !commandAvailable("dpkg-deb") {
		t.Skip("dpkg-deb is required for Debian package contract tests")
	}
	canonical := filepath.Join(t.TempDir(), "canonical-leamas")
	if err := os.WriteFile(canonical, []byte("canonical bytes\n"), 0755); err != nil {
		t.Fatal(err)
	}
	packagePath := buildDebFixture(t, "amd64", []byte("different bytes\n"), true)
	output, err := runMake(t,
		"package-deb-verify",
		"VERSION=0.1.0",
		"DEB_PACKAGE="+packagePath,
		"RELEASE_BINARY="+canonical,
	)
	if err == nil {
		t.Fatalf("package-deb-verify accepted different extracted bytes:\n%s", output)
	}
	if !strings.Contains(output, "SHA-256 differs") {
		t.Fatalf("binary-mismatch diagnostic missing:\n%s", output)
	}
}

func commandAvailable(name string) bool {
	_, err := commandOutput(context.Background(), "", "sh", "-c", "command -v "+name)
	return err == nil
}

func buildDebFixture(t *testing.T, architecture string, payload []byte, includePayload bool) string {
	t.Helper()
	if !commandAvailable("dpkg-deb") {
		t.Fatal("dpkg-deb unavailable")
	}
	root := t.TempDir()
	controlDir := filepath.Join(root, "DEBIAN")
	if err := os.Mkdir(controlDir, 0755); err != nil {
		t.Fatal(err)
	}
	control := fmt.Sprintf("Package: leamas\nVersion: 0.1.0-1\nArchitecture: %s\nSection: devel\nPriority: optional\nMaintainer: Test <test@example.invalid>\nDescription: fixture\n", architecture)
	if err := os.WriteFile(filepath.Join(controlDir, "control"), []byte(control), 0644); err != nil {
		t.Fatal(err)
	}
	if includePayload {
		binDir := filepath.Join(root, "usr", "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(binDir, "leamas"), payload, 0755); err != nil {
			t.Fatal(err)
		}
	}
	packagePath := filepath.Join(t.TempDir(), "fixture.deb")
	if output, err := commandOutput(context.Background(), "", "dpkg-deb", "--build", "--root-owner-group", root, packagePath); err != nil {
		t.Fatalf("build Debian fixture: %v\n%s", err, output)
	}
	return packagePath
}
