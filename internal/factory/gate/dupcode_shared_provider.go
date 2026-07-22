// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"fmt"
	"slices"
	"sync"

	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// DupcodeAnalyzer is the function type for running dupcode analysis.
// Tests can inject a counting or failing analyzer via this interface.
type DupcodeAnalyzer func(root string, cfg dupcode.Config) ([]dupcode.Finding, error)

// DupcodeAnalysis represents the immutable result of a dupcode scan.
// Findings are deep-copied on publication to prevent mutation across consumers.
type DupcodeAnalysis struct {
	Findings   []dupcode.Finding
	Root       string
	MinLines   int
	MinTokens  int
	Executions int // number of actual analyzer calls (always 1 in production)
	Consumers  int // number of consumers that received this result
}

// DupcodeInput captures the effective configuration for the analysis.
// Two analyses can only share results when their inputs are identical.
// All fields are deep-copied on construction to prevent caller mutation.
type DupcodeInput struct {
	Config dupcode.Config
}

// effectiveDupcodeConfig returns the canonical effective configuration.
// Nil values are replaced with defaults; empty values remain empty.
func effectiveDupcodeConfig(cfg dupcode.Config) dupcode.Config {
	defaults := dupcode.DefaultConfig()

	if cfg.MinLines == 0 {
		cfg.MinLines = defaults.MinLines
	}
	if cfg.MinTokens == 0 {
		cfg.MinTokens = defaults.MinTokens
	}
	if cfg.ExcludeDirs == nil {
		cfg.ExcludeDirs = defaults.ExcludeDirs
	}
	if cfg.ExcludeFileSuffixes == nil {
		cfg.ExcludeFileSuffixes = defaults.ExcludeFileSuffixes
	}
	// Note: IgnoreGenerated has a zero value of false; nil and false are equivalent

	return cloneConfig(cfg)
}

// cloneConfig creates a deep copy of a dupcode config to prevent mutation.
func cloneConfig(cfg dupcode.Config) dupcode.Config {
	cloned := cfg
	cloned.ExcludeDirs = slices.Clone(cfg.ExcludeDirs)
	cloned.ExcludeFileSuffixes = slices.Clone(cfg.ExcludeFileSuffixes)
	return cloned
}

// newDupcodeInput creates a new DupcodeInput with canonicalized and cloned config.
func newDupcodeInput(cfg dupcode.Config) DupcodeInput {
	return DupcodeInput{Config: effectiveDupcodeConfig(cfg)}
}

// equal returns true when two inputs have identical effective configuration.
// Nil and empty slices are treated as equal (slices.Equal behavior).
func (i DupcodeInput) equal(other DupcodeInput) bool {
	if i.Config.Root != other.Config.Root {
		return false
	}
	if i.Config.MinLines != other.Config.MinLines {
		return false
	}
	if i.Config.MinTokens != other.Config.MinTokens {
		return false
	}
	if !slices.Equal(i.Config.ExcludeDirs, other.Config.ExcludeDirs) {
		return false
	}
	if !slices.Equal(i.Config.ExcludeFileSuffixes, other.Config.ExcludeFileSuffixes) {
		return false
	}
	if i.Config.IgnoreGenerated != other.Config.IgnoreGenerated {
		return false
	}
	return true
}

// state represents the current state of the shared analysis.
type state int

const (
	stateEmpty state = iota
	stateScanning
	stateSuccess
	stateFailed
)

// DupcodeAnalysisProvider manages a single-scan context for dupcode analysis
// within one factorize invocation. It implements single-flight semantics:
// the first consumer performs the scan, subsequent consumers receive the same
// result without re-scanning.
//
// The provider is safe for concurrent calls.
// Calls are serialized because analyzer execution occurs while the mutex is held.
// stateScanning is a defensive invariant for internal state corruption.
type DupcodeAnalysisProvider struct {
	input      DupcodeInput
	analyzer   DupcodeAnalyzer
	mu         sync.Mutex
	state      state
	result     *DupcodeAnalysis
	err        error
	executions int // track actual analyzer calls for test verification
}

// NewDupcodeAnalysisProvider creates a new provider for one factorize invocation.
// The analyzer parameter allows tests to inject a counting or controlled analyzer.
// Production code should pass dupcode.CheckRepo as the analyzer.
//
// The provider clones and canonicalizes the input configuration at construction time,
// ensuring immutability regardless of how the input is passed.
func NewDupcodeAnalysisProvider(input DupcodeInput, analyzer DupcodeAnalyzer) *DupcodeAnalysisProvider {
	if analyzer == nil {
		analyzer = dupcode.CheckRepo
	}
	// Canonicalize and clone the input to ensure immutability
	bound := DupcodeInput{Config: effectiveDupcodeConfig(input.Config)}
	return &DupcodeAnalysisProvider{
		input:      bound,
		analyzer:   analyzer,
		state:      stateEmpty,
		executions: 0,
	}
}

