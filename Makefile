# Leamas Makefile

.PHONY: help gate test clean digest factorize verify-doctrine verify-factory
.PHONY: verify-forbidden verify-single-lang verify-static verify-agent-doctrine
.PHONY: verify-tooling-boundaries verify-llm-friendly verify-agent-context
.PHONY: verify-git-hooks verify-domain-boundaries bootstrap install-git-hooks build digest install
.PHONY: coverage dupcode-baseline release release-build release-checksum release-verify release-clean
.PHONY: test-helper

# Install variables (GNU conventions)
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
INSTALL ?= install

# Build variables
MODULE_PATH := github.com/s1onique/leamas

# Version injection via linker flags
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)

LDFLAGS := -X '$(MODULE_PATH)/internal/version.Version=$(VERSION)' \
           -X '$(MODULE_PATH)/internal/version.Commit=$(COMMIT)' \
           -X '$(MODULE_PATH)/internal/version.BuildTime=$(BUILD_TIME)'

# Release variables
DIST_DIR ?= dist
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
ARTIFACT_DIR = $(DIST_DIR)/leamas_$(VERSION)_$(GOOS)_$(GOARCH)

# Digest target: generate targeted digest for review
# Uses smart default: dirty digest when working tree has changes, previous commit digest when clean
digest:
	@mkdir -p build
	@go run ./cmd/leamas factory digest --output build/leamas-digest.txt
	@cat build/leamas-digest.txt

# Colors
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m

help:
	@echo "Leamas - Make targets"
	@echo ""
	@echo "  make gate           - Run quality gate (verifiers + Go toolchain)"
	@echo "  make factorize     - Run factory verifiers only (no toolchain)"
	@echo "  make coverage      - Generate coverage profile and check threshold"
	@echo "  make bootstrap     - Configure repo-local git hooks path"
	@echo "  make test          - Run Go tests (if module exists)"
	@echo "  make build         - Build static binary to bin/leamas"
	@echo "  make clean         - Clean build artifacts"
	@echo ""
	@echo "  make verify-*      - Run individual verifiers:"
	@echo "    make verify-agent-doctrine     Doctrine Agent Contract check"
	@echo "    make verify-agent-context     Agent context files check"
	@echo "    make verify-doctrine          Doctrine inventory check"
	@echo "    make verify-factory          Factory docs check"
	@echo "    make verify-forbidden        Forbidden patterns check"
	@echo "    make verify-single-lang      Single language check"
	@echo "    make verify-static          Static binary intent check"
	@echo "    make verify-tooling-boundaries Tooling boundaries check"
	@echo "    make verify-llm-friendly   LLM-friendliness check"
	@echo "    make verify-git-hooks      Git hooks check"
	@echo ""
	@echo "  make install-git-hooks - Install Git hooks"
	@echo "  make install        - Build and install leamas to $(PREFIX)/bin"

gate:
	@echo "Running quality gate..."
	@chmod +x scripts/quality_gate.sh
	@./scripts/quality_gate.sh

factorize:
	@echo "Running factory factorize..."
	@chmod +x scripts/verify_*.sh
	@go run ./cmd/leamas factory factorize

# Coverage: generate coverage profile and check threshold
# Conservative ratchet threshold: raised from 60 to 64 per ACT-LEAMAS-FACTORY-GO-COVERAGE-RATCHET02
COVERAGE_PROFILE ?= .factory/coverage.out
COVERAGE_MIN_TOTAL ?= 64
COVERAGE_MIN_CMD_LEAMAS ?= 50
COVERAGE_MIN_INTERNAL_FACTORY ?= 67
COVERAGE_MIN_INTERNAL_HULK ?= 90
COVERAGE_MIN_INTERNAL_WEB ?= 70
COVERAGE_MIN_INTERNAL_WITNESS ?= 80

coverage:
	@echo "Generating coverage profile..."
	@mkdir -p .factory
	@go test ./... -covermode=atomic -coverprofile $(COVERAGE_PROFILE)
	@echo ""
	@go run ./cmd/leamas factory coverage \
		--profile $(COVERAGE_PROFILE) \
		--min-total $(COVERAGE_MIN_TOTAL) \
		--min-module cmd/leamas=$(COVERAGE_MIN_CMD_LEAMAS) \
		--min-module internal/factory=$(COVERAGE_MIN_INTERNAL_FACTORY) \
		--min-module internal/hulk=$(COVERAGE_MIN_INTERNAL_HULK) \
		--min-module internal/web=$(COVERAGE_MIN_INTERNAL_WEB) \
		--min-module internal/witness=$(COVERAGE_MIN_INTERNAL_WITNESS) \
		--json-output .factory/coverage-summary.json

# Dupcode baseline: generate or update the duplicate code baseline
# Use this to create or refresh .factory/dupcode-baseline.json
dupcode-baseline:
	@echo "Updating duplicate code baseline..."
	@mkdir -p .factory
	@go run ./cmd/leamas factory verify dupcode --update-baseline

bootstrap:
	@echo "Configuring git hooks path..."
	@git config --local core.hooksPath githooks
	@test "$$(git config --local --get core.hooksPath)" = "githooks"
	@echo "Bootstrap complete: core.hooksPath=$$(git config --local --get core.hooksPath)"

test:
	@if [ -f go.mod ]; then \
		echo "Running go test..."; \
		go test ./...; \
	else \
		echo "Go module not initialized yet; skipping go test."; \
	fi

build:
	@echo "Building Leamas (static binary)..."
	@mkdir -p bin
	@CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/leamas ./cmd/leamas
	@echo "Done. Binary: bin/leamas"

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f leamas
	@find . -name '*.test' -delete 2>/dev/null || true
	@echo "Done."

