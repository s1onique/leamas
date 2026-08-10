// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_contract_parity_r7_fixtures_part2_test.go
// is the second half of the B2-R7 fixture data.
package closure

import (
	"strings"
	"testing"
)

func r7ParityRowsPart2(t *testing.T, rows []r7Fixture) []r7Fixture {
	{
		b := newR7Builder()
		ex := makeExcludeCheck("ex1", "obsolete")
		ex.Environment = map[string]string{"K": "V"}
		b.plan.Checks = []PlanCheck{ex}
		rows = append(rows, r7Fixture{
			name: "exclude with environment", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// zero artifacts (acceptable: empty array is valid).
	{
		b := newR7Builder()
		b.plan.Artifacts = []PlanArtifact{}
		rows = append(rows, r7Fixture{
			name: "zero artifacts", bytes: b.bytes(t), plan: b.plan, wantErr: false,
		})
	}

	// MaxArtifacts - accepted (uses minimal artifact shape).
	{
		b := newR7Builder()
		b.plan.Artifacts = make([]PlanArtifact, 4096)
		for i := range b.plan.Artifacts {
			b.plan.Artifacts[i] = r7MinArtifact("a" + intToStr(i))
		}
		// Drop checks array; matrix already covers checks.
		b.plan.Checks = []PlanCheck{makeRunCheck("c1")}
		rows = append(rows, r7Fixture{
			name: "MaxArtifacts", bytes: b.bytes(t), plan: b.plan, wantErr: false,
		})
	}

	// MaxArtifacts+1 - rejected.
	{
		b := newR7Builder()
		b.plan.Artifacts = make([]PlanArtifact, 4097)
		for i := range b.plan.Artifacts {
			b.plan.Artifacts[i] = r7MinArtifact("a" + intToStr(i))
		}
		b.plan.Checks = []PlanCheck{makeRunCheck("c1")}
		rows = append(rows, r7Fixture{
			name: "MaxArtifacts+1", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// duplicate artifact ID.
	{
		b := newR7Builder()
		b.plan.Artifacts = []PlanArtifact{
			makeArtifact("dup"),
			makeArtifact("dup"),
		}
		rows = append(rows, r7Fixture{
			name: "duplicate artifact ID", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// artifact required missing (not present in struct).
	// json.Marshal emits a JSON null for a nil *bool
	// pointer, which the leaf treats as "absent" and
	// rejects. This is the typed surface.
	{
		b := newR7Builder()
		art := makeArtifact("a1")
		art.Required = nil
		b.plan.Artifacts = []PlanArtifact{art}
		rows = append(rows, r7Fixture{
			name: "artifact required missing", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// artifact required non-boolean (string). Build via
	// raw JSON to bypass the typed *bool surface; this
	// proves the leaf rejects malformed JSON wire values.
	{
		base := newR7Builder()
		baseBytes := base.bytes(t)
		modified := strings.Replace(string(baseBytes),
			`"required":true`, `"required":"yes"`, 1)
		rows = append(rows, r7Fixture{
			name: "artifact required non-boolean", bytes: []byte(modified), plan: base.plan, wantErr: true,
		})
	}

	// artifact media_type whitespace-only (rejected).
	{
		base := newR7Builder()
		baseBytes := base.bytes(t)
		modified := strings.Replace(string(baseBytes),
			`"media_type":"text/plain"`, `"media_type":"   "`, 1)
		rows = append(rows, r7Fixture{
			name: "artifact media_type whitespace-only", bytes: []byte(modified), plan: base.plan, wantErr: true,
		})
	}

	// policy absent (rejected).
	{
		b := newR7Builder()
		b.plan.Policy = PlanPolicy{}
		rows = append(rows, r7Fixture{
			name: "policy absent", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// policy null - hand-crafted wire bytes with
	// "policy":null so the matrix exercises the explicit
	// null case independently of the typed builder. The
	// typed builder cannot emit "policy":null because
	// json.Marshal omits the empty object.
	rows = append(rows, r7Fixture{
		name:    "policy null",
		bytes:   []byte(`{"contract_version":1,"act_id":"ACT-R7-PARITY-01","baseline":{"commit_oid":"a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1","tree_oid":"b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2"},"execution":{"mode":"serial_fail_fast"},"checks":[{"id":"c1","mode":"run","argv":["go","test"],"working_directory":".","timeout_seconds":60,"environment":{"K":"V"}}],"artifacts":[{"id":"a1","path":"docs/a1.md","required":true,"max_bytes":1024,"media_type":"text/plain"}],"policy":null}`),
		plan:    Plan{},
		wantErr: true,
	})

	// each policy sibling missing - exercised by setting
	// a single sibling to nil and the rest to true.
	for _, name := range []string{
		"each policy sibling missing: require_clean_before",
		"each policy sibling missing: require_clean_after",
		"each policy sibling missing: forbid_tracked_full_digests",
		"each policy sibling missing: require_diff_check",
	} {
		b := newR7Builder()
		t1 := true
		switch name {
		case "each policy sibling missing: require_clean_before":
			b.plan.Policy = PlanPolicy{
				RequireCleanAfter:        &t1,
				ForbidTrackedFullDigests: &t1,
				RequireDiffCheck:         &t1,
			}
		case "each policy sibling missing: require_clean_after":
			b.plan.Policy = PlanPolicy{
				RequireCleanBefore:       &t1,
				ForbidTrackedFullDigests: &t1,
				RequireDiffCheck:         &t1,
			}
		case "each policy sibling missing: forbid_tracked_full_digests":
			b.plan.Policy = PlanPolicy{
				RequireCleanBefore: &t1,
				RequireCleanAfter:  &t1,
				RequireDiffCheck:   &t1,
			}
		case "each policy sibling missing: require_diff_check":
			b.plan.Policy = PlanPolicy{
				RequireCleanBefore:       &t1,
				RequireCleanAfter:        &t1,
				ForbidTrackedFullDigests: &t1,
			}
		}
		rows = append(rows, r7Fixture{
			name: name, bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// each policy sibling=false (rejected).
	for _, name := range []string{
		"each policy sibling=false: require_clean_before",
		"each policy sibling=false: require_clean_after",
		"each policy sibling=false: forbid_tracked_full_digests",
		"each policy sibling=false: require_diff_check",
	} {
		b := newR7Builder()
		t1 := true
		f := false
		switch name {
		case "each policy sibling=false: require_clean_before":
			b.plan.Policy = PlanPolicy{
				RequireCleanBefore:       &f,
				RequireCleanAfter:        &t1,
				ForbidTrackedFullDigests: &t1,
				RequireDiffCheck:         &t1,
			}
		case "each policy sibling=false: require_clean_after":
			b.plan.Policy = PlanPolicy{
				RequireCleanBefore:       &t1,
				RequireCleanAfter:        &f,
				ForbidTrackedFullDigests: &t1,
				RequireDiffCheck:         &t1,
			}
		case "each policy sibling=false: forbid_tracked_full_digests":
			b.plan.Policy = PlanPolicy{
				RequireCleanBefore:       &t1,
				RequireCleanAfter:        &t1,
				ForbidTrackedFullDigests: &f,
				RequireDiffCheck:         &t1,
			}
		case "each policy sibling=false: require_diff_check":
			b.plan.Policy = PlanPolicy{
				RequireCleanBefore:       &t1,
				RequireCleanAfter:        &t1,
				ForbidTrackedFullDigests: &t1,
				RequireDiffCheck:         &f,
			}
		}
		rows = append(rows, r7Fixture{
			name: name, bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// policy invalid type - one sibling set to a string.
	{
		base := newR7Builder()
		baseBytes := base.bytes(t)
		modified := strings.Replace(string(baseBytes),
			`"require_clean_before":true`, `"require_clean_before":"yes"`, 1)
		rows = append(rows, r7Fixture{
			name: "policy invalid type", bytes: []byte(modified), plan: base.plan, wantErr: true,
		})
	}

	// policy unknown sibling.
	{
		base := newR7Builder()
		baseBytes := base.bytes(t)
		modified := strings.Replace(string(baseBytes),
			`"require_clean_before":true`,
			`"require_clean_before":true,"policy_extra":true`, 1)
		rows = append(rows, r7Fixture{
			name: "policy unknown sibling", bytes: []byte(modified), plan: base.plan, wantErr: true,
		})
	}

	// runner authority absent (treated as no runner_authority).
	{
		b := newR7Builder()
		b.plan.RunnerAuthority = nil
		rows = append(rows, r7Fixture{
			name: "runner authority absent", bytes: b.bytes(t), plan: b.plan, wantErr: false,
		})
	}

	// subject_exact valid.
	{
		b := newR7Builder()
		b.plan.RunnerAuthority = &RunnerAuthority{
			Mode: RunnerAuthoritySubjectExact,
			Tool: nil,
		}
		rows = append(rows, r7Fixture{
			name: "subject_exact valid", bytes: b.bytes(t), plan: b.plan, wantErr: false,
		})
	}

	// subject_exact with tool (rejected).
	{
		b := newR7Builder()
		b.plan.RunnerAuthority = &RunnerAuthority{
			Mode: RunnerAuthoritySubjectExact,
			Tool: &ToolAuthority{
				Revision:     "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
				BinarySHA256: "c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3",
			},
		}
		rows = append(rows, r7Fixture{
			name: "subject_exact with tool", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// tool_release_exact valid.
	{
		b := newR7Builder()
		b.plan.RunnerAuthority = &RunnerAuthority{
			Mode: RunnerAuthorityToolReleaseExact,
			Tool: &ToolAuthority{
				Revision:     "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
				BinarySHA256: "c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3",
			},
		}
		rows = append(rows, r7Fixture{
			name: "tool_release_exact valid", bytes: b.bytes(t), plan: b.plan, wantErr: false,
		})
	}

	// tool_release_exact missing tool (rejected).
	{
		b := newR7Builder()
		b.plan.RunnerAuthority = &RunnerAuthority{
			Mode: RunnerAuthorityToolReleaseExact,
		}
		rows = append(rows, r7Fixture{
			name: "tool_release_exact missing tool", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// invalid revision.
	{
		b := newR7Builder()
		b.plan.RunnerAuthority = &RunnerAuthority{
			Mode: RunnerAuthorityToolReleaseExact,
			Tool: &ToolAuthority{
				Revision:     "not-an-oid",
				BinarySHA256: "c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3",
			},
		}
		rows = append(rows, r7Fixture{
			name: "invalid revision", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// binary_sha256 wrong length.
	{
		b := newR7Builder()
		b.plan.RunnerAuthority = &RunnerAuthority{
			Mode: RunnerAuthorityToolReleaseExact,
			Tool: &ToolAuthority{
				Revision:     "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
				BinarySHA256: "c3c3",
			},
		}
		rows = append(rows, r7Fixture{
			name: "binary_sha256 wrong length", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// binary_sha256 uppercase (rejected: leaf rejects uppercase).
	{
		b := newR7Builder()
		b.plan.RunnerAuthority = &RunnerAuthority{
			Mode: RunnerAuthorityToolReleaseExact,
			Tool: &ToolAuthority{
				Revision:     "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
				BinarySHA256: "C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3C3",
			},
		}
		rows = append(rows, r7Fixture{
			name: "binary_sha256 uppercase", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// binary_sha256 non-hex.
	{
		b := newR7Builder()
		b.plan.RunnerAuthority = &RunnerAuthority{
			Mode: RunnerAuthorityToolReleaseExact,
			Tool: &ToolAuthority{
				Revision:     "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
				BinarySHA256: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			},
		}
		rows = append(rows, r7Fixture{
			name: "binary_sha256 non-hex", bytes: b.bytes(t), plan: b.plan, wantErr: true,
		})
	}

	// version number (raw JSON). Use raw-bytes fixture to
	// exercise the wire-shape path because the typed
	// PlanCheck.Version field is a string. We hand-craft
	// the tool block with version:123 to bypass the
	// omitempty default.
	rows = append(rows, r7Fixture{
		name:    "tool version number",
		bytes:   []byte(`{"contract_version":1,"act_id":"ACT-R7-PARITY-01","baseline":{"commit_oid":"a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1","tree_oid":"b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2"},"execution":{"mode":"serial_fail_fast"},"checks":[{"id":"c1","mode":"run","argv":["go","test"],"working_directory":".","timeout_seconds":60,"environment":{"K":"V"}}],"artifacts":[{"id":"a1","path":"docs/a1.md","required":true,"max_bytes":1024,"media_type":"text/plain"}],"policy":{"require_clean_before":true,"require_clean_after":true,"forbid_tracked_full_digests":true,"require_diff_check":true},"runner_authority":{"mode":"tool_release_exact","tool":{"revision":"a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1","binary_sha256":"c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3","version":123}}}`),
		plan:    Plan{},
		wantErr: true,
	})

	// tag_name number (raw JSON).
	rows = append(rows, r7Fixture{
		name:    "tool tag_name number",
		bytes:   []byte(`{"contract_version":1,"act_id":"ACT-R7-PARITY-01","baseline":{"commit_oid":"a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1","tree_oid":"b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2"},"execution":{"mode":"serial_fail_fast"},"checks":[{"id":"c1","mode":"run","argv":["go","test"],"working_directory":".","timeout_seconds":60,"environment":{"K":"V"}}],"artifacts":[{"id":"a1","path":"docs/a1.md","required":true,"max_bytes":1024,"media_type":"text/plain"}],"policy":{"require_clean_before":true,"require_clean_after":true,"forbid_tracked_full_digests":true,"require_diff_check":true},"runner_authority":{"mode":"tool_release_exact","tool":{"revision":"a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1","binary_sha256":"c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3","tag_name":456}}}`),
		plan:    Plan{},
		wantErr: true,
	})

	// version + tag_name as strings (accepted).
	{
		b := newR7Builder()
		b.plan.RunnerAuthority = &RunnerAuthority{
			Mode: RunnerAuthorityToolReleaseExact,
			Tool: &ToolAuthority{
				Revision:     "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
				BinarySHA256: "c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3",
				Version:      "v1.2.3",
				TagName:      "leamas-v1.2.3",
			},
		}
		rows = append(rows, r7Fixture{
			name: "tool version+tag_name strings", bytes: b.bytes(t), plan: b.plan, wantErr: false,
		})
	}

	return rows
}
