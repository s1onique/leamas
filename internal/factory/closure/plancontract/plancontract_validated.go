// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_validated.go owns the
// canonical ValidatedPlan projection that the closure runner
// and the evidence package both consume.
//
// B2-R7 motivation: prior to B2-R7, the leaf exposed a
// minimal DecodeResult (contract_version + ordered checks)
// which was sufficient for the evidence package but not for
// the closure runner, which needed a typed projection of
// every wire-contract field. The runner therefore performed
// a parallel typed decode + semantic pass and duplicated
// the wire rules in the closure package's validatePlanTyped
// helper, with a "parity assertion" against the leaf to
// catch drift. Two paths could still disagree in subtle ways
// because each used its own rules.
//
// B2-R7 makes the leaf the single semantic authority by
// introducing ValidatedPlan: the canonical projection of
// every wire-contract field the runner and the evidence
// package need. One call to DecodeAndValidateFull performs
// the full pipeline (strict syntax, duplicate-key rejection,
// unknown-field rejection, single-document EOF, full
// semantic validation, canonical projection) and either
// returns the validated plan or a typed DecodeError.
//
// The closure runner's ValidatePlan function is now an
// adapter: it serialises the typed Plan to JSON bytes,
// invokes DecodeAndValidateFull, and adapts the leaf's
// DecodeError to the closure package's legacy typed-error
// contract. The adapter MUST NOT re-implement any semantic
// rule; doing so would re-introduce the second authority
// B2-R7 was opened to delete.
package plancontract

import "encoding/json"

// ValidatedPlan is the canonical projection of a Plan
// Contract v1 document after the single-pass decoder/validator
// pipeline has accepted it. Every field on this struct is a
// pure projection of a wire-contract value; no field is
// derived from another wire field. Callers may treat the
// struct as immutable.
//
// The shape mirrors the Plan Contract v1 wire document:
//
//	{
//	  "contract_version":   int,
//	  "act_id":             string,
//	  "baseline":           Baseline,
//	  "execution":          Execution,
//	  "checks":             []Check,
//	  "artifacts":          []Artifact,
//	  "policy":             Policy,
//	  "runner_authority":   RunnerAuthority | null
//	}
type ValidatedPlan struct {
	ContractVersion int
	ActID           string
	Baseline        Baseline
	Execution       Execution
	Checks          []Check
	Artifacts       []Artifact
	Policy          Policy
	RunnerAuthority *RunnerAuthority
}

// Baseline is the canonical projection of the /baseline
// subtree. Both OIDs are required, lowercase 40- or
// 64-character hex, and free of closure placeholders.
type Baseline struct {
	CommitOID string
	TreeOID   string
}

// Execution is the canonical projection of the /execution
// subtree. The leaf validates the closed enum
// {"serial_fail_fast"}; the canonical struct keeps the value
// as a plain string so callers can render it without
// re-validating.
type Execution struct {
	Mode string
}

// Check is the canonical projection of a single Plan
// Contract v1 /checks[i] entry. The Mode field carries the
// validated enum value ("run" or "exclude") and the
// remaining fields are pure projections of the wire values.
type Check struct {
	ID               string
	Mode             string
	Argv             []string
	WorkingDirectory string
	TimeoutSeconds   int
	Environment      map[string]string
	Reason           string
}

// Artifact is the canonical projection of a single Plan
// Contract v1 /artifacts[i] entry. Required is a
// three-valued projection: nil means "absent", a non-nil
// pointer carries the boolean value. The leaf preserves
// the absence as nil so the closure package can keep its
// existing semantic for "required missing" without
// re-validating the rule.
type Artifact struct {
	ID        string
	Path      string
	Required  *bool
	MaxBytes  int64
	MediaType string
}

// Policy is the canonical projection of the /policy
// subtree. The four siblings are preserved as pointers so
// the closure package's existing "missing required field"
// diagnostic contract continues to work without re-evaluation.
type Policy struct {
	RequireCleanBefore       *bool
	RequireCleanAfter        *bool
	ForbidTrackedFullDigests *bool
	RequireDiffCheck         *bool
}

// RunnerAuthority is the canonical projection of the
// /runner_authority subtree. A nil *RunnerAuthority means
// the field is absent; a non-nil pointer carries the parsed
// mode and (optionally) the parsed tool block.
type RunnerAuthority struct {
	Mode string
	Tool *ToolAuthority
}

// ToolAuthority is the canonical projection of the
// /runner_authority/tool subtree. Each field is a pure
// projection of the wire value; the leaf enforces the
// length and lowercase-hex shape on Revision, BinarySHA256,
// TreeOID, and TagObjectOID so callers do not re-validate.
//
// Version and TagName are unconstrained strings because the
// wire contract treats them as human-readable metadata only;
// their presence is recorded so callers see exactly what the
// producer declared.
type ToolAuthority struct {
	Revision     string
	TreeOID      string
	BinarySHA256 string
	Version      string
	TagName      string
	TagObjectOID string
}

// DecodeAndValidateFull is the B2-R7 single canonical
// entry point. It performs, in a single call, every stage
// of the Plan Contract v1 pipeline:
//
//  1. strict single-document JSON syntax,
//  2. duplicate-key rejection at every nesting level,
//  3. unknown-field rejection at the root,
//  4. single-document EOF guarantee,
//  5. full semantic validation against the Plan Contract
//     v1 wire rules,
//  6. canonical projection of every accepted field into
//     the ValidatedPlan struct.
//
// On failure the returned DecodeError carries the canonical
// Code, Field, InstancePath, and Message. On success the
// returned ValidatedPlan is the canonical projection the
// closure runner and the evidence package both consume.
//
// The closure runner's ValidatePlan and the evidence
// package's ValidateFullAndProject are both adapters around
// this function. They MUST NOT re-evaluate any wire rule.
func DecodeAndValidateFull(data []byte) (ValidatedPlan, error) {
	root, err := DecodeBytes(data)
	if err != nil {
		return ValidatedPlan{}, err
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return ValidatedPlan{}, &DecodeError{
			Code:    "invalid_json",
			Message: "root is not a JSON object",
		}
	}
	if err := ValidateFullMap(obj); err != nil {
		return ValidatedPlan{}, err
	}
	return projectToValidatedPlan(obj)
}

