package closure

// planContractV1PolicyField returns the /policy subtree descriptor.
// The subtree enforces the contract that every /policy sibling is
// required, is a boolean, and equals true. The descriptor pins the
// constant value so the structural validator emits the documented
// "must equal true" diagnostic whenever a producer submits false.
//
// The four sibling fields form a closed, ordered, deterministic
// set; validatePlanPolicy derives the missing-field order from
// this descriptor's Required slice (via PolicyFieldOrder), so the
// runtime and the descriptor share a single authority.
func planContractV1PolicyField() planFieldDescriptor {
	return planFieldDescriptor{
		JSONName:     "policy",
		GoName:       "Policy",
		Kind:         kindObject,
		Required:     true,
		SemanticRule: "validatePlan(Policy)",
		Description:  "Closure policy fields. Every boolean field is required and must be true.",
		ExampleValue: map[string]any{
			"require_clean_before":        true,
			"require_clean_after":         true,
			"forbid_tracked_full_digests": true,
			"require_diff_check":          true,
		},
		Children: &planObjectDescriptor{
			Path: "/policy",
			Kind: objectClosed,
			Required: []string{
				"require_clean_before",
				"require_clean_after",
				"forbid_tracked_full_digests",
				"require_diff_check",
			},
			Fields: map[string]planFieldDescriptor{
				"require_clean_before": {
					JSONName:      "require_clean_before",
					GoName:        "RequireCleanBefore",
					Kind:          kindBoolean,
					Required:      true,
					Pointer:       true,
					Nullable:      false,
					ConstantValue: true,
					SemanticRule:  "policy.require_clean_before==true",
					Description:   "Whether the worktree must be clean before the ACT.",
					ExampleValue:  true,
				},
				"require_clean_after": {
					JSONName:      "require_clean_after",
					GoName:        "RequireCleanAfter",
					Kind:          kindBoolean,
					Required:      true,
					Pointer:       true,
					Nullable:      false,
					ConstantValue: true,
					SemanticRule:  "policy.require_clean_after==true",
					Description:   "Whether the worktree must be clean after the ACT.",
					ExampleValue:  true,
				},
				"forbid_tracked_full_digests": {
					JSONName:      "forbid_tracked_full_digests",
					GoName:        "ForbidTrackedFullDigests",
					Kind:          kindBoolean,
					Required:      true,
					Pointer:       true,
					Nullable:      false,
					ConstantValue: true,
					SemanticRule:  "policy.forbid_tracked_full_digests==true",
					Description:   "Whether the ACT forbids adding new tracked full digests.",
					ExampleValue:  true,
				},
				"require_diff_check": {
					JSONName:      "require_diff_check",
					GoName:        "RequireDiffCheck",
					Kind:          kindBoolean,
					Required:      true,
					Pointer:       true,
					Nullable:      false,
					ConstantValue: true,
					SemanticRule:  "policy.require_diff_check==true",
					Description:   "Whether `git diff --check` must pass on the F..S range.",
					ExampleValue:  true,
				},
			},
		},
	}
}
