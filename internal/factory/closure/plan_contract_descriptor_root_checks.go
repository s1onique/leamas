package closure

// planContractV1ChecksField is the descriptor for the /checks array
// and its per-item object. It is split out of
// plan_contract_descriptor_root.go so the root builder stays under
// the LLM-friendly 400-line threshold while every field's required
// status, mode-dependent applicability, enum authority, and
// description remains reviewable in one screen.
//
// Mode-dependent rules (recovered from validatePlanChecks /
// validateRunnableCheck):
//   - mode=run: argv MUST be present and non-empty; working_directory,
//     timeout_seconds, environment governed by the runtime validators;
//     reason MUST be absent.
//   - mode=exclude: reason MUST be present and compact final prose;
//     argv, working_directory, timeout_seconds, environment MUST be
//     absent.
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
						Description:  "Command arguments. Required when sibling mode='run'.",
						ExampleValue: []any{"true"},
						MinItems:     1,
						Applicability: &fieldApplicability{
							Sibling:  "mode",
							Value:    CheckModeRun,
							Required: true,
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
						Description:  "Repository-relative working directory.",
						ExampleValue: ".",
						Applicability: &fieldApplicability{
							Sibling:  "mode",
							Value:    CheckModeRun,
							Required: false,
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
						Description:  "Per-check timeout in seconds.",
						ExampleValue: 60,
						Applicability: &fieldApplicability{
							Sibling:  "mode",
							Value:    CheckModeRun,
							Required: false,
						},
					},
					"environment": {
						JSONName:     "environment",
						GoName:       "Environment",
						Kind:         kindObject,
						Required:     false,
						SemanticRule: "validateRunnableCheck(Environment)",
						Description:  "Per-check environment overrides (free-form string map).",
						ExampleValue: map[string]any{"FOO": "bar"},
						Applicability: &fieldApplicability{
							Sibling:  "mode",
							Value:    CheckModeRun,
							Required: false,
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
						Description:  "Compact prose explaining why the check is excluded.",
						ExampleValue: "No source or registration changed.",
						Applicability: &fieldApplicability{
							Sibling:   "mode",
							Value:     CheckModeExclude,
							Required:  true,
							Forbidden: false,
						},
					},
				},
			},
		},
	}
}
