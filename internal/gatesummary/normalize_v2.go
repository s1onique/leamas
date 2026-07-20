package gatesummary

// projectV2 projects a v2 wire summary into the common normalized Summary.
// V2 includes all scope, parent, execution, and cleanliness fields.
// Lifecycle values are normalized from uppercase to lowercase.
func projectV2(wire V2Summary) (Summary, error) {
	s := Summary{
		SchemaVersion: Version2,
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
		// Evidence
		if wc.Evidence != "" {
			ev := wc.Evidence
			c.Evidence = &ev
		}
		// Detail
		if wc.Detail != "" {
			d := wc.Detail
			c.Detail = &d
		}
		// Duration
		dur, err := newIntegerFromWire(wc.Extras.DurationMs)
		if err != nil {
			return Summary{}, err
		}
		c.DurationMs = &dur
		// Execution
		exec := CheckExecution{
			StdoutSHA256: wc.Extras.StdoutSHA256,
			StderrSHA256: wc.Extras.StderrSHA256,
		}
		// Argv - deep copy
		if len(wc.Extras.Argv) > 0 {
			argv := make([]string, len(wc.Extras.Argv))
			copy(argv, wc.Extras.Argv)
			exec.Argv = argv
		}
		// Exit code
		if wc.Extras.ExitCode != nil {
			ec, err := newIntegerFromWire(*wc.Extras.ExitCode)
			if err != nil {
				return Summary{}, err
			}
			exec.ExitCode = &ec
		}
		c.Execution = &exec
		// Totals
		if wc.Total != nil {
			t := TestTotals{}
			tot, err := newIntegerFromWire(*wc.Total)
			if err != nil {
				return Summary{}, err
			}
			t.Total = tot
			if wc.PassCount != nil {
				pc, err := newIntegerFromWire(*wc.PassCount)
				if err != nil {
					return Summary{}, err
				}
				t.Pass = pc
			}
			if wc.FailCount != nil {
				fc, err := newIntegerFromWire(*wc.FailCount)
				if err != nil {
					return Summary{}, err
				}
				t.Fail = fc
			}
			if wc.SkipCount != nil {
				sc, err := newIntegerFromWire(*wc.SkipCount)
				if err != nil {
					return Summary{}, err
				}
				t.Skip = sc
			}
			if wc.UnavailableCount != nil {
				uc, err := newIntegerFromWire(*wc.UnavailableCount)
				if err != nil {
					return Summary{}, err
				}
				t.Unavailable = uc
			}
			c.Totals = &t
		}
		s.Checks[i] = c
	}

	return s, nil
}

// cloneV2Wire creates a deep copy of a v2 wire summary.
func cloneV2Wire(w V2Summary) V2Summary {
	clone := V2Summary{
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
		Checks:              make([]V2Check, len(w.Checks)),
	}
	for i, c := range w.Checks {
		clone.Checks[i] = V2Check{
			Name:     c.Name,
			Scope:    c.Scope,
			Status:   c.Status,
			Evidence: c.Evidence,
			Detail:   c.Detail,
			Extras: V2Extras{
				DurationMs:   c.Extras.DurationMs,
				StdoutSHA256: c.Extras.StdoutSHA256,
				StderrSHA256: c.Extras.StderrSHA256,
			},
		}
		if c.Extras.Argv != nil {
			argv := make([]string, len(c.Extras.Argv))
			copy(argv, c.Extras.Argv)
			clone.Checks[i].Extras.Argv = argv
		}
		if c.Extras.ExitCode != nil {
			ec := *c.Extras.ExitCode
			clone.Checks[i].Extras.ExitCode = &ec
		}
		if c.Total != nil {
			tot := *c.Total
			clone.Checks[i].Total = &tot
		}
		if c.PassCount != nil {
			pc := *c.PassCount
			clone.Checks[i].PassCount = &pc
		}
		if c.FailCount != nil {
			fc := *c.FailCount
			clone.Checks[i].FailCount = &fc
		}
		if c.SkipCount != nil {
			sc := *c.SkipCount
			clone.Checks[i].SkipCount = &sc
		}
		if c.UnavailableCount != nil {
			uc := *c.UnavailableCount
			clone.Checks[i].UnavailableCount = &uc
		}
	}
	return clone
}

// cloneCheck creates a deep copy of a Check.
func cloneCheck(c Check) Check {
	clone := Check{
		Name:   c.Name,
		Status: c.Status,
	}
	if c.Scope != nil {
		s := *c.Scope
		clone.Scope = &s
	}
	if c.Evidence != nil {
		e := *c.Evidence
		clone.Evidence = &e
	}
	if c.Detail != nil {
		d := *c.Detail
		clone.Detail = &d
	}
	if c.DurationMs != nil {
		dm := *c.DurationMs
		clone.DurationMs = &dm
	}
	if c.Execution != nil {
		e := CheckExecution{
			StdoutSHA256: c.Execution.StdoutSHA256,
			StderrSHA256: c.Execution.StderrSHA256,
		}
		if c.Execution.Argv != nil {
			argv := make([]string, len(c.Execution.Argv))
			copy(argv, c.Execution.Argv)
			e.Argv = argv
		}
		if c.Execution.ExitCode != nil {
			ec := *c.Execution.ExitCode
			e.ExitCode = &ec
		}
		clone.Execution = &e
	}
	if c.Totals != nil {
		t := *c.Totals
		clone.Totals = &t
	}
	return clone
}
