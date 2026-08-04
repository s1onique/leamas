// SPDX-License-Identifier: Apache-2.0

package protectedverifier

import (
	"errors"
	"testing"

	"github.com/s1onique/leamas/internal/factory/dupcode"
)

func providerIsolationInput() DupcodeInput {
	cfg := dupcode.DefaultConfig()
	cfg.Root = "."
	cfg.ExcludeDirs = []string{"bound-dir"}
	cfg.ExcludeFileSuffixes = []string{".bound.go"}
	return DupcodeInput{
		Root:      ".",
		MinLines:  cfg.MinLines,
		MinTokens: cfg.MinTokens,
		Config:    cfg,
	}
}

func providerIsolationFindings() []dupcode.Finding {
	return []dupcode.Finding{{
		Fingerprint:       "short",
		StableFingerprint: "stable",
		TokenCount:        400,
		LineCount:         40,
		Occurrences: []dupcode.Occurrence{{
			Path:      "original.go",
			StartLine: 1,
			EndLine:   40,
		}},
	}}
}

func TestDupcodeProviderInputIdentityCallerMutation(t *testing.T) {
	input := providerIsolationInput()
	provider := NewDupcodeAnalysisProvider(input, func(_ string, _ dupcode.Config) ([]dupcode.Finding, error) {
		return providerIsolationFindings(), nil
	})

	input.Config.ExcludeDirs[0] = "caller-mutated-dir"
	input.Config.ExcludeFileSuffixes[0] = ".caller-mutated.go"

	if got := provider.input.Config.ExcludeDirs[0]; got != "bound-dir" {
		t.Fatalf("bound ExcludeDirs = %q, want %q", got, "bound-dir")
	}
	if got := provider.input.Config.ExcludeFileSuffixes[0]; got != ".bound.go" {
		t.Fatalf("bound ExcludeFileSuffixes = %q, want %q", got, ".bound.go")
	}

	bound := providerIsolationInput()
	if _, err := provider.ConsumedBy("dupcode", bound); err != nil {
		t.Fatalf("ConsumedBy with construction-time identity: %v", err)
	}
}

func TestDupcodeProviderAnalyzerConfigMutationPreservesInputIdentity(t *testing.T) {
	input := providerIsolationInput()
	calls := 0
	provider := NewDupcodeAnalysisProvider(input, func(_ string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		calls++
		cfg.ExcludeDirs[0] = "analyzer-mutated-dir"
		cfg.ExcludeFileSuffixes[0] = ".analyzer-mutated.go"
		return providerIsolationFindings(), nil
	})

	first, err := provider.ConsumedBy("dupcode", providerIsolationInput())
	if err != nil {
		t.Fatalf("first ConsumedBy: %v", err)
	}
	second, err := provider.ConsumedBy("dupcode-baseline", providerIsolationInput())
	if err != nil {
		t.Fatalf("second ConsumedBy: %v", err)
	}
	if calls != 1 {
		t.Fatalf("analyzer calls = %d, want 1", calls)
	}
	if got := provider.input.Config.ExcludeDirs[0]; got != "bound-dir" {
		t.Fatalf("provider ExcludeDirs after analyzer mutation = %q", got)
	}
	if got := provider.input.Config.ExcludeFileSuffixes[0]; got != ".bound.go" {
		t.Fatalf("provider ExcludeFileSuffixes after analyzer mutation = %q", got)
	}
	if got := first.Config.ExcludeDirs[0]; got != "bound-dir" {
		t.Fatalf("first result ExcludeDirs = %q", got)
	}
	if got := second.Config.ExcludeFileSuffixes[0]; got != ".bound.go" {
		t.Fatalf("second result ExcludeFileSuffixes = %q", got)
	}
}

func TestDupcodeProviderConsumerIsolationDeepCopy(t *testing.T) {
	input := providerIsolationInput()
	provider := NewDupcodeAnalysisProvider(input, func(_ string, _ dupcode.Config) ([]dupcode.Finding, error) {
		return providerIsolationFindings(), nil
	})

	consumerA, err := provider.ConsumedBy("dupcode", providerIsolationInput())
	if err != nil {
		t.Fatalf("consumer A: %v", err)
	}
	consumerB, err := provider.ConsumedBy("dupcode-baseline", providerIsolationInput())
	if err != nil {
		t.Fatalf("consumer B: %v", err)
	}
	if consumerA == consumerB {
		t.Fatal("consumer analyses share a top-level pointer")
	}

	consumerA.Config.ExcludeDirs[0] = "mutated-dir"
	consumerA.Config.ExcludeFileSuffixes[0] = ".mutated.go"
	consumerA.Findings[0].Occurrences[0].Path = "mutated-finding.go"
	consumerA.Occurrences[0].Path = "mutated-analysis.go"
	consumerA.Findings = append(consumerA.Findings, dupcode.Finding{Fingerprint: "appended"})

	assertIsolatedAnalysis(t, "consumer B", consumerB)

	consumerB.Config.ExcludeDirs[0] = "consumer-b-dir"
	consumerB.Config.ExcludeFileSuffixes[0] = ".consumer-b.go"
	consumerB.Findings[0].Occurrences[0].Path = "consumer-b-finding.go"
	consumerB.Occurrences[0].Path = "consumer-b-analysis.go"
	consumerB.Findings = append(consumerB.Findings, dupcode.Finding{Fingerprint: "consumer-b-appended"})

	third, err := provider.ConsumedBy("third-consumer", providerIsolationInput())
	if err != nil {
		t.Fatalf("third consumer: %v", err)
	}
	assertIsolatedAnalysis(t, "third consumer", third)
}

func assertIsolatedAnalysis(t *testing.T, label string, analysis *DupcodeAnalysis) {
	t.Helper()
	if got := analysis.Config.ExcludeDirs[0]; got != "bound-dir" {
		t.Errorf("%s ExcludeDirs = %q", label, got)
	}
	if got := analysis.Config.ExcludeFileSuffixes[0]; got != ".bound.go" {
		t.Errorf("%s ExcludeFileSuffixes = %q", label, got)
	}
	if len(analysis.Findings) != 1 {
		t.Errorf("%s findings length = %d, want 1", label, len(analysis.Findings))
		return
	}
	if got := analysis.Findings[0].Occurrences[0].Path; got != "original.go" {
		t.Errorf("%s finding occurrence path = %q", label, got)
	}
	if got := analysis.Occurrences[0].Path; got != "original.go" {
		t.Errorf("%s analysis occurrence path = %q", label, got)
	}
}

func TestDupcodeProviderFailureStable(t *testing.T) {
	boom := errors.New("scan boom")
	calls := 0
	provider := NewDupcodeAnalysisProvider(providerIsolationInput(), func(_ string, _ dupcode.Config) ([]dupcode.Finding, error) {
		calls++
		return nil, boom
	})

	for _, consumer := range []string{"dupcode", "dupcode-baseline", "third"} {
		_, err := provider.ConsumedBy(consumer, providerIsolationInput())
		if !errors.Is(err, boom) {
			t.Fatalf("%s error = %v, want errors.Is(scan boom)", consumer, err)
		}
	}
	if calls != 1 {
		t.Fatalf("analyzer calls = %d, want 1", calls)
	}
}
