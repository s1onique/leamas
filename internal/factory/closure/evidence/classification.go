// SPDX-License-Identifier: Apache-2.0

// Package evidence - classification.go implements Phase 5 of
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.
//
// ACT-owned classification is the deterministic mapping from
// observed gate status + finding paths to one of PASS / FAIL /
// UNAVAILABLE. The function is the only authority that may flip
// a finding from FAIL to PASS (when the finding is an unchanged
// baseline finding that does not intersect any ACT-owned path).

package evidence

import (
	"strings"
)

// ACTOwnedClassification is the deterministic verdict derived
// from a GateCapture plus the ACT-owned path set.
type ACTOwnedClassification string

const (
	ACTOwnedPass        ACTOwnedClassification = "PASS"
	ACTOwnedFail        ACTOwnedClassification = "FAIL"
	ACTOwnedUnavailable ACTOwnedClassification = "UNAVAILABLE"
)

// ClassificationInputs parameterises ClassifyACTOwnedGate.
type ClassificationInputs struct {
	// ObservedStatus is the verbatim status parsed from the
	// fast-lane raw output (OK, FAILED, SKIP, UNKNOWN).
	ObservedStatus string
	// ObservedFindings is every finding extracted from the raw
	// output.
	ObservedFindings []GateFinding
	// BaselineFindings is the frozen baseline set; findings that
	// match (path, rule) are treated as unchanged.
	BaselineFindings []GateFinding
	// ACTOwnedPaths lists every path the current ACT owns.
	// Findings that intersect any ACTOwnedPath are never treated
	// as baseline.
	ACTOwnedPaths []string
	// LaneMissing reports whether the aggregate lane was absent
	// from the raw output.
	LaneMissing bool
	// LaneTimedOut reports whether the aggregate lane timed out.
	LaneTimedOut bool
	// LaneTruncated reports whether the aggregate lane's output
	// was truncated.
	LaneTruncated bool
}

// ClassifyACTOwnedGate applies the Phase 5 rule set. It returns
// one of PASS, FAIL, or UNAVAILABLE.
//
//   - UNAVAILABLE: lane missing, skipped, malformed, timed out,
//     or truncated.
//   - PASS: observed status is OK OR every observed finding is
//     proven unchanged (matches a baseline finding) AND no
//     unchanged finding intersects an ACT-owned path.
//   - FAIL: any new or changed finding intersects an ACT-owned
//     path.
func ClassifyACTOwnedGate(inputs ClassificationInputs) ACTOwnedClassification {
	if inputs.LaneMissing || inputs.LaneTimedOut || inputs.LaneTruncated {
		return ACTOwnedUnavailable
	}
	switch strings.ToUpper(strings.TrimSpace(inputs.ObservedStatus)) {
	case "OK":
		return ACTOwnedPass
	case "FAILED":
		if len(inputs.ObservedFindings) == 0 {
			return ACTOwnedUnavailable
		}
		// fall through to finding-level classification
	case "SKIP", "UNKNOWN", "":
		return ACTOwnedUnavailable
	}
	for _, finding := range inputs.ObservedFindings {
		if intersectsOwnedPath(finding.Path, inputs.ACTOwnedPaths) {
			return ACTOwnedFail
		}
		if !isBaselineFinding(finding, inputs.BaselineFindings) {
			return ACTOwnedFail
		}
	}
	return ACTOwnedPass
}

// intersectsOwnedPath reports whether path matches any owned path.
// Owned paths may use glob-style prefixes (e.g. "cmd/leamas/**");
// the implementation honours the literal and "*" prefix forms.
func intersectsOwnedPath(path string, owned []string) bool {
	for _, candidate := range owned {
		if candidate == "" {
			continue
		}
		if candidate == path {
			return true
		}
		if strings.HasSuffix(candidate, "/**") {
			prefix := strings.TrimSuffix(candidate, "/**")
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				return true
			}
		}
		if strings.HasSuffix(candidate, "/*") {
			prefix := strings.TrimSuffix(candidate, "/*")
			if strings.HasPrefix(path, prefix+"/") && !strings.Contains(strings.TrimPrefix(path, prefix+"/"), "/") {
				return true
			}
		}
	}
	return false
}

// isBaselineFinding reports whether finding matches (path, rule)
// in the supplied baseline set. Severity and message are
// deliberately ignored so the comparison stays deterministic.
func isBaselineFinding(finding GateFinding, baseline []GateFinding) bool {
	for _, candidate := range baseline {
		if candidate.Path == finding.Path && candidate.Rule == finding.Rule {
			return true
		}
	}
	return false
}
