// SPDX-License-Identifier: Apache-2.0

// Package evidence - parse.go implements the typed parsing used
// by GateCapture. The parsers convert the textual fast-lane
// output into Go structs; they never spawn a shell process.

package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// sha256Hex is the canonical helper used by GateCapture.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// laneLinePattern matches "lane <id>:<status>" lines.
var laneLinePattern = regexp.MustCompile(`(?m)^\s*lane\s+([a-zA-Z0-9_-]+)\s*:\s*([A-Z]+)\s*(?:[:#]\s*(.*))?$`)

// findingPattern matches "<path>:<line>:<col>:<severity>:<rule>:<message>" lines.
var findingPattern = regexp.MustCompile(`(?m)^([^\s:]+):(?:(\d+):)?(?:(\d+):)?([a-z]+):([a-z0-9_-]+):(.+)$`)

// observedStatusPattern matches the single observed status line
// emitted by the fast lane ("EXEC_GATE_OBSERVED_STATUS: OK").
var observedStatusPattern = regexp.MustCompile(`(?m)^\s*EXEC_GATE_OBSERVED_STATUS\s*:\s*([A-Z]+)\s*$`)

// parseLaneStatus extracts every lane result from the supplied
// raw output. Lines that do not match the lane grammar are
// silently ignored.
func parseLaneStatus(raw []byte) []GateLaneResult {
	matches := laneLinePattern.FindAllSubmatch(raw, -1)
	if len(matches) == 0 {
		return nil
	}
	results := make([]GateLaneResult, 0, len(matches))
	for _, match := range matches {
		result := GateLaneResult{
			LaneID: string(match[1]),
			Status: string(match[2]),
		}
		if len(match) > 3 {
			result.Message = strings.TrimSpace(string(match[3]))
		}
		results = append(results, result)
	}
	return results
}

// parseObservedStatus returns the first observed exec-gate status
// declared in the raw output. When no status line is present the
// function returns "UNKNOWN".
func parseObservedStatus(raw []byte) string {
	match := observedStatusPattern.FindSubmatch(raw)
	if match == nil {
		return "UNKNOWN"
	}
	return string(match[1])
}

// parseFindings extracts findings from the raw output. Lines that
// do not match the finding grammar are silently ignored.
func parseFindings(raw []byte) []GateFinding {
	matches := findingPattern.FindAllSubmatch(raw, -1)
	if len(matches) == 0 {
		return nil
	}
	findings := make([]GateFinding, 0, len(matches))
	for _, match := range matches {
		findings = append(findings, GateFinding{
			Path:     string(match[1]),
			Severity: string(match[4]),
			Rule:     string(match[5]),
			Message:  strings.TrimSpace(string(match[6])),
		})
	}
	return findings
}
