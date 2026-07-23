# make/long-tests.mk - Long-test tier targets
# These targets support the Factory long-test policy enforcement.

.PHONY: test-fast test-long gate-fast gate-dupcode gate gate-canonical gate-context-guard

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

# gate-context-guard checks if the gate should run in the current context.
# Returns 0 to allow, 2 to refuse with diagnostic.
# Uses sequential $(MAKE) invocation to ensure guard runs BEFORE any canonical work.
# Guard order: caller validation → override check → fallback detection → refusal.
# IMPORTANT: Both variables are validated before any allow/refuse logic.
gate-context-guard:
	@case "$${LEAMAS_GATE_CALLER:-}" in \
	""|cline|codium|vscode|editor) ;; \
	*) \
		printf '%s\n' \
			"gate: invalid LEAMAS_GATE_CALLER='$${LEAMAS_GATE_CALLER}'" \
			"gate: expected cline, codium, vscode, editor, or unset" >&2; \
		exit 2; \
		;; \
	esac && \
	case "$${LEAMAS_ALLOW_FULL_GATE:-}" in \
	""|0|1) ;; \
	*) \
		printf '%s\n' \
			"gate: invalid LEAMAS_ALLOW_FULL_GATE='$${LEAMAS_ALLOW_FULL_GATE}'" \
			"gate: expected 0, 1, or an unset variable" >&2; \
		exit 2; \
		;; \
	esac && \
	case "$${LEAMAS_ALLOW_FULL_GATE:-}" in \
	1) exit 0 ;; \
	esac && \
	editor_context=0 && \
	case "$${LEAMAS_GATE_CALLER:-}" in \
	cline|codium|vscode|editor) editor_context=1 ;; \
	esac && \
	case "$${TERM_PROGRAM:-}" in \
	vscode|vscodium|codium) editor_context=1 ;; \
	esac && \
	if [ -n "$${VSCODE_PID:-}" ]; then \
		editor_context=1; \
	fi && \
	if [ "$$editor_context" = 1 ]; then \
		printf '%s\n' \
			"gate: REFUSED in Codium/VS Code/Cline terminal context." \
			"gate: use 'make gate-fast' for interactive verification." \
			"gate: for deliberate canonical verification, run:" \
			"gate:   LEAMAS_ALLOW_FULL_GATE=1 make gate" >&2; \
		exit 2; \
	fi

# gate-canonical is the internal target that runs the full factory gate.
# It is NOT a prerequisite of gate; it is invoked only after gate-context-guard passes.
# This preserves canonical gate behavior: fast lane → dupcode lane → long lane.
gate-canonical: build
	@echo "Running quality gate (full mode)..."
	@./bin/leamas factory gate --test-mode=full

# gate is the public entry point that guards against editor-context execution.
# The guard MUST execute before any canonical work. Sequential $(MAKE) ensures this:
# 1. gate-context-guard runs first; if it fails, gate-canonical never executes.
# 2. gate-canonical runs only if the guard passes.
# 3. Guard failure is non-zero, non-success, and emits no canonical markers.
gate:
	@$(MAKE) --no-print-directory gate-context-guard
	@$(MAKE) --no-print-directory gate-canonical

# factorize-context-guard checks if factorize should run in the current context.
# Returns 0 to allow, 2 to refuse with diagnostic.
# Uses sequential $(MAKE) invocation to ensure guard runs BEFORE any factorize work.
# Guard order: caller validation → override check → fallback detection → refusal.
# IMPORTANT: Both variables are validated before any allow/refuse logic.
factorize-context-guard:
	@case "$${LEAMAS_GATE_CALLER:-}" in \
	""|cline|codium|vscode|editor) ;; \
	*) \
		printf '%s\n' \
			"factorize: invalid LEAMAS_GATE_CALLER='$${LEAMAS_GATE_CALLER}'" \
			"factorize: expected cline, codium, vscode, editor, or unset" >&2; \
		exit 2; \
		;; \
	esac && \
	case "$${LEAMAS_ALLOW_FULL_FACTORIZE:-}" in \
	""|0|1) ;; \
	*) \
		printf '%s\n' \
			"factorize: invalid LEAMAS_ALLOW_FULL_FACTORIZE='$${LEAMAS_ALLOW_FULL_FACTORIZE}'" \
			"factorize: expected 0, 1, or an unset variable" >&2; \
		exit 2; \
		;; \
	esac && \
	case "$${LEAMAS_ALLOW_FULL_FACTORIZE:-}" in \
	1) exit 0 ;; \
	esac && \
	editor_context=0 && \
	case "$${LEAMAS_GATE_CALLER:-}" in \
	cline|codium|vscode|editor) editor_context=1 ;; \
	esac && \
	case "$${TERM_PROGRAM:-}" in \
	vscode|vscodium|codium) editor_context=1 ;; \
	esac && \
	if [ -n "$${VSCODE_PID:-}" ]; then \
		editor_context=1; \
	fi && \
	if [ "$$editor_context" = 1 ]; then \
		printf '%s\n' \
			"factorize: REFUSED in Codium/VS Code/Cline terminal context." \
			"factorize: use 'make gate-fast' for interactive verification." \
			"factorize: for deliberate factorize execution, run:" \
			"factorize:   LEAMAS_ALLOW_FULL_FACTORIZE=1 make factorize" >&2; \
		exit 2; \
	fi

# factorize-canonical is the canonical factorize entry point.
# It depends on factorize to apply the guard first.
# Both LEAMAS_ALLOW_FULL_FACTORIZE=1 make factorize and
# LEAMAS_ALLOW_FULL_FACTORIZE=1 make factorize-canonical will work.
factorize-canonical: factorize

# FACTORIZE_COMMAND is the test/build override seam.
# Production: executes the real multi-minute factorize verifier suite.
# Tests: override with a bounded sentinel (e.g., touch sentinel && echo done).
FACTORIZE_COMMAND ?= go run ./cmd/leamas factory factorize

# factorize is the public entry point that guards against editor-context execution.
# The guard MUST execute before any factorize work. Sequential $(MAKE) ensures this:
# 1. factorize-context-guard runs first; if it fails, the real command never executes.
# 2. Real work executes only if the guard passes.
# 3. Guard failure is non-zero, non-success, and emits no factorize markers.
factorize:
	@$(MAKE) --no-print-directory factorize-context-guard
	@echo "Running factory factorize..."
	@chmod +x scripts/verify_*.sh
	@$(FACTORIZE_COMMAND)
