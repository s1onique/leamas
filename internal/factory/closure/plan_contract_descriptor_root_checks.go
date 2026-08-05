package closure

// planContractV1ChecksField is the descriptor for the /checks array
// and its per-item object. It is split out of
// plan_contract_descriptor_root.go so the root builder stays under
// the LLM-friendly 400-line threshold while every field's required
// status, mode-dependent applicability, enum authority, and
// description remains reviewable in one screen.
//
// Mode-dependent rules (recovered from validatePlanChecks /
// validateRunnableCheck) are encoded exhaustively via
// ApplicabilityRules so both branches are documented:
//
//   - mode=run: argv, working_directory, timeout_seconds, and
//     environment required; reason forbidden.
//   - mode=exclude: reason required; argv/working_directory/
//     timeout_seconds/environment forbidden.
//
// Working-directory and timeout_seconds presence rules align
// the descriptor with the runtime validation that has always
// required them for run-mode checks. The structural applicability
// walker reports `required_property_missing` with the exact
// property_name when the field is absent; the structural value
// constraints (minLength/pattern/minimum/maximum) reject empty,
// invalid, or out-of-range values with `invalid_type` plus a
// stable keyword.
func planContractV1ChecksField() planFieldDescriptor {
	return planFieldDescriptor{
		JSONName:     "checks",
		GoName:       "Checks",
		Kind:         kindArray,
		Required:     true,
		MinItems:     1,
		SemanticRule: "validatePlanChecks",
		Description:  "Ordered list of checks the runner must execute.",
		ExampleValue: []any{
			map[string]any{
				"id":                "noop",
				"mode":              CheckModeRun,
				"argv":              []any{"true"},
				"working_directory": ".",
				"timeout_seconds":   60,
				"environment":       map[string]any{"FOO": "bar"},
			},
		},
		ItemDescriptor: &planFieldDescriptor{
			JSONName: "checks[]",
			Kind:     kindObject,
			Required: true,
			Children: &planObjectDescriptor{
				Path:     "/checks",
				Kind:     objectClosed,
				Required: []string{"id", "mode"},
				Fields: map[string]planFieldDescriptor{
					"id": {
						JSONName:     "id",
						GoName:       "ID",
						Kind:         kindString,
						Required:     true,
						SemanticRule: "itemIDPattern",
						Description:  "Stable, unique identifier for the check.",
						ExampleValue: "noop",
					},
					"mode": {
						JSONName:      "mode",
						GoName:        "Mode",
						Kind:          kindEnum,
						Required:      true,
						EnumAuthority: enumAuthorityCheckMode(),
						SemanticRule:  "validatePlanChecks switch",
						Description:   "Whether the check is run or excluded.",
						ExampleValue:  CheckModeRun,
					},
					"argv": {
						JSONName:     "argv",
						GoName:       "Argv",
						Kind:         kindArray,
						Required:     false,
						SemanticRule: "validateRunnableCheck(Argv)",
						Description:  "Command arguments. Required when mode='run'; forbidden when mode='exclude'.",
						ExampleValue: []any{"true"},
						MinItems:     1,
						ApplicabilityRules: []fieldApplicabilityRule{
							{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
							{Sibling: "mode", Value: CheckModeExclude, Presence: PresenceForbidden},
						},
						RejectedAliases: []string{
							"command",
							"cmd",
						},
						ItemDescriptor: &planFieldDescriptor{
							JSONName:     "argv[]",
							Kind:         kindString,
							Description:  "Command-line argument (string).",
							ExampleValue: "true",
						},
					},
					"working_directory": {
						JSONName:     "working_directory",
						GoName:       "WorkingDirectory",
						Kind:         kindString,
						Required:     false,
						SemanticRule: "validateRepositoryRelativePath",
						Description:  "Repository-relative working directory. Required when mode='run'; forbidden when mode='exclude'. Must be non-empty, must not be an absolute path, must not start with '..', and must be lexically clean.",
						ExampleValue: ".",
						MinLength:    1,
						Pattern:      `^[^/]+(/[^/]+)*$`,
						ApplicabilityRules: []fieldApplicabilityRule{
							{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
							{Sibling: "mode", Value: CheckModeExclude, Presence: PresenceForbidden},
						},
						RejectedAliases: []string{
							"cwd",
							"dir",
						},
					},
					"timeout_seconds": {
						JSONName:     "timeout_seconds",
						GoName:       "TimeoutSeconds",
						Kind:         kindInteger,
						Required:     false,
						SemanticRule: "validateRunnableCheck(TimeoutSeconds)",
						Description:  "Per-check timeout in seconds, inclusive bounds [1, 600]. Required when mode='run'; forbidden when mode='exclude'.",
						ExampleValue: 60,
						Minimum:      1,
						Maximum:      600,
						ApplicabilityRules: []fieldApplicabilityRule{
							{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
							{Sibling: "mode", Value: CheckModeExclude, Presence: PresenceForbidden},
						},
					},
					"environment": {
						JSONName:     "environment",
						GoName:       "Environment",
						Kind:         kindObject,
						Required:     false,
						SemanticRule: "validateRunnableCheck(Environment)",
						Description:  "Per-check environment overrides (free-form string map). Required when mode='run'; {} means no environment overrides; forbidden when mode='exclude'.",
						ExampleValue: map[string]any{"FOO": "bar"},
						ApplicabilityRules: []fieldApplicabilityRule{
							{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
							{Sibling: "mode", Value: CheckModeExclude, Presence: PresenceForbidden},
						},
						Children: &planObjectDescriptor{
							Path: "/checks/[]/environment",
							Kind: objectStringMap,
						},
					},
					"reason": {
						JSONName:     "reason",
						GoName:       "Reason",
						Kind:         kindString,
						Required:     false,
						SemanticRule: "validatePlanChecks(Exclude)",
						Description:  "Compact prose explaining why the check is excluded. Required when mode='exclude'; forbidden when mode='run'.",
						ExampleValue: "No source or registration changed.",
						ApplicabilityRules: []fieldApplicabilityRule{
							{Sibling: "mode", Value: CheckModeExclude, Presence: PresenceRequired},
							{Sibling: "mode", Value: CheckModeRun, Presence: PresenceForbidden},
						},
					},
				},
			},
		},
	}
}
