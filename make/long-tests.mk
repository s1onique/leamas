# make/long-tests.mk - Long-test tier targets
# These targets support the Factory long-test policy enforcement.

# test-fast runs Go tests in fast mode (skips registered long tests)
test-fast:
	@echo "Running go test (fast mode)..."
	@go test -short ./...

# test-long runs all registered long tests using the baseline-driven runner
test-long:
	@echo "Running registered long tests from baseline..."
	@bin/leamas factory test-long

# gate-fast runs the full quality gate in fast mode using --test-mode=short
# which skips long-running tests that are registered in .factory/long-tests-baseline.json
gate-fast:
	@echo "Running quality gate (fast mode)..."
	@chmod +x scripts/quality_gate.sh
	@./scripts/quality_gate.sh --test-mode=short

# gate is the canonical target that aggregates fast and long lanes.
# Both lanes must pass for the gate to be considered green.
gate: gate-fast test-long
	@echo "Complete gate: fast and long lanes passed"
