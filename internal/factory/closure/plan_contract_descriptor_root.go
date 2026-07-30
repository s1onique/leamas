package closure

// buildPlanRootV1 is split out of plan_contract_descriptor.go so the
// root descriptor stays under the LLM-friendly 400-line threshold.
// The body remains bit-identical to the original definition: every
// field descriptor still pins the same JSON name, Go name, type
// category, required status, defaulting annotation, enum authority,
// semantic rule identifier, description, and example value. The
// /runner_authority subtree is delegated to
// planContractV1RunnerAuthorityField() so this file stays reviewable
// in one screen.
// buildPlanRootV1 assembles the root object descriptor. The order of
// the Required list and the iteration order of the Fields map are
// both deterministic: Required is in the order the schema declares;
// Fields is iterated lexicographically by JSONName so that the
// stable codes the structural validator emits have a stable order.
func buildPlanRootV1() planObjectDescriptor {
	root := planObjectDescriptor{
		Path: "",
		Required: []string{
			"contract_version",
			"act_id",
			"baseline",
			"execution",
			"checks",
			"artifacts",
			"policy",
		},
		Fields: map[string]planFieldDescriptor{
			"contract_version": {
				JSONName:      "contract_version",
				GoName:        "ContractVersion",
				Kind:          kindInteger,
				Required:      true,
				ConstantValue: ContractVersionV1,
				SemanticRule:  "unsupported_contract_version",
				Description:   "Integer discriminator identifying the v1 wire format.",
				ExampleValue:  1,
				RejectedAliases: []string{
					"protocol_version",
					"version",
				},
			},
			"act_id": {
				JSONName:     "act_id",
				GoName:       "ActID",
				Kind:         kindString,
				Required:     true,
				SemanticRule: "act_id_pattern",
				Description:  "Repository-scoped identifier of the Action Closure Tracker.",
				ExampleValue: "ACT-LEAMAS-EXAMPLE01",
				RejectedAliases: []string{
					"act",
					"id",
				},
			},
			"baseline": {
				JSONName:     "baseline",
				GoName:       "Baseline",
				Kind:         kindObject,
				Required:     true,
				SemanticRule: "validateOID",
				Description:  "Git identity anchoring the closure window.",
				ExampleValue: map[string]any{
					"commit_oid": "1111111111111111111111111111111111111111",
					"tree_oid":   "2222222222222222222222222222222222222222",
				},
				Children: &planObjectDescriptor{
					Path: "/baseline",
					Required: []string{
						"commit_oid",
						"tree_oid",
					},
					Fields: map[string]planFieldDescriptor{
						"commit_oid": {
							JSONName:     "commit_oid",
							GoName:       "CommitOID",
							Kind:         kindString,
							Required:     true,
							SemanticRule: "validateOID(baseline.commit_oid)",
							Description:  "Full lowercase SHA-1 or SHA-256 Git OID for the baseline commit.",
							ExampleValue: "1111111111111111111111111111111111111111",
						},
						"tree_oid": {
							JSONName:     "tree_oid",
							GoName:       "TreeOID",
							Kind:         kindString,
							Required:     true,
							SemanticRule: "validateOID(baseline.tree_oid)",
							Description:  "Full lowercase Git OID for the baseline tree.",
							ExampleValue: "2222222222222222222222222222222222222222",
						},
					},
				},
			},
			"execution": {
				JSONName:     "execution",
				GoName:       "Execution",
				Kind:         kindObject,
				Required:     true,
				SemanticRule: "validatePlanExecutionMode",
				Description:  "Execution policy descriptor.",
				ExampleValue: map[string]any{
					"mode": string(ExecutionModeSerialFailFast),
				},
				Children: &planObjectDescriptor{
					Path:     "/execution",
					Required: []string{"mode"},
					Fields: map[string]planFieldDescriptor{
						"mode": {
							JSONName:      "mode",
							GoName:        "Mode",
							Kind:          kindEnum,
							Required:      true,
							Pointer:       true,
							EnumAuthority: enumAuthorityExecutionMode(),
							SemanticRule:  "ParseExecutionMode(/execution/mode)",
							Description:   "Execution mode adopted by the runtime.",
							ExampleValue:  string(ExecutionModeSerialFailFast),
							RejectedAliases: []string{
								"execution_mode",
								"strategy",
							},
						},
					},
				},
			},
			"checks":    planContractV1ChecksField(),
			"artifacts": planContractV1ArtifactsField(),
			"policy": {
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
							ConstantValue: true,
							SemanticRule:  "policy.require_diff_check==true",
							Description:   "Whether `git diff --check` must pass on the F..S range.",
							ExampleValue:  true,
						},
					},
				},
			},
			"policy_profile": {
				JSONName:       "policy_profile",
				GoName:         "PolicyProfile",
				Kind:           kindEnum,
				Required:       false,
				EnumAuthority:  enumAuthorityPolicyProfile(),
				SemanticRule:   "validatePlanAuthority",
				Description:    "Optional policy profile identifier.",
				ExampleValue:   PolicyProfileLeamasActV1,
				DefaultingRule: "Omitted → no policy profile enforced.",
			},
			"runner_binding": {
				JSONName:       "runner_binding",
				GoName:         "RunnerBinding",
				Kind:           kindEnum,
				Required:       false,
				EnumAuthority:  enumAuthorityRunnerBinding(),
				SemanticRule:   "VerifyRunnerBinding",
				Description:    "Optional runner binding identifier.",
				ExampleValue:   RunnerBindingTrustedClean,
				DefaultingRule: "Omitted → equivalent to trusted_clean.",
			},
			"runner_authority": planContractV1RunnerAuthorityField(),
		},
	}
	return root
}
