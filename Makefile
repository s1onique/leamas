# Leamas Makefile

.PHONY: help gate test clean

help:
	@echo "Leamas - Make targets"
	@echo ""
	@echo "  make gate   - Run quality gate (documentation + tests)"
	@echo "  make test   - Run Go tests (if module exists)"
	@echo "  make clean  - Clean build artifacts"
	@echo "  make help   - Show this help"

gate: scripts/quality_gate.sh
	@echo "Running quality gate..."
	chmod +x scripts/quality_gate.sh
	./scripts/quality_gate.sh

test:
	@if [ -f go.mod ]; then \
		echo "Running go test..."; \
		go test ./...; \
	else \
		echo "Go module not initialized yet; skipping go test."; \
	fi

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@find . -name '*.test' -delete 2>/dev/null || true
	@echo "Done."
