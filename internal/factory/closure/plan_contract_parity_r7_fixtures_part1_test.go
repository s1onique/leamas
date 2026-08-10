// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_contract_parity_r7_fixtures_part1_test.go
// is the first half of the B2-R7 fixture data. The
// matrix assembles the rows via the helpers in this
// file and the matching part2 file.
package closure

import (
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

func r7ParityRows(t *testing.T) []r7Fixture {
	t.Helper()
	rows := []r7Fixture{}

	// valid base.
	{
		b := newR7Builder()
		rows = append(rows, r7Fixture{
			name: "valid base", bytes: b.bytes(t), plan: b.plan, wantErr: false,
		})
	}

	// contract_version=2.
	{
		b := newR7Builder()
		b.plan.ContractVersion = 2
		rows = append(rows, r7Fixture{
			name: "contract_version=2", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// invalid act_id.
	{
		b := newR7Builder()
		b.plan.ActID = "not-an-act-id"
		rows = append(rows, r7Fixture{
			name: "invalid act_id", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// baseline placeholder.
	{
		b := newR7Builder()
		b.plan.Baseline.CommitOID = "TBD"
		rows = append(rows, r7Fixture{
			name: "baseline placeholder", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// malformed baseline OID.
	{
		b := newR7Builder()
		b.plan.Baseline.TreeOID = "not-a-hex-oid"
		rows = append(rows, r7Fixture{
			name: "malformed baseline OID", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// zero checks.
	{
		b := newR7Builder()
		b.plan.Checks = nil
		rows = append(rows, r7Fixture{
			name: "zero checks", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// MaxChecks - accepted (uses minimal check shape so
	// the marshalled JSON stays under MaxPlanBytes).
	{
		b := newR7Builder()
		b.plan.Checks = make([]PlanCheck, 4096)
		for i := range b.plan.Checks {
			b.plan.Checks[i] = r7MinRunCheck("c" + intToStr(i))
		}
		// Drop the artifact array; the matrix already
		// covers artifacts elsewhere.
		b.plan.Artifacts = []PlanArtifact{}
		rows = append(rows, r7Fixture{
			name: "MaxChecks", bytes: b.bytes(t), plan: b.plan, wantErr: false,
		})
	}

	// MaxChecks+1 - rejected.
	{
		b := newR7Builder()
		b.plan.Checks = make([]PlanCheck, 4097)
		for i := range b.plan.Checks {
			b.plan.Checks[i] = r7MinRunCheck("c" + intToStr(i))
		}
		b.plan.Artifacts = []PlanArtifact{}
		rows = append(rows, r7Fixture{
			name: "MaxChecks+1", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// duplicate check ID.
	{
		b := newR7Builder()
		b.plan.Checks = []PlanCheck{makeRunCheck("dup"), makeRunCheck("dup")}
		rows = append(rows, r7Fixture{
			name: "duplicate check ID", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// invalid check mode (unknown literal).
	{
		b := newR7Builder()
		b.plan.Checks = []PlanCheck{
			{ID: "c1", Mode: "garbage", Argv: []string{"go"}, WorkingDirectory: ".", TimeoutSeconds: 60, Environment: map[string]string{}},
		}
		rows = append(rows, r7Fixture{
			name: "invalid mode", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// run missing argv.
	{
		b := newR7Builder()
		b.plan.Checks = []PlanCheck{
			{ID: "c1", Mode: CheckModeRun, WorkingDirectory: ".", TimeoutSeconds: 60, Environment: map[string]string{}},
		}
		rows = append(rows, r7Fixture{
			name: "run missing argv", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// run missing working_directory.
	{
		b := newR7Builder()
		b.plan.Checks = []PlanCheck{
			{ID: "c1", Mode: CheckModeRun, Argv: []string{"go"}, TimeoutSeconds: 60, Environment: map[string]string{}},
		}
		rows = append(rows, r7Fixture{
			name: "run missing working_directory", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// run invalid working_directory.
	{
		b := newR7Builder()
		c := makeRunCheck("c1")
		c.WorkingDirectory = "../escape"
		b.plan.Checks = []PlanCheck{c}
		rows = append(rows, r7Fixture{
			name: "invalid working_directory", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// timeout=0.
	{
		b := newR7Builder()
		c := makeRunCheck("c1")
		c.TimeoutSeconds = 0
		b.plan.Checks = []PlanCheck{c}
		rows = append(rows, r7Fixture{
			name: "timeout=0", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// timeout=max.
	{
		b := newR7Builder()
		c := makeRunCheck("c1")
		c.TimeoutSeconds = plancontract.MaxCheckTimeoutSeconds
		b.plan.Checks = []PlanCheck{c}
		rows = append(rows, r7Fixture{
			name: "timeout=max", bytes: b.bytes(t), plan: b.plan, wantErr: false,
		})
	}

	// timeout=max+1.
	{
		b := newR7Builder()
		c := makeRunCheck("c1")
		c.TimeoutSeconds = plancontract.MaxCheckTimeoutSeconds + 1
		b.plan.Checks = []PlanCheck{c}
		rows = append(rows, r7Fixture{
			name: "timeout=max+1", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// invalid env name.
	{
		b := newR7Builder()
		c := makeRunCheck("c1")
		c.Environment = map[string]string{"1bad": "V"}
		b.plan.Checks = []PlanCheck{c}
		rows = append(rows, r7Fixture{
			name: "invalid env name", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// exclude valid.
	{
		b := newR7Builder()
		b.plan.Checks = []PlanCheck{makeExcludeCheck("ex1", "obsolete")}
		rows = append(rows, r7Fixture{
			name: "exclude valid", bytes: b.bytes(t), plan: b.plan, wantErr: false,
		})
	}

	// exclude missing reason.
	{
		b := newR7Builder()
		b.plan.Checks = []PlanCheck{
			{ID: "ex1", Mode: CheckModeExclude},
		}
		rows = append(rows, r7Fixture{
			name: "exclude missing reason", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// exclude with argv.
	{
		b := newR7Builder()
		ex := makeExcludeCheck("ex1", "obsolete")
		ex.Argv = []string{"go", "test"}
		b.plan.Checks = []PlanCheck{ex}
		rows = append(rows, r7Fixture{
			name: "exclude with argv", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// exclude with working_directory.
	{
		b := newR7Builder()
		ex := makeExcludeCheck("ex1", "obsolete")
		ex.WorkingDirectory = "."
		b.plan.Checks = []PlanCheck{ex}
		rows = append(rows, r7Fixture{
			name: "exclude with working_directory", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// exclude with timeout.
	{
		b := newR7Builder()
		ex := makeExcludeCheck("ex1", "obsolete")
		ex.TimeoutSeconds = 60
		b.plan.Checks = []PlanCheck{ex}
		rows = append(rows, r7Fixture{
			name: "exclude with timeout", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	return r7ParityRowsPart2(t, rows)
}
