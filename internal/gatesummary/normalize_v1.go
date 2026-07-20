package gatesummary

import "strings"

// projectV1 projects a v1 wire summary into the common normalized Summary.
// V1 has no scope, parent, execution binding, cleanliness, disposition, or totals.
// The producer's overall_status is preserved as authoritative.
func projectV1(wire V1Summary) (Summary, error) {
	s := Summary{
		SchemaVersion: Version1,
		GeneratedAt:   wire.GeneratedAt,
		Overall: Overall{
			Status: wireToGateStatus(wire.OverallStatus),
		},
		Checks: make([]Check, len(wire.Checks)),
	}

	// Preserve tool if present
	if wire.Tool != nil {
		tool := *wire.Tool
		s.Tool = &tool
	}

	// Project checks - v1 has minimal check structure
	for i, wc := range wire.Checks {
		c := Check{
			Name:   wc.Name,
			Status: wireToGateStatus(wc.Status),
		}
		// Duration
		if wc.DurationMs != nil {
			dur, err := newIntegerFromWire(*wc.DurationMs)
			if err != nil {
				return Summary{}, err
			}
			c.DurationMs = &dur
		}
		// Evidence
		if wc.Evidence != nil {
			ev := *wc.Evidence
			c.Evidence = &ev
		}
		s.Checks[i] = c
	}

	return s, nil
}

// cloneV1Wire creates a deep copy of a v1 wire summary.
func cloneV1Wire(w V1Summary) V1Summary {
	clone := V1Summary{
		SchemaVersion: w.SchemaVersion,
		GeneratedAt:   w.GeneratedAt,
		OverallStatus: w.OverallStatus,
		Checks:        make([]V1Check, len(w.Checks)),
	}
	if w.Tool != nil {
		t := *w.Tool
		clone.Tool = &t
	}
	for i, c := range w.Checks {
		clone.Checks[i] = V1Check{
			Name:   c.Name,
			Status: c.Status,
		}
		if c.DurationMs != nil {
			dur := *c.DurationMs
			clone.Checks[i].DurationMs = &dur
		}
		if c.Evidence != nil {
			e := *c.Evidence
			clone.Checks[i].Evidence = &e
		}
	}
	return clone
}

// stringsClone creates a copy of a string.
func stringsClone(s string) string {
	return strings.Clone(s)
}
