// SPDX-License-Identifier: Apache-2.0

// Package authority provides the executable-authority model that
// guarantees leamas invocations inside the Leamas source repository
// execute a binary bound to the current repository state.
//
// Capabilities are monotonic integers embedded into the binary at
// build time. The repository declares the minimum capability value
// each named contract requires; older binaries lacking that level
// fail closed even when they are technically an ancestor of HEAD.
package authority

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Capability names surfaced through the executable contract. Each
// named capability has a monotonic integer embedded into the binary;
// the repository declares the minimum required level.
const (
	CapDigestAutoRange     = "factory_digest_auto_range"
	CapSelfHostedAuthority = "factory_self_hosted_authority"
	CapClosureProtocol     = "closure_protocol"
)

// Embedded returns the capabilities this binary was built with. The
// returned map is owned by the caller.
func Embedded() map[string]int {
	out := make(map[string]int, len(capabilities))
	for k, v := range capabilities {
		out[k] = v
	}
	return out
}

// SetEmbedded is exported so tests can override the embedded
// capabilities without rebuilding the binary. Production code should
// rely on the default values declared below.
func SetEmbedded(name string, level int) {
	capabilities[name] = level
}

// capabilities is the build-time capability table. Values are
// injected via ldflags at build:
//
//	-X 'github.com/s1onique/leamas/internal/factory/authority.<name>=<level>'
//
// For local builds the defaults below are sufficient; production
// release builds are expected to inject real values via the Makefile.
var capabilities = map[string]int{
	CapDigestAutoRange:     1,
	CapSelfHostedAuthority: 1,
	CapClosureProtocol:     1,
}

// RequiredCapabilities is the parsed contents of the repository
// metadata file `.factory/required-capabilities.json`.
type RequiredCapabilities struct {
	Raw map[string]int
}

// LoadRequired parses the JSON manifest at path. Missing files
// return an empty RequiredCapabilities with nil Raw.
func LoadRequired(path string) (*RequiredCapabilities, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RequiredCapabilities{Raw: map[string]int{}}, nil
		}
		return nil, fmt.Errorf("read required capabilities: %w", err)
	}
	return parseRequired(data)
}

func parseRequired(data []byte) (*RequiredCapabilities, error) {
	var raw map[string]int
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode required capabilities: %w", err)
	}
	if raw == nil {
		raw = map[string]int{}
	}
	return &RequiredCapabilities{Raw: raw}, nil
}

// HasCapability returns true when the embedded level for name is at
// least required. Unknown names are treated as missing on both sides
// and therefore compatible.
func (e *EmbeddedCapabilities) HasCapability(name string, required int) bool {
	if e == nil {
		return required <= 0
	}
	got, ok := e.levels[name]
	if !ok {
		return required <= 0
	}
	return got >= required
}

// EmbeddedCapabilities is the immutable snapshot of the running
// binary's capability levels.
type EmbeddedCapabilities struct {
	levels map[string]int
}

// SnapshotEmbedded returns a sorted, copy-on-read snapshot of the
// current embedded capability table.
func SnapshotEmbedded() *EmbeddedCapabilities {
	return &EmbeddedCapabilities{levels: Embedded()}
}

// Names returns the capability names in deterministic order.
func (e *EmbeddedCapabilities) Names() []string {
	names := make([]string, 0, len(e.levels))
	for k := range e.levels {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Level returns the embedded level for name, or 0 when unknown.
func (e *EmbeddedCapabilities) Level(name string) int {
	if e == nil {
		return 0
	}
	return e.levels[name]
}

// Required is the parsed contents of `.factory/required-capabilities.json`.
type Required = RequiredCapabilities

// SatisfiedBy returns nil when every required capability is met by
// the embedded snapshot. The returned error is a *CapabilityGap
// listing the unsatisfied capabilities when not.
func (r *Required) SatisfiedBy(e *EmbeddedCapabilities) error {
	if r == nil {
		return nil
	}
	var gaps []string
	for name, required := range r.Raw {
		if !e.HasCapability(name, required) {
			gaps = append(gaps, fmt.Sprintf("%s: required>=%d, embedded=%d",
				name, required, e.Level(name)))
		}
	}
	if len(gaps) == 0 {
		return nil
	}
	sort.Strings(gaps)
	return &CapabilityGap{Missing: gaps}
}

// CapabilityGap is returned by (*Required).SatisfiedBy when one or
// more required capabilities are not satisfied by the running binary.
type CapabilityGap struct {
	Missing []string
}

// Error implements the error interface.
func (g *CapabilityGap) Error() string {
	if len(g.Missing) == 0 {
		return "capability gap"
	}
	return "missing capabilities: " + strings.Join(g.Missing, ", ")
}

// FormatLine returns a single-line summary suitable for diagnostics
// output.
func FormatLine(name string, level int) string {
	return fmt.Sprintf("%s=%d", name, level)
}

// DefaultPath returns the canonical location of the required
// capabilities manifest relative to a repository root.
func DefaultPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".factory", "required-capabilities.json")
}