verify-agent-doctrine:
	@chmod +x scripts/verify_doctrine_agent_contracts.sh
	@echo "Running doctrine agent contract verifier..."
	@./scripts/verify_doctrine_agent_contracts.sh

verify-doctrine:
	@chmod +x scripts/verify_doctrine_inventory.sh
	@echo "Running doctrine inventory verifier..."
	@./scripts/verify_doctrine_inventory.sh

verify-factory:
	@chmod +x scripts/verify_factory_docs.sh
	@echo "Running factory docs verifier..."
	@./scripts/verify_factory_docs.sh

verify-forbidden:
	@chmod +x scripts/verify_forbidden_patterns.sh
	@echo "Running forbidden patterns verifier..."
	@./scripts/verify_forbidden_patterns.sh

verify-single-lang:
	@chmod +x scripts/verify_single_language.sh
	@echo "Running single language verifier..."
	@./scripts/verify_single_language.sh

verify-static:
	@chmod +x scripts/verify_static_binary_intent.sh
	@echo "Running static binary intent verifier..."
	@./scripts/verify_static_binary_intent.sh

verify-tooling-boundaries:
	@chmod +x scripts/verify_tooling_boundaries.sh
	@echo "Running tooling boundary verifier..."
	@./scripts/verify_tooling_boundaries.sh

verify-llm-friendly:
	@chmod +x scripts/verify_llm_friendliness.sh
	@echo "Running LLM-friendliness verifier..."
	@./scripts/verify_llm_friendliness.sh

verify-agent-context:
	@chmod +x scripts/verify_agent_context.sh
	@echo "Running agent context verifier..."
	@./scripts/verify_agent_context.sh

verify-git-hooks:
	@echo "Running Git hooks verifier..."
	@go run ./cmd/leamas factory verify git-hooks

verify-domain-boundaries:
	@echo "Running domain boundaries verifier..."
	@go run ./cmd/leamas factory verify domain-boundaries

install-git-hooks:
	@chmod +x scripts/install_git_hooks.sh
	@./scripts/install_git_hooks.sh

# Build test helper binary for adversarial tests
# This ensures tests can run from a clean checkout without manual setup
test-helper:
	@echo "Building test helper..."
	@mkdir -p internal/execution/testdata/testhelper
	@cd internal/execution/testdata/testhelper && go build -o main main.go
	@echo "Done. Test helper: internal/execution/testdata/testhelper/main"

install: build
	@echo "Installing Leamas to $(DESTDIR)$(BINDIR)/leamas"
	@$(INSTALL) -d "$(DESTDIR)$(BINDIR)"
	@$(INSTALL) -m 0755 bin/leamas "$(DESTDIR)$(BINDIR)/leamas"
	@echo "Done. Installed: $(DESTDIR)$(BINDIR)/leamas"

# Release targets

release-build:
	@echo "Building release for version $(VERSION)..."
	@mkdir -p "$(ARTIFACT_DIR)"
	@CGO_ENABLED=0 go build -trimpath \
		-ldflags "$(LDFLAGS) -s -w" \
		-o "$(ARTIFACT_DIR)/leamas" ./cmd/leamas
	@echo "version=$(VERSION)" > "$(ARTIFACT_DIR)/release.txt"
	@echo "commit=$(COMMIT)" >> "$(ARTIFACT_DIR)/release.txt"
	@echo "build_time=$(BUILD_TIME)" >> "$(ARTIFACT_DIR)/release.txt"
	@echo "goos=$(GOOS)" >> "$(ARTIFACT_DIR)/release.txt"
	@echo "goarch=$(GOARCH)" >> "$(ARTIFACT_DIR)/release.txt"
	@echo "Done. Artifact: $(ARTIFACT_DIR)/leamas"

release-checksum:
	@echo "Generating checksums for $(ARTIFACT_DIR)..."
	@if command -v sha256sum >/dev/null 2>&1; then \
		(cd "$(ARTIFACT_DIR)" && sha256sum leamas > SHA256SUMS); \
	elif command -v shasum >/dev/null 2>&1; then \
		(cd "$(ARTIFACT_DIR)" && shasum -a 256 leamas > SHA256SUMS); \
	else \
		echo "ERROR: Neither sha256sum nor shasum found"; \
		exit 1; \
	fi
	@echo "Done. Checksum: $(ARTIFACT_DIR)/SHA256SUMS"

release-verify:
	@echo "Verifying release artifacts..."
	@if [ ! -x "$(ARTIFACT_DIR)/leamas" ]; then \
		echo "ERROR: $(ARTIFACT_DIR)/leamas is not executable"; \
		exit 1; \
	fi
	@$(ARTIFACT_DIR)/leamas version
	@if [ -f "$(ARTIFACT_DIR)/SHA256SUMS" ]; then \
		if command -v sha256sum >/dev/null 2>&1; then \
			(cd "$(ARTIFACT_DIR)" && sha256sum -c SHA256SUMS); \
		elif command -v shasum >/dev/null 2>&1; then \
			(cd "$(ARTIFACT_DIR)" && shasum -a 256 -c SHA256SUMS); \
		else \
			echo "WARNING: Cannot verify checksums (no sha256sum or shasum)"; \
		fi; \
	else \
		echo "WARNING: SHA256SUMS not found, skipping checksum verification"; \
	fi
	@echo "Verification complete."

release-clean:
	@echo "Cleaning release artifacts..."
	@rm -rf "$(DIST_DIR)"
	@echo "Done."

release: release-build release-checksum release-verify
