package closure

// planContractV1RunnerAuthorityField returns the descriptor for the
// /runner_authority subtree. The subtree is extracted from the main
// root builder so each descriptor file stays under the LLM-friendly
// 400-line threshold. The descriptor continues to mirror the V2
// portable runner authority contract:
//
//   - /runner_authority.mode is required and accepts only the
//     declared runner authority enum authority.
//   - /runner_authority.tool is optional and required only for
//     tool_release_exact mode (a downstream semantic rule).
//   - /runner_authority.tool.revision and binary_sha256 are the
//     two tool block required fields.
func planContractV1RunnerAuthorityField() planFieldDescriptor {
	return planFieldDescriptor{
		JSONName:     "runner_authority",
		GoName:       "RunnerAuthority",
		Kind:         kindObject,
		Required:     false,
		Pointer:      true,
		Nullable:     true,
		SemanticRule: "VerifyRunnerBinding (V2 portable runner authority)",
		Description:  "Optional runner authority declaration for Closure Protocol v2 portable runner authority.",
		ExampleValue: map[string]any{
			"mode": string(RunnerAuthoritySubjectExact),
		},
		Children: &planObjectDescriptor{
			Path:     "/runner_authority",
			Required: []string{"mode"},
			Fields: map[string]planFieldDescriptor{
				"mode": {
					JSONName:      "mode",
					GoName:        "Mode",
					Kind:          kindEnum,
					Required:      true,
					EnumAuthority: enumAuthorityRunnerAuthorityMode(),
					SemanticRule:  "VerifyRunnerBinding",
					Description:   "Runner authority mode.",
					ExampleValue:  string(RunnerAuthoritySubjectExact),
				},
				"tool": {
					JSONName:     "tool",
					GoName:       "Tool",
					Kind:         kindObject,
					Required:     false,
					Pointer:      true,
					Nullable:     true,
					SemanticRule: "tool_release_exact required tool block",
					Description:  "Tool authority block.",
					Children: &planObjectDescriptor{
						Path:     "/runner_authority/tool",
						Required: []string{"revision", "binary_sha256"},
						Fields: map[string]planFieldDescriptor{
							"revision": {
								JSONName:     "revision",
								GoName:       "Revision",
								Kind:         kindString,
								Required:     true,
								SemanticRule: "sha1 pattern",
								Description:  "Full lowercase 40-character Git OID of the Leamas source revision.",
								ExampleValue: "1111111111111111111111111111111111111111",
							},
							"tree_oid": {
								JSONName:     "tree_oid",
								GoName:       "TreeOID",
								Kind:         kindString,
								Required:     false,
								SemanticRule: "sha1/sha256 pattern",
								Description:  "Full lowercase Git OID of the Leamas source tree.",
								ExampleValue: "2222222222222222222222222222222222222222",
							},
							"binary_sha256": {
								JSONName:     "binary_sha256",
								GoName:       "BinarySHA256",
								Kind:         kindString,
								Required:     true,
								SemanticRule: "sha256 pattern",
								Description:  "Lowercase SHA-256 hex digest of the runner binary.",
								ExampleValue: "3333333333333333333333333333333333333333333333333333333333333333",
							},
							"version": {
								JSONName:     "version",
								GoName:       "Version",
								Kind:         kindString,
								Required:     false,
								SemanticRule: "declared Leamas version",
								Description:  "Declared Leamas version string.",
								ExampleValue: "v0.0.0",
							},
							"tag_name": {
								JSONName:     "tag_name",
								GoName:       "TagName",
								Kind:         kindString,
								Required:     false,
								SemanticRule: "annotated release tag name",
								Description:  "Annotated release tag name.",
								ExampleValue: "v0.0.0",
							},
							"tag_object_oid": {
								JSONName:     "tag_object_oid",
								GoName:       "TagObjectOID",
								Kind:         kindString,
								Required:     false,
								SemanticRule: "sha1/sha256 pattern",
								Description:  "Annotated tag object OID.",
								ExampleValue: "4444444444444444444444444444444444444444",
							},
						},
					},
				},
			},
		},
	}
}
