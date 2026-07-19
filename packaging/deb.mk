# Debian packaging targets. The release binary remains owned by release-build.

RELEASE_BINARY ?= $(ARTIFACT_DIR)/leamas
LICENSE_FILE ?= LICENSE
LEAMAS_LICENSE ?= Apache-2.0
HOST_GOOS := $(shell env -u GOOS -u GOARCH go env GOOS)
HOST_GOARCH := $(shell env -u GOOS -u GOARCH go env GOARCH)

package-deb:
	@$(MAKE) --no-print-directory release-deb-preflight
	@$(MAKE) --no-print-directory release-build release-stamp-verify
	@test -x "$(RELEASE_BINARY)" || { echo "ERROR: canonical release binary is missing: $(RELEASE_BINARY)" >&2; exit 1; }
	@mkdir -p "$(DIST_DIR)"
	@LEAMAS_PACKAGE_VERSION="$(VERSION)" \
	 LEAMAS_RELEASE_BINARY="$(RELEASE_BINARY)" \
	 LEAMAS_LICENSE="$(LEAMAS_LICENSE)" \
	 GOOS="$(HOST_GOOS)" GOARCH="$(HOST_GOARCH)" \
	 go run github.com/goreleaser/nfpm/v2/cmd/nfpm@$(NFPM_VERSION) \
	 package --config packaging/nfpm.yaml --packager deb --target "$(DEB_PACKAGE)"
	@test -f "$(DEB_PACKAGE)"
	@echo "Done. Debian package: $(DEB_PACKAGE)"

release-deb-preflight:
	@test "$(DEB_ARCH)" = "amd64" || { echo "ERROR: DEB_ARCH must be amd64 (got $(DEB_ARCH))" >&2; exit 2; }
	@GOOS="$(HOST_GOOS)" GOARCH="$(HOST_GOARCH)" go run ./cmd/leamas factory verify release-deb preflight \
	 --version "$(VERSION)" --goos "$(GOOS)" --goarch "$(GOARCH)" \
	 --license-file "$(LICENSE_FILE)" --license "$(LEAMAS_LICENSE)" \
	 --nfpm-version "$(NFPM_VERSION)"

package-deb-inspect:
	@GOOS="$(HOST_GOOS)" GOARCH="$(HOST_GOARCH)" go run ./cmd/leamas factory verify release-deb inspect \
	 --package "$(DEB_PACKAGE)" --version "$(VERSION)" --arch "$(DEB_ARCH)"

package-deb-verify:
	@GOOS="$(HOST_GOOS)" GOARCH="$(HOST_GOARCH)" go run ./cmd/leamas factory verify release-deb verify \
	 --package "$(DEB_PACKAGE)" --binary "$(RELEASE_BINARY)" \
	 --version "$(VERSION)" --arch "$(DEB_ARCH)" --commit "$(COMMIT)"

package-deb-install-smoke:
	@GOOS="$(HOST_GOOS)" GOARCH="$(HOST_GOARCH)" go run ./cmd/leamas factory verify release-deb install-smoke \
	 --package "$(DEB_PACKAGE)" --version "$(VERSION)" \
	 --commit "$(COMMIT)" --arch "$(DEB_ARCH)"

release-deb:
	@$(MAKE) --no-print-directory package-deb
	@$(MAKE) --no-print-directory package-deb-inspect
	@$(MAKE) --no-print-directory package-deb-verify
	@GOOS="$(HOST_GOOS)" GOARCH="$(HOST_GOARCH)" go run ./cmd/leamas factory verify release-deb checksum \
	 --package "$(DEB_PACKAGE)" --output "$(DIST_DIR)/SHA256SUMS"
	@echo "Done. Debian release checksum: $(DIST_DIR)/SHA256SUMS"
