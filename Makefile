# Leamas Makefile

.PHONY: help gate test clean digest factorize verify-doctrine verify-factory verify-forbidden verify-single-lang verify-static verify-agent-doctrine verify-tooling-boundaries

# Colors
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m

help:
	@echo "Leamas - Make targets"
	@echo ""
	@echo "  make gate           - Run quality gate (documentation + tests + Go checks)"
	@echo "  make test           - Run Go tests (if module exists)"
	@echo "  make digest         - Generate targeted digest of staged changes"
	@echo "  make factorize      - Run factory verifiers (doctrine, factory docs, patterns)"
	@echo "  make verify-*       - Run individual verifiers:"
	@echo "    make verify-agent-doctrine  Agent doctrine contract check"
	@echo "    make verify-doctrine         Doctrine inventory check"
	@echo "    make verify-factory          Factory docs check"
	@echo "    make verify-forbidden        Forbidden patterns check"
	@echo "    make verify-single-lang      Single language check"
	@echo "    make verify-static           Static binary intent check"
	@echo "    make verify-tooling-boundaries Tooling language boundaries check"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make help           - Show this help"

gate: scripts/quality_gate.sh
	@echo "Running quality gate..."
	@chmod +x scripts/quality_gate.sh
	./scripts/quality_gate.sh

test:
	@if [ -f go.mod ]; then \
		echo "Running go test..."; \
		go test ./...; \
	else \
		echo "Go module not initialized yet; skipping go test."; \
	fi

digest:
	@if [ -f scripts/make_targeted_digest.sh ]; then \
		chmod +x scripts/make_targeted_digest.sh; \
		./scripts/make_targeted_digest.sh --staged --output /tmp/digest.md; \
		cat /tmp/digest.md; \
	else \
		echo "make_targeted_digest.sh not found"; \
		exit 1; \
	fi

factorize: verify-agent-doctrine verify-doctrine verify-factory verify-forbidden verify-single-lang verify-static verify-tooling-boundaries
	@echo ""
	@echo -e "$(GREEN)=========================================="
	@echo -e "Factory factorize PASSED"
	@echo -e "==========================================$(NC)"

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

# Build target with static linking
build:
	@echo "Building Leamas (static binary)..."
	@CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
	@echo "Done. Binary: bin/leamas"

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f leamas
	@find . -name '*.test' -delete 2>/dev/null || true
	@echo "Done."