// ConsumedBy returns the analysis result for a verifier. It performs the scan
// on the first call and returns the memoized result for subsequent calls.
// Returns an error if the input does not match the provider's bound configuration.
//
// The input is canonicalized before comparison to ensure that semantically
// equivalent configurations (e.g., nil vs default exclusions) are treated equally.
func (p *DupcodeAnalysisProvider) ConsumedBy(consumerName string, input DupcodeInput) (*DupcodeAnalysis, error) {
	// Canonicalize the requested input before comparison
	requested := DupcodeInput{Config: effectiveDupcodeConfig(input.Config)}

	// Configuration mismatch: refuse to share results for different inputs
	if !p.input.equal(requested) {
		return nil, fmt.Errorf("dupcode analysis input mismatch for %s: "+
			"got Config.Root=%q MinLines=%d MinTokens=%d ExcludeDirs=%v ExcludeFileSuffixes=%v IgnoreGenerated=%v, "+
			"want Config.Root=%q MinLines=%d MinTokens=%d ExcludeDirs=%v ExcludeFileSuffixes=%v IgnoreGenerated=%v",
			consumerName,
			requested.Config.Root, requested.Config.MinLines, requested.Config.MinTokens,
			requested.Config.ExcludeDirs, requested.Config.ExcludeFileSuffixes, requested.Config.IgnoreGenerated,
			p.input.Config.Root, p.input.Config.MinLines, p.input.Config.MinTokens,
			p.input.Config.ExcludeDirs, p.input.Config.ExcludeFileSuffixes, p.input.Config.IgnoreGenerated)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.state {
	case stateEmpty:
		// First consumer: perform the scan
		p.state = stateScanning
		p.executions++
		// Clone the config to prevent caller mutation after this point
		cfg := cloneConfig(p.input.Config)
		cfg.Root = p.input.Config.Root
		findings, err := p.analyzer(p.input.Config.Root, cfg)
		if err != nil {
			p.state = stateFailed
			p.err = err
			return nil, err
		}
		// Deep-copy findings to prevent mutation across consumers.
		// We store the copy internally but return a new copy to the consumer
		// so that the consumer's returned slice doesn't share the same
		// underlying array as our internal storage.
		internalCopy := deepCopyFindings(findings)
		p.result = &DupcodeAnalysis{
			Findings:   internalCopy,
			Root:       p.input.Config.Root,
			MinLines:   p.input.Config.MinLines,
			MinTokens:  p.input.Config.MinTokens,
			Executions: 1,
			Consumers:  1,
		}
		p.state = stateSuccess
		// Return a copy so the consumer's reference doesn't alias our internal storage
		return &DupcodeAnalysis{
			Findings:   deepCopyFindings(findings),
			Root:       p.input.Config.Root,
			MinLines:   p.input.Config.MinLines,
			MinTokens:  p.input.Config.MinTokens,
			Executions: 1,
			Consumers:  1,
		}, nil

	case stateScanning:
		// This state is unreachable because the mutex is held during scan.
		// If reached, it indicates a programming error.
		return nil, fmt.Errorf("dupcode analysis: concurrent scan detected (programming error)")

	case stateSuccess:
		// Subsequent consumers: return deep-copied result to prevent mutation
		p.result.Consumers++
		copied := &DupcodeAnalysis{
			Findings:   deepCopyFindings(p.result.Findings),
			Root:       p.result.Root,
			MinLines:   p.result.MinLines,
			MinTokens:  p.result.MinTokens,
			Executions: p.result.Executions,
			Consumers:  1, // Each call gets its own copy
		}
		return copied, nil

	case stateFailed:
		// Return memoized error
		return nil, p.err
	}

	return nil, fmt.Errorf("dupcode analysis: unexpected state %v", p.state)
}

// Executions returns the number of actual analyzer calls made.
// This is used by tests to verify single-execution semantics.
func (p *DupcodeAnalysisProvider) Executions() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.executions
}

// deepCopyFindings creates a deep copy of findings to prevent mutation.
func deepCopyFindings(src []dupcode.Finding) []dupcode.Finding {
	if src == nil {
		return nil
	}
	result := make([]dupcode.Finding, len(src))
	for i := range src {
		f := src[i]
		result[i] = dupcode.Finding{
			Fingerprint:       f.Fingerprint,
			StableFingerprint: f.StableFingerprint,
			TokenCount:        f.TokenCount,
			LineCount:         f.LineCount,
			Occurrences:       deepCopyOccurrences(f.Occurrences),
		}
	}
	return result
}

// deepCopyOccurrences creates a deep copy of occurrences.
func deepCopyOccurrences(src []dupcode.Occurrence) []dupcode.Occurrence {
	if src == nil {
		return nil
	}
	result := make([]dupcode.Occurrence, len(src))
	copy(result, src)
	return result
}
