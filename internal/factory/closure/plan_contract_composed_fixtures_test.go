package closure

// plan_contract_composed_fixtures_test.go centralises the v1
// closure-plan JSON fixtures the composed authority and
// applicability tests share. Splitting the JSON constants into
// a focused file keeps every individual file under the
// LLM-friendly 400-line threshold while each fixture remains
// short enough to read in a single screen.

// canonicalComposedPlan returns a v1 closure plan JSON document
// that passes every validator (structural, applicability, typed,
// semantic). The fixture is intentionally plain to make
// assertions about the diagnostic stream easy to construct.
// Splitting the JSON across lines keeps every source line well
// under the LLM-friendly 240-character threshold.
func canonicalComposedPlan() []byte {
	return []byte(canonicalComposedPlanJSON)
}

const canonicalComposedPlanJSON = `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-COMPOSED",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {
      "id": "noop",
      "mode": "run",
      "argv": ["true"],
      "working_directory": ".",
      "timeout_seconds": 60,
      "environment": {}
    }
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// composedPlanDuplicateCheckID returns a structurally valid plan
// whose /checks array contains two items sharing the same id.
// The structural validator accepts the document; the semantic
// validator (validatePlanChecks) rejects it as a duplicate id.
func composedPlanDuplicateCheckID() []byte {
	return []byte(composedPlanDuplicateCheckIDJSON)
}

const composedPlanDuplicateCheckIDJSON = `{
  "contract_version": 1,
  "act_id": "ACT-DUP-ID",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "x", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}},
    {"id": "x", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// composedPlanMissingRunArgv returns a run-mode check whose
// argv is absent; the structural applicability walker must
// emit required_property_missing at /checks/0/argv.
func composedPlanMissingRunArgv() []byte {
	return []byte(composedPlanMissingRunArgvJSON)
}

const composedPlanMissingRunArgvJSON = `{
  "contract_version": 1,
  "act_id": "ACT-MISSING-ARGV",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "noop", "mode": "run", "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// composedPlanRunReasonForbidden returns a run-mode check that
// also carries a `reason`; the applicability walker must emit
// semantic_constraint_failed at /checks/0/reason.
func composedPlanRunReasonForbidden() []byte {
	return []byte(composedPlanRunReasonForbiddenJSON)
}

const composedPlanRunReasonForbiddenJSON = `{
  "contract_version": 1,
  "act_id": "ACT-RUN-REASON",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}, "reason": "noop"}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// composedPlanMissingExcludeReason returns an exclude-mode check
// without a `reason`; applicability must reject it as a missing
// required property.
func composedPlanMissingExcludeReason() []byte {
	return []byte(composedPlanMissingExcludeReasonJSON)
}

const composedPlanMissingExcludeReasonJSON = `{
  "contract_version": 1,
  "act_id": "ACT-EXCL-NOREASON",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "x", "mode": "exclude"}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// composedPlanExcludeWithArgv returns an exclude-mode check that
// also carries an argv; applicability must reject it as a
// forbidden property.
func composedPlanExcludeWithArgv() []byte {
	return []byte(composedPlanExcludeWithArgvJSON)
}

const composedPlanExcludeWithArgvJSON = `{
  "contract_version": 1,
  "act_id": "ACT-EXCL-ARGV",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "x", "mode": "exclude", "reason": "noop", "argv": ["true"]}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// emptyReasonPlan returns a run-mode check that carries an
// empty-string `reason`. The applicability walker rejects it
// because reason is forbidden under mode=run.
func emptyReasonPlan() []byte {
	return []byte(emptyReasonPlanJSON)
}

const emptyReasonPlanJSON = `{
  "contract_version": 1,
  "act_id": "ACT-FORB-REASON-EMPTY",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "x", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}, "reason": ""}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// argvEmptyArrayExcludePlan returns an exclude-mode check with
// an empty argv array. The structural validator's minItems
// check fires before applicability, producing an invalid_type
// diagnostic; the test accepts either that or the applicability
// semantic_constraint_failed diagnostic.
func argvEmptyArrayExcludePlan() []byte {
	return []byte(argvEmptyArrayExcludePlanJSON)
}

const argvEmptyArrayExcludePlanJSON = `{
  "contract_version": 1,
  "act_id": "ACT-FORB-ARGV-EMPTY",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "x", "mode": "exclude", "reason": "noop", "argv": []}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// workingDirectoryEmptyExcludePlan returns an exclude-mode
// check with an empty working_directory string.
func workingDirectoryEmptyExcludePlan() []byte {
	return []byte(workingDirectoryEmptyExcludePlanJSON)
}

const workingDirectoryEmptyExcludePlanJSON = `{
  "contract_version": 1,
  "act_id": "ACT-FORB-WD-EMPTY",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "x", "mode": "exclude", "reason": "noop", "working_directory": ""}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// timeoutSecondsZeroExcludePlan returns an exclude-mode check
// with timeout_seconds=0. The applicability walker rejects
// timeout_seconds whenever it is present at all, so the
// forbidden-state diagnostic fires.
func timeoutSecondsZeroExcludePlan() []byte {
	return []byte(timeoutSecondsZeroExcludePlanJSON)
}

const timeoutSecondsZeroExcludePlanJSON = `{
  "contract_version": 1,
  "act_id": "ACT-FORB-TS-ZERO",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "x", "mode": "exclude", "reason": "noop", "timeout_seconds": 0}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// environmentEmptyExcludePlan returns an exclude-mode check
// with an empty environment object.
func environmentEmptyExcludePlan() []byte {
	return []byte(environmentEmptyExcludePlanJSON)
}

const environmentEmptyExcludePlanJSON = `{
  "contract_version": 1,
  "act_id": "ACT-FORB-ENV-EMPTY",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "x", "mode": "exclude", "reason": "noop", "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`
