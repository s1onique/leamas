package releasedeb

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", ".."))
}

func readRepositoryFile(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(content)
}

func requireText(t *testing.T, content, want, source string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Errorf("%s does not contain %q", source, want)
	}
}

func rejectText(t *testing.T, content, unwanted, source string) {
	t.Helper()
	if strings.Contains(content, unwanted) {
		t.Errorf("%s unexpectedly contains %q", source, unwanted)
	}
}

func TestNFPMConfigDeclaresDebianContract(t *testing.T) {
	config := readRepositoryFile(t, "packaging/nfpm.yaml")
	for _, want := range []string{
		"name: leamas",
		"arch: amd64",
		"platform: linux",
		"version: ${LEAMAS_PACKAGE_VERSION}",
		"release: \"1\"",
		"section: devel",
		"priority: optional",
		"maintainer: \"Alex Chistyakov <s1onique@users.noreply.github.com>\"",
		"homepage: \"https://github.com/s1onique/leamas\"",
		"license: \"Apache-2.0\"",
		"src: \"${LEAMAS_RELEASE_BINARY}\"",
		"dst: /usr/bin/leamas",
		"expand: true",
		"mode: 0755",
	} {
		requireText(t, config, want, "packaging/nfpm.yaml")
	}
}

func TestMakefileDeclaresPinnedDebianTargets(t *testing.T) {
	makefile := readRepositoryFile(t, "Makefile") + "\n" + readRepositoryFile(t, "packaging/deb.mk")
	for _, want := range []string{
		"NFPM_VERSION ?= v2.47.0",
		"DEB_ARCH ?= amd64",
		"leamas_$(VERSION)_$(DEB_ARCH).deb",
		"package-deb:",
		"package-deb-inspect:",
		"package-deb-install-smoke:",
		"package-deb-verify:",
		"release-deb:",
		"github.com/goreleaser/nfpm/v2/cmd/nfpm@$(NFPM_VERSION)",
	} {
		requireText(t, makefile, want, "Makefile")
	}
	rejectText(t, makefile, "cmd/nfpm@latest", "Makefile")
}

func TestReleaseWorkflowDeclaresPublicationSafetyContract(t *testing.T) {
	workflow := readRepositoryFile(t, ".github/workflows/release-deb.yml")
	for _, want := range []string{
		"\"v[0-9]+.[0-9]+.[0-9]+\"",
		"workflow_dispatch:",
		"release_tag:",
		"contents: write",
		"runs-on: ubuntu-24.04",
		"actions/checkout@v7",
		"actions/setup-go@v7",
		"go-version-file: go.mod",
		"fetch-depth: 0",
		"persist-credentials: false",
		"test -z \"$(git status --porcelain)\"",
		"git tag --points-at HEAD",
		"git ls-remote",
		"refs/tags/$RELEASE_TAG",
		"ab041dc611b276c38bc27d8d38c8159f84729c50",
		"GOTOOLCHAIN=auto",
		"name: Release input preflight",
		"name: Build canonical release binary",
		"name: Verify release stamp",
		"name: Build Debian package",
		"name: Inspect Debian metadata",
		"name: Verify extracted binary",
		"name: Run Lintian",
		"name: Generate checksums",
		"name: Verify checksums",
		"name: Install package",
		"name: Execute installed package",
		"name: Remove package",
		"name: Publish GitHub Release",
		"go env GOOS GOARCH GOTOOLCHAIN GOPATH GOMODCACHE",
		"lintian --version",
		"git rev-parse \"${RELEASE_TAG}^{commit}\"",
		"gh release create \"$RELEASE_TAG\"",
		"--verify-tag",
		"--generate-notes",
		"GH_TOKEN: ${{ github.token }}",
		"cancel-in-progress: false",
	} {
		requireText(t, workflow, want, ".github/workflows/release-deb.yml")
	}
	rejectText(t, workflow, "--clobber", ".github/workflows/release-deb.yml")
	rejectText(t, workflow, "git tag -f", ".github/workflows/release-deb.yml")
	rejectText(t, workflow, "git push --force", ".github/workflows/release-deb.yml")
}

func TestFactoryWorkflowUsesRepositoryGoAuthority(t *testing.T) {
	workflow := readRepositoryFile(t, ".github/workflows/factory.yml")
	for _, want := range []string{
		"actions/setup-go@v7",
		"go-version-file: go.mod",
		"cache: true",
		"cache-dependency-path: go.sum",
	} {
		requireText(t, workflow, want, ".github/workflows/factory.yml")
	}
	rejectText(t, workflow, "go-version: '1.22'", ".github/workflows/factory.yml")
}

func TestApacheLicenseSelectionIsExplicit(t *testing.T) {
	license := readRepositoryFile(t, "LICENSE")
	for _, want := range []string{
		"Apache License",
		"Version 2.0, January 2004",
		"http://www.apache.org/licenses/",
		"END OF TERMS AND CONDITIONS",
	} {
		requireText(t, license, want, "LICENSE")
	}
	if len(license) != 11358 {
		t.Errorf("LICENSE length = %d, want the unmodified Apache-2.0 text length 11358", len(license))
	}
}
