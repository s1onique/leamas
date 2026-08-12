package gatesummary

// projectV3 projects a v3 wire summary into the common normalized Summary.
//
// V3 is a strict superset of v2 at the top level: scope, parent,
// execution binding, cleanliness, and disposition are projected the
// same way as v2. The v3-only top-level evidence bindings
// (registry_sha256/events_sha256/transcript_sha256) are stored on the
// normalized Summary so downstream consumers can re-bind the document
// to its evidence. The v3 counts block and four parallel name lists are
// authoritative: projectV3 does not recompute them, but the v3 semantic
// validator re-derives them from Checks and rejects any mismatch.
//
// V3 per-check evidence identifiers (id/order/execution_class/argv/
// implementation) are surfaced on the normalized Check; the runner-
// classified canonical_exit_code is preferred for the normalized Check's
// execution exit code so consumers see the same canonical value the
// runner uses. raw_process_exit_code, deadline_exceeded, and the
// captured termination_signal are stored on CheckExecution as auxiliary
// process-execution evidence.
func projectV3(wire V3Summary) (Summary, error) {
	s := Summary{
		SchemaVersion: Version3,
		GeneratedAt:   wire.GeneratedAt,
		Overall: Overall{
			Status: wireToGateStatus(wire.OverallStatus),
		},
		Scope: &Scope{
			ID:          wire.ScopeID,
			Status:      normalizeLifecycle(wire.ScopeStatus),
			Disposition: wire.ScopeDisposition,
		},
		Parent: &Parent{
			Act:         wire.ParentAct,
			Status:      normalizeLifecycle(wire.ParentStatus),
			Disposition: wire.ParentDisposition,
			Root:        wire.ParentAct == "",
		},
		Execution: &ExecutionBinding{
			HeadOID:    wire.ExecutionHeadOID,
			TreeOID:    wire.ExecutionTreeOID,
			SubjectOID: wire.SubjectTreeOID,
		},
		Worktree: &WorktreeState{
			CleanBefore: wire.WorktreeCleanBefore,
			CleanAfter:  wire.WorktreeCleanAfter,
		},
		EvidenceHashes: &EvidenceHashes{
			RegistrySHA256:   wire.RegistrySHA256,
			EventsSHA256:     wire.EventsSHA256,
			TranscriptSHA256: wire.TranscriptSHA256,
		},
		Counts: Counts{
			Total:       wire.Counts.Total,
			Pass:        wire.Counts.Pass,
			Fail:        wire.Counts.Fail,
			Timeout:     wire.Counts.Timeout,
			Skip:        wire.Counts.Skip,
			Unavailable: wire.Counts.Unavailable,
		},
		FailedNames:      append([]string(nil), wire.FailedNames...),
		TimeoutNames:     append([]string(nil), wire.TimeoutNames...),
		SkippedNames:     append([]string(nil), wire.SkippedNames...),
		UnavailableNames: append([]string(nil), wire.UnavailableNames...),
		Checks: make([]Check, len(wire.Checks)),
	}

	// Overall disposition
	if wire.OverallDisposition != "" {
		disp := wire.OverallDisposition
		s.Overall.Disposition = &disp
	}

	// Project checks
	for i, wc := range wire.Checks {
		c := Check{
			Name:   wc.Name,
			Status: wireToGateStatus(wc.Status),
		}
		// Scope
		if wc.Scope != "" {
			scope := wc.Scope
			c.Scope = &scope
		}
		// Evidence (v3 uses per-check Detail as the wire-evidence
		// string in the digest surface; preserve verbatim).
		if wc.Detail != "" {
			ev := wc.Detail
			c.Evidence = &ev
		}
		// Duration
		dur, err := newIntegerFromWire(wc.DurationMs)
		if err != nil {
			return Summary{}, err
		}
		c.DurationMs = &dur
		// Execution evidence
		exec := CheckExecution{
			StdoutSHA256:    wc.StdoutSHA256,
			StderrSHA256:    wc.StderrSHA256,
			StdoutBytes:     wc.StdoutBytes,
			StderrBytes:     wc.StderrBytes,
			StdoutTruncated: wc.StdoutTruncated,
			StderrTruncated: wc.StderrTruncated,
		}
		// Argv - deep copy
		if len(wc.Argv) > 0 {
			argv := make([]string, len(wc.Argv))
			copy(argv, wc.Argv)
			exec.Argv = argv
		}
		// Canonical exit code (nullable)
		if wc.CanonicalExitCode != nil {
			ec, err := newIntegerFromWire(*wc.CanonicalExitCode)
			if err != nil {
				return Summary{}, err
			}
			exec.ExitCode = &ec
		}
		// Auxiliary process-execution evidence (v3-only)
		if wc.RawProcessExit != nil {
			rpe, err := newIntegerFromWire(*wc.RawProcessExit)
			if err != nil {
				return Summary{}, err
			}
			exec.RawExitCode = &rpe
		}
		if wc.TerminationSignal != nil {
			sig := *wc.TerminationSignal
			exec.TerminationSignal = &sig
		}
		exec.DeadlineExceeded = wc.DeadlineExceeded
		// Implementation argv sha256 binding
		exec.Implementation = wc.Implementation
		// V3 ID and order and execution_class
		if wc.ID != "" {
			id := wc.ID
			c.ID = &id
		}
		if order, err := newIntegerFromWire(wc.Order); err == nil {
			o := order
			c.Order = &o
		}
		if wc.ExecutionClass != "" {
			cls := wc.ExecutionClass
			c.ExecutionClass = &cls
		}
		c.Execution = &exec
		s.Checks[i] = c
	}

	return s, nil
}