// projectToValidatedPlan converts the syntactically-validated
// JSON object into the canonical ValidatedPlan. The function
// is pure: it never re-runs ValidateFullMap and never inspects
// any value for shape beyond what the leaf already accepted.
func projectToValidatedPlan(obj map[string]any) (ValidatedPlan, error) {
	plan := ValidatedPlan{}
	if v, ok := obj["contract_version"]; ok {
		if n, ok := v.(json.Number); ok {
			if iv, err := n.Int64(); err == nil {
				plan.ContractVersion = int(iv)
			}
		}
	}
	if v, ok := obj["act_id"].(string); ok {
		plan.ActID = v
	}
	if baseline, ok := obj["baseline"].(map[string]any); ok {
		if v, ok := baseline["commit_oid"].(string); ok {
			plan.Baseline.CommitOID = v
		}
		if v, ok := baseline["tree_oid"].(string); ok {
			plan.Baseline.TreeOID = v
		}
	}
	if execution, ok := obj["execution"].(map[string]any); ok {
		if v, ok := execution["mode"].(string); ok {
			plan.Execution.Mode = trimSpaceLower(v)
		}
	}
	if checks, ok := obj["checks"].([]any); ok {
		for _, item := range checks {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			check := Check{
				ID:               stringFromMap(entry, "id"),
				Mode:             stringFromMap(entry, "mode"),
				WorkingDirectory: stringFromMap(entry, "working_directory"),
				Reason:           stringFromMap(entry, "reason"),
			}
			if v, ok := entry["timeout_seconds"]; ok {
				if n, ok := v.(json.Number); ok {
					if iv, err := n.Int64(); err == nil {
						check.TimeoutSeconds = int(iv)
					}
				}
			}
			if argv, ok := entry["argv"].([]any); ok {
				for _, arg := range argv {
					if s, ok := arg.(string); ok {
						check.Argv = append(check.Argv, s)
					}
				}
			}
			if env, ok := entry["environment"].(map[string]any); ok {
				check.Environment = make(map[string]string, len(env))
				for k, v := range env {
					if s, ok := v.(string); ok {
						check.Environment[k] = s
					}
				}
			}
			plan.Checks = append(plan.Checks, check)
		}
	}
	if artifacts, ok := obj["artifacts"].([]any); ok {
		for _, item := range artifacts {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			artifact := Artifact{
				ID:        stringFromMap(entry, "id"),
				Path:      stringFromMap(entry, "path"),
				MediaType: stringFromMap(entry, "media_type"),
			}
			if v, ok := entry["max_bytes"]; ok {
				if n, ok := v.(json.Number); ok {
					if iv, err := n.Int64(); err == nil {
						artifact.MaxBytes = iv
					}
				}
			}
			if v, ok := entry["required"]; ok {
				if b, ok := v.(bool); ok {
					artifact.Required = &b
				}
			}
			plan.Artifacts = append(plan.Artifacts, artifact)
		}
	}
	if policy, ok := obj["policy"].(map[string]any); ok {
		plan.Policy = projectPolicyMap(policy)
	}
	if ra, ok := obj["runner_authority"]; ok && ra != nil {
		if ram, ok := ra.(map[string]any); ok {
			auth := RunnerAuthority{
				Mode: stringFromMap(ram, "mode"),
			}
			if tool, ok := ram["tool"]; ok && tool != nil {
				if tm, ok := tool.(map[string]any); ok {
					auth.Tool = &ToolAuthority{
						Revision:     stringFromMap(tm, "revision"),
						TreeOID:      stringFromMap(tm, "tree_oid"),
						BinarySHA256: stringFromMap(tm, "binary_sha256"),
						Version:      stringFromMap(tm, "version"),
						TagName:      stringFromMap(tm, "tag_name"),
						TagObjectOID: stringFromMap(tm, "tag_object_oid"),
					}
				}
			}
			plan.RunnerAuthority = &auth
		}
	}
	return plan, nil
}

// projectPolicyMap converts the /policy JSON object into
// the canonical Policy projection. The four siblings are
// preserved as pointers so the absence-of-value signal is
// not lost.
func projectPolicyMap(obj map[string]any) Policy {
	var policy Policy
	if v, ok := obj["require_clean_before"]; ok {
		if b, ok := v.(bool); ok {
			policy.RequireCleanBefore = &b
		}
	}
	if v, ok := obj["require_clean_after"]; ok {
		if b, ok := v.(bool); ok {
			policy.RequireCleanAfter = &b
		}
	}
	if v, ok := obj["forbid_tracked_full_digests"]; ok {
		if b, ok := v.(bool); ok {
			policy.ForbidTrackedFullDigests = &b
		}
	}
	if v, ok := obj["require_diff_check"]; ok {
		if b, ok := v.(bool); ok {
			policy.RequireDiffCheck = &b
		}
	}
	return policy
}

// stringFromMap returns the string value at key in obj, or
// "" if the key is absent or not a string. The canonical
// projection treats absent and wrong-typed values uniformly
// because ValidateFullMap has already rejected malformed
// inputs; any wrong-typed value reaching this point is a
// leaf defect, not a wire-contract violation.
func stringFromMap(obj map[string]any, key string) string {
	if v, ok := obj[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
