// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"fmt"
	"strings"
)

// highestPriority returns the candidate with the highest strategy
// priority. Tie-breaks by ActID then Range for determinism.
func highestPriority(candidates []candidateResolution) candidateResolution {
	best := candidates[0]
	for _, c := range candidates[1:] {
		if strategyPriority[c.strategy] > strategyPriority[best.strategy] {
			best = c
			continue
		}
		if strategyPriority[c.strategy] == strategyPriority[best.strategy] {
			if c.resolution.ActID < best.resolution.ActID {
				best = c
				continue
			}
			if c.resolution.ActID == best.resolution.ActID &&
				c.resolution.Range() < best.resolution.Range() {
				best = c
			}
		}
	}
	return best
}

// ambiguousRangeError builds the documented fail-closed diagnostic.
func ambiguousRangeError(candidates []candidateResolution, headOID string) error {
	var sb strings.Builder
	sb.WriteString("\ncandidates:")
	seen := map[string]bool{}
	for _, c := range candidates {
		key := c.resolution.ActID + "@" + c.strategy
		if seen[key] {
			continue
		}
		seen[key] = true
		sb.WriteString("\n  - act=")
		sb.WriteString(c.resolution.ActID)
		sb.WriteString(" strategy=")
		sb.WriteString(c.strategy)
		sb.WriteString(" range=")
		sb.WriteString(c.resolution.Range())
	}
	sb.WriteString("\n  head=")
	sb.WriteString(headOID)
	sb.WriteString("\nrerun with --range only after resolving lifecycle metadata")
	return fmt.Errorf("%w: %s", ErrAmbiguousRange, sb.String())
}