// cloneV3Wire creates a deep copy of a v3 wire summary.
func cloneV3Wire(w V3Summary) V3Summary {
	clone := V3Summary{
		SchemaVersion:       w.SchemaVersion,
		GeneratedAt:         w.GeneratedAt,
		ScopeID:             w.ScopeID,
		ScopeStatus:         w.ScopeStatus,
		ScopeDisposition:    w.ScopeDisposition,
		ParentAct:           w.ParentAct,
		ParentStatus:        w.ParentStatus,
		ParentDisposition:   w.ParentDisposition,
		OverallStatus:       w.OverallStatus,
		OverallDisposition:  w.OverallDisposition,
		ExecutionHeadOID:    w.ExecutionHeadOID,
		ExecutionTreeOID:    w.ExecutionTreeOID,
		SubjectTreeOID:      w.SubjectTreeOID,
		WorktreeCleanBefore: w.WorktreeCleanBefore,
		WorktreeCleanAfter:  w.WorktreeCleanAfter,
		RegistrySHA256:      w.RegistrySHA256,
		EventsSHA256:        w.EventsSHA256,
		TranscriptSHA256:    w.TranscriptSHA256,
		Counts:              w.Counts,
		FailedNames:         append([]string(nil), w.FailedNames...),
		TimeoutNames:        append([]string(nil), w.TimeoutNames...),
		SkippedNames:        append([]string(nil), w.SkippedNames...),
		UnavailableNames:    append([]string(nil), w.UnavailableNames...),
		Checks:              make([]V3Check, len(w.Checks)),
	}
	for i, c := range w.Checks {
		clone.Checks[i] = V3Check{
			ID:               c.ID,
			Name:             c.Name,
			Scope:            c.Scope,
			Order:            c.Order,
			ExecutionClass:   c.ExecutionClass,
			Status:           c.Status,
			Detail:           c.Detail,
			DurationMs:       c.DurationMs,
			DeadlineExceeded: c.DeadlineExceeded,
			StdoutSHA256:     c.StdoutSHA256,
			StderrSHA256:     c.StderrSHA256,
			StdoutBytes:      c.StdoutBytes,
			StderrBytes:      c.StderrBytes,
			StdoutTruncated:  c.StdoutTruncated,
			StderrTruncated:  c.StderrTruncated,
			Implementation:   c.Implementation,
		}
		if len(c.Argv) > 0 {
			argv := make([]string, len(c.Argv))
			copy(argv, c.Argv)
			clone.Checks[i].Argv = argv
		}
		if c.CanonicalExitCode != nil {
			ec := *c.CanonicalExitCode
			clone.Checks[i].CanonicalExitCode = &ec
		}
		if c.RawProcessExit != nil {
			rpe := *c.RawProcessExit
			clone.Checks[i].RawProcessExit = &rpe
		}
		if c.TerminationSignal != nil {
			sig := *c.TerminationSignal
			clone.Checks[i].TerminationSignal = &sig
		}
	}
	return clone
}
