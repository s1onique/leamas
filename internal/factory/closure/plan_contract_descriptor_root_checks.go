package closure

// planContractV1ChecksField is the descriptor for the /checks array
// and its per-item object. It is split out of
// plan_contract_descriptor_root.go so the root builder stays under
// the LLM-friendly 400-line threshold while every field's required
// status, mode-dependent applicability, enum authority, and
// description remains reviewable in one screen.
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
				"environment":       map[string]any{},
			},
		},
		ItemDescriptor: &planFieldDescriptor{
			JSONName: "checks[]",
			Kind:     kindObject,
			Required: true,
			Children: &planObjectDescriptor{
				Path:     "/checks",
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
						JSONName:      "argv",
						GoName:        "Argv",
						Kind:          kindArray,
						Required:      false,
						SemanticRule:  "validateRunnableCheck(Argv)",
						Description:   "Command arguments. Required when sibling mode='run'.",
						ExampleValue:  []any{"true"},
						MinItems:      1,
						ModeDependent: []string{"mode"},
						RejectedAliases: []string{
							"command",
							"cmd",
						},
					},
					"working_directory": {
						JSONName:      "working_directory",
						GoName:        "WorkingDirectory",
						Kind:          kindString,
						Required:      false,
						SemanticRule:  "validateRepositoryRelativePath",
						Description:   "Repository-relative working directory.",
						ExampleValue:  ".",
						ModeDependent: []string{"mode"},
						RejectedAliases: []string{
							"cwd",
							"dir",
						},
					},
					"timeout_seconds": {
						JSONName:      "timeout_seconds",
						GoName:        "TimeoutSeconds",
						Kind:          kindInteger,
						Required:      false,
						SemanticRule:  "validateRunnableCheck(TimeoutSeconds)",
						Description:   "Per-check timeout in seconds.",
						ExampleValue:  60,
						ModeDependent: []string{"mode"},
					},
					"environment": {
						JSONName:      "environment",
						GoName:        "Environment",
						Kind:          kindObject,
						Required:      false,
						SemanticRule:  "validateRunnableCheck(Environment)",
						Description:   "Per-check environment overrides (free-form string map).",
						ExampleValue:  map[string]any{},
						ModeDependent: []string{"mode"},
						Children: &planObjectDescriptor{
							Path:     "/checks/[]/environment",
							Fields:   map[string]planFieldDescriptor{},
							Required: []string{},
						},
					},
					"reason": {
						JSONName:      "reason",
						GoName:        "Reason",
						Kind:          kindString,
						Required:      false,
						SemanticRule:  "validatePlanChecks(Exclude)",
						Description:   "Compact prose explaining why the check is excluded.",
						ExampleValue:  "No source or registration changed.",
						ModeDependent: []string{"mode"},
					},
				},
			},
		},
	}
}
