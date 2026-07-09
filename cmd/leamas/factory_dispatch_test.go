// Package main provides tests for factory command dispatch.
package main

import (
	"strings"
	"testing"
)

func TestParseFactoryCommand_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantCmd         string
		wantErr         bool
		wantErrContains string
	}{
		// Success cases
		{"verify", []string{"verify"}, "verify", false, ""},
		{"gate", []string{"gate"}, "gate", false, ""},
		{"factorize", []string{"factorize"}, "factorize", false, ""},
		{"digest", []string{"digest"}, "digest", false, ""},
		{"coverage", []string{"coverage"}, "coverage", false, ""},
		// Error cases
		{"empty args", []string{}, "", true, "missing factory command"},
		{"empty string", []string{""}, "", true, "unknown factory command"},
		{"random word", []string{"random"}, "", true, "unknown factory command"},
		{"partial verify", []string{"verif"}, "", true, "unknown factory command"},
		{"partial gate", []string{"gat"}, "", true, "unknown factory command"},
		{"case mismatch verify", []string{"Verify"}, "", true, "unknown factory command"},
		{"case mismatch gate", []string{"GATE"}, "", true, "unknown factory command"},
		{"extra args ignored", []string{"verify", "extra"}, "verify", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFactoryCommand(tt.args)

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
			if got != tt.wantCmd {
				t.Errorf("command = %q, want %q", got, tt.wantCmd)
			}
		})
	}
}

func TestParseFactoryCommand_KnownCommands(t *testing.T) {
	knownCommands := []string{"verify", "gate", "factorize", "digest", "coverage"}

	for _, cmd := range knownCommands {
		result, err := parseFactoryCommand([]string{cmd})
		if err != nil {
			t.Errorf("expected no error for known command '%s', got: %v", cmd, err)
		}
		if result != cmd {
			t.Errorf("expected '%s', got '%s'", cmd, result)
		}
	}
}

func TestParseFactoryCommand_UnknownCommandVariants(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"empty string", []string{""}},
		{"random word", []string{"random"}},
		{"partial match", []string{"verif"}},
		{"extra chars", []string{"verifyx"}},
		{"single char", []string{"v"}},
		{"numbers", []string{"123"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFactoryCommand(tt.args)
			if err == nil {
				t.Errorf("expected error for unknown command '%s'", tt.args[0])
			}
		})
	}
}

func TestParseFactoryCommand_ErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantContains string
	}{
		{"empty args", []string{}, "missing factory command"},
		{"empty string", []string{""}, "unknown factory command"},
		{"random word", []string{"random"}, "unknown factory command: random"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFactoryCommand(tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantContains)
			}
		})
	}
}

// TestHandleFactory_UsesParseFactoryCommandContract verifies that parseFactoryCommand
// returns the same command names that handleFactory dispatches on.
func TestHandleFactory_UsesParseFactoryCommandContract(t *testing.T) {
	// Known factory commands from handleFactory switch statement
	knownCommands := []string{"verify", "gate", "factorize", "digest", "coverage"}

	for _, cmd := range knownCommands {
		t.Run(cmd, func(t *testing.T) {
			result, err := parseFactoryCommand([]string{cmd})
			if err != nil {
				t.Errorf("parseFactoryCommand should recognize %q (used in handleFactory)", cmd)
			}
			if result != cmd {
				t.Errorf("parseFactoryCommand(%q) = %q, want %q", cmd, result, cmd)
			}
		})
	}
}

// TestParseFactoryCommand_ContractWithHandleFactory verifies that unknown commands
// from parseFactoryCommand match the default case in handleFactory
func TestParseFactoryCommand_ContractWithHandleFactory(t *testing.T) {
	unknownCommands := []string{"", "invalid", "random", "verifyx"}

	for _, cmd := range unknownCommands {
		t.Run("unknown-"+cmd, func(t *testing.T) {
			_, err := parseFactoryCommand([]string{cmd})
			if err == nil {
				t.Errorf("parseFactoryCommand should reject unknown command %q", cmd)
			}
		})
	}
}
