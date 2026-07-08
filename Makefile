# Leamas Makefile

.PHONY: help gate test clean digest factorize verify-doctrine verify-factory
.PHONY: verify-forbidden verify-single-lang verify-static verify-agent-doctrine
.PHONY: verify-tooling-boundaries verify-llm-friendly verify-agent-context
.PHONY: verify-git-hooks install-git-hooks build digest install

# Install variables (GNU conventions)
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
INSTALL ?= install

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
	@CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
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

install-git-hooks:
	@chmod +x scripts/install_git_hooks.sh
	@./scripts/install_git_hooks.sh

install: build
	@echo "Installing Leamas to $(DESTDIR)$(BINDIR)/leamas"
	@$(INSTALL) -d "$(DESTDIR)$(BINDIR)"
	@$(INSTALL) -m 0755 bin/leamas "$(DESTDIR)$(BINDIR)/leamas"
	@echo "Done. Installed: $(DESTDIR)$(BINDIR)/leamas"
