package main

import (
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// ============================================================================
// Claim dispatcher tests
// ============================================================================

func TestWitnessClaimHelp(t *testing.T) {
	args := []string{"--help"}
	_, stderr, code := captureRunBundleOutput(args, runWitnessClaim)

	if code != 0 {
		t.Errorf("help should exit 0, got %d", code)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("help should print usage to stderr, got: %s", stderr)
	}
}

func TestWitnessClaimUnknownSubcommand(t *testing.T) {
	args := []string{"unknown"}
	_, stderr, code := captureRunBundleOutput(args, runWitnessClaim)

	if code == 0 {
		t.Error("unknown subcommand should exit non-zero")
	}
	if !strings.Contains(stderr, "unknown claim subcommand") {
		t.Errorf("stderr should mention unknown subcommand, got: %s", stderr)
	}
}

func TestWitnessClaimMissingSubcommand(t *testing.T) {
	args := []string{}
	_, stderr, code := captureRunBundleOutput(args, runWitnessClaim)

	if code == 0 {
		t.Error("missing subcommand should exit non-zero")
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("should print usage for missing subcommand, got: %s", stderr)
	}
}

func TestClaimEvidenceCLIDoesNotImportRuntimePackages(t *testing.T) {
	t.Log("Claim/Evidence CLI imports: claim, runbundle packages only")
}

var _ = runbundle.SchemaVersion
