package main

import (
	"strings"
	"testing"
)

// TestParseFactoryCommandRemovesBootstrapDoctor proves the dead
// dispatcher entries are no longer recognized.
func TestParseFactoryCommandRemovesBootstrapDoctor(t *testing.T) {
	for _, name := range []string{"bootstrap", "doctor"} {
		t.Run(name, func(t *testing.T) {
			_, err := parseFactoryCommand([]string{name})
			if err == nil {
				t.Fatalf("parseFactoryCommand(%q) = nil error; want unknown command", name)
			}
			if !strings.Contains(err.Error(), "unknown factory command") {
				t.Errorf("err = %v; want unknown factory command", err)
			}
		})
	}
}

// TestParseFactoryCommandKnownCommands proves the existing commands
// remain recognized.
func TestParseFactoryCommandKnownCommands(t *testing.T) {
	for _, name := range []string{
		"verify", "gate", "factorize", "digest", "coverage",
		"gate-summary", "output-contract", "doctrine", "test-long", "close",
	} {
		t.Run(name, func(t *testing.T) {
			got, err := parseFactoryCommand([]string{name})
			if err != nil {
				t.Fatalf("parseFactoryCommand(%q) err = %v", name, err)
			}
			if got != name {
				t.Errorf("parseFactoryCommand(%q) = %q; want %q", name, got, name)
			}
		})
	}
}

// TestParseFactoryCommandMissingArgs proves empty args produce a useful error.
func TestParseFactoryCommandMissingArgs(t *testing.T) {
	_, err := parseFactoryCommand([]string{})
	if err == nil {
		t.Fatal("expected error for missing command")
	}
	if !strings.Contains(err.Error(), "missing factory command") {
		t.Errorf("err = %v; want missing factory command", err)
	}
}

// TestParseFactoryCommandUnknownCommand proves unknown commands fail.
func TestParseFactoryCommandUnknownCommand(t *testing.T) {
	_, err := parseFactoryCommand([]string{"unknown-thing"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "unknown factory command") {
		t.Errorf("err = %v; want unknown factory command", err)
	}
}

// TestUsageTextOmitsBootstrapDoctor proves help/usage text does not
// advertise the removed commands.
func TestUsageTextOmitsBootstrapDoctor(t *testing.T) {
	text := usageText()
	if strings.Contains(text, "leamas doctor") {
		t.Error("usageText still advertises leamas doctor")
	}
	if strings.Contains(text, "leamas bootstrap") {
		t.Error("usageText still advertises leamas bootstrap")
	}
}

// TestFactoryUsageTextOmitsBootstrapDoctor proves factory subcommand
// usage text does not advertise the removed commands.
func TestFactoryUsageTextOmitsBootstrapDoctor(t *testing.T) {
	text := factoryUsageText()
	if strings.Contains(text, "bootstrap") {
		t.Errorf("factoryUsageText still mentions bootstrap: %s", text)
	}
	if strings.Contains(text, "doctor") {
		t.Errorf("factoryUsageText still mentions doctor: %s", text)
	}
}
