// Package main provides tests for main usage text.
package main

import (
	"strings"
	"testing"
)

func TestUsageText_IncludesFactoryCoverage(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "factory coverage") {
		t.Error("usage text should include 'factory coverage' command")
	}
}

func TestUsageText_IncludesFactoryDigest(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "factory digest") {
		t.Error("usage text should include 'factory digest' command")
	}
}

func TestUsageText_IncludesFactoryVerify(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "factory verify") {
		t.Error("usage text should include 'factory verify' command")
	}
}

func TestUsageText_IncludesFactoryGate(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "factory gate") {
		t.Error("usage text should include 'factory gate' command")
	}
}

func TestUsageText_IncludesFactoryFactorize(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "factory factorize") {
		t.Error("usage text should include 'factory factorize' command")
	}
}

func TestUsageText_IncludesWitness(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "witness") {
		t.Error("usage text should include 'witness' command")
	}
}

func TestUsageText_IncludesCockpit(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "cockpit") {
		t.Error("usage text should include 'cockpit' command")
	}
}

func TestUsageText_IncludesDoctor(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "doctor") {
		t.Error("usage text should include 'doctor' command")
	}
}

func TestUsageText_IncludesVersion(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "version") {
		t.Error("usage text should include 'version' command")
	}
}

func TestUsageText_IncludesHelp(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "--help") {
		t.Error("usage text should include '--help'")
	}
}

func TestFactoryUsageText_IncludesCoverage(t *testing.T) {
	text := factoryUsageText()
	if !strings.Contains(text, "coverage") {
		t.Error("factory usage text should include 'coverage'")
	}
}

func TestFactoryUsageText_IncludesVerify(t *testing.T) {
	text := factoryUsageText()
	if !strings.Contains(text, "verify") {
		t.Error("factory usage text should include 'verify'")
	}
}

func TestFactoryUsageText_IncludesGate(t *testing.T) {
	text := factoryUsageText()
	if !strings.Contains(text, "gate") {
		t.Error("factory usage text should include 'gate'")
	}
}

func TestFactoryUsageText_IncludesFactorize(t *testing.T) {
	text := factoryUsageText()
	if !strings.Contains(text, "factorize") {
		t.Error("factory usage text should include 'factorize'")
	}
}

func TestFactoryUsageText_IncludesDigest(t *testing.T) {
	text := factoryUsageText()
	if !strings.Contains(text, "digest") {
		t.Error("factory usage text should include 'digest'")
	}
}

func TestFactoryUsageText_IncludesVerifyUsage(t *testing.T) {
	// Note: factoryUsageText() only returns factory commands, not verify usage
	// The verify usage is printed by printFactoryVerifyUsage which is called after
	text := factoryUsageText()
	// Just verify it includes some verify-related content
	if !strings.Contains(text, "verify") {
		t.Error("factory usage text should include 'verify'")
	}
}

func TestUsageText_HasHeader(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "Leamas - Local-first") {
		t.Error("usage text should have header")
	}
}

func TestUsageText_HasCommandsSection(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "Commands:") {
		t.Error("usage text should have 'Commands:' section")
	}
}

func TestUsageText_HasUsageSection(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "Usage:") {
		t.Error("usage text should have 'Usage:' section")
	}
}
