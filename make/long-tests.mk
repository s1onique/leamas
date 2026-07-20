# make/long-tests.mk - Long-test tier targets
# These targets support the Factory long-test policy enforcement.

# test-fast runs Go tests in fast mode (skips registered long tests)
test-fast:
	@echo "Running go test (fast mode)..."
	@go test -short ./...

# test-long runs all registered long tests using the baseline-driven runner
# Requires build to ensure bin/leamas exists
test-long: build
	@echo "Running registered long tests from baseline..."
	@bin/leamas factory test-long

# gate-fast runs the fast verifier lane using --lane=fast
# This executes all fast-lane verifiers and explicitly skips dupcode verifiers.
gate-fast: build
	@echo "Running quality gate (fast lane)..."
	@./bin/leamas factory gate --lane=fast

# gate-dupcode runs the dupcode verifier lane using --lane=dupcode
# This executes only dupcode and dupcode-baseline verifiers.
gate-dupcode: build
	@echo "Running quality gate (dupcode lane)..."
	@./bin/leamas factory gate --lane=dupcode

# gate is the canonical target that runs the full factory gate with all checks.
# Uses --test-mode=full which runs both fast and long lanes.
gate: build
	@echo "Running quality gate (full mode)..."
	@./bin/leamas factory gate --test-mode=full
