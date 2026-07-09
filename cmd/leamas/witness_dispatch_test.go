// Package main provides tests for witness command dispatch.
package main

import (
	"bytes"
	"strings"
	"testing"
)

// ============================================================================
// parseWitnessCommand tests
// ============================================================================

func TestParseWitnessCommand_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantCmd         string
		wantErr         bool
		wantErrContains string
	}{
		// Success cases
		{"proxy", []string{"proxy"}, "proxy", false, ""},
		{"run-bundle", []string{"run-bundle"}, "run-bundle", false, ""},
		{"claim", []string{"claim"}, "claim", false, ""},
		{"evidence", []string{"evidence"}, "evidence", false, ""},
		// Error cases
		{"empty args", []string{}, "", true, "missing witness command"},
		{"empty string", []string{""}, "", true, "unknown witness command"},
		{"random word", []string{"random"}, "", true, "unknown witness command"},
		{"case mismatch proxy", []string{"Proxy"}, "", true, "unknown witness command"},
		{"case mismatch claim", []string{"CLAIM"}, "", true, "unknown witness command"},
		{"extra args ignored", []string{"proxy", "extra"}, "proxy", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWitnessCommand(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != tt.wantCmd {
				t.Errorf("command = %q, want %q", got.Name, tt.wantCmd)
			}
		})
	}
}

func TestParseWitnessCommand_KnownCommands(t *testing.T) {
	knownCommands := []string{"proxy", "run-bundle", "claim", "evidence"}

	for _, cmd := range knownCommands {
		result, err := parseWitnessCommand([]string{cmd})
		if err != nil {
			t.Errorf("expected no error for known command '%s', got: %v", cmd, err)
		}
		if result.Name != cmd {
			t.Errorf("expected '%s', got '%s'", cmd, result.Name)
		}
	}
}

func TestParseWitnessCommand_RejectsUnknownCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"empty string", []string{""}},
		{"random word", []string{"random"}},
		{"partial match", []string{"clai"}},
		{"extra chars", []string{"proxys"}},
		{"single char", []string{"p"}},
		{"numbers", []string{"123"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseWitnessCommand(tt.args)
			if err == nil {
				t.Errorf("expected error for unknown command '%s'", tt.args[0])
			}
		})
	}
}

// ============================================================================
// printWitnessUsageTo tests
// ============================================================================

func TestPrintWitnessUsageTo(t *testing.T) {
	var buf bytes.Buffer
	printWitnessUsageTo(&buf)

	output := buf.String()
	if !strings.Contains(output, "Witness commands:") {
		t.Error("output should contain 'Witness commands:'")
	}
	if !strings.Contains(output, "leamas witness proxy") {
		t.Error("output should mention proxy command")
	}
	if !strings.Contains(output, "leamas witness run-bundle") {
		t.Error("output should mention run-bundle command")
	}
	if !strings.Contains(output, "leamas witness claim") {
		t.Error("output should mention claim command")
	}
	if !strings.Contains(output, "leamas witness evidence") {
		t.Error("output should mention evidence command")
	}
}

// ============================================================================
// printWitnessProxyUsageTo tests
// ============================================================================

func TestPrintWitnessProxyUsageTo(t *testing.T) {
	var buf bytes.Buffer
	printWitnessProxyUsageTo(&buf)

	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Error("output should contain 'Usage:'")
	}
	if !strings.Contains(output, "--upstream") {
		t.Error("output should mention --upstream flag")
	}
	if !strings.Contains(output, "--listen") {
		t.Error("output should mention --listen flag")
	}
	if !strings.Contains(output, "--max-records") {
		t.Error("output should mention --max-records flag")
	}
	if !strings.Contains(output, "--capture-headers") {
		t.Error("output should mention --capture-headers flag")
	}
}

// ============================================================================
// runWitness tests
// ============================================================================

func TestRunWitness_MissingSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := witnessDeps{}

	code := runWitness([]string{}, &stdout, &stderr, deps)

	if code == 0 {
		t.Error("missing subcommand should exit non-zero")
	}
	if !strings.Contains(stderr.String(), "Witness commands:") {
		t.Errorf("stderr should print usage, got: %s", stderr.String())
	}
}

func TestRunWitness_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := witnessDeps{}

	code := runWitness([]string{"unknown"}, &stdout, &stderr, deps)

	if code == 0 {
		t.Error("unknown subcommand should exit non-zero")
	}
	if !strings.Contains(stderr.String(), "unknown witness command") {
		t.Errorf("stderr should mention unknown command, got: %s", stderr.String())
	}
}

func TestRunWitness_HelpFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := witnessDeps{}

	code := runWitness([]string{"--help"}, &stdout, &stderr, deps)

	if code != 0 {
		t.Errorf("help should exit 0, got %d", code)
	}
	if stdout.String() == "" && stderr.String() == "" {
		t.Error("help should produce output")
	}
}

func TestRunWitness_ClaimSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := witnessDeps{}

	// Claim without subcommand should trigger usage
	code := runWitness([]string{"claim"}, &stdout, &stderr, deps)

	if code == 0 {
		t.Error("missing claim subcommand should exit non-zero")
	}
}
