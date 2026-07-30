package closure

// planContractV1ArtifactsField is the descriptor for the /artifacts
// array and its per-item object. It is split out of
// plan_contract_descriptor_root.go for the same reason as
// planContractV1ChecksField: keep the root builder reviewable while
// every field's required status, pointer-vs-value semantics, enum
// authority, and description remains reviewable in one screen.
func planContractV1ArtifactsField() planFieldDescriptor {
	return planFieldDescriptor{
		JSONName:     "artifacts",
		GoName:       "Artifacts",
		Kind:         kindArray,
		Required:     true,
		MinItems:     0,
		SemanticRule: "validatePlanArtifacts",
		Description:  "Artifacts the runner must record in the manifest.",
		ExampleValue: []any{},
		ItemDescriptor: &planFieldDescriptor{
			JSONName: "artifacts[]",
			Kind:     kindObject,
			Required: true,
			Children: &planObjectDescriptor{
				Path:     "/artifacts",
				Required: []string{"id", "path", "required", "max_bytes", "media_type"},
				Fields: map[string]planFieldDescriptor{
					"id": {
						JSONName:     "id",
						GoName:       "ID",
						Kind:         kindString,
						Required:     true,
						SemanticRule: "itemIDPattern",
						Description:  "Stable identifier for the artifact.",
						ExampleValue: "summary",
					},
					"path": {
						JSONName:     "path",
						GoName:       "Path",
						Kind:         kindString,
						Required:     true,
						SemanticRule: "validateRepositoryRelativePath",
						Description:  "Repository-relative path the artifact will be recorded at.",
						ExampleValue: ".factory/gate-fast-summary.json",
					},
					"required": {
						JSONName:     "required",
						GoName:       "Required",
						Kind:         kindBoolean,
						Required:     true,
						Pointer:      true,
						SemanticRule: "validatePlanArtifacts(Required)",
						Description:  "Whether the artifact must exist at the role's lifecycle boundary.",
						ExampleValue: true,
					},
					"max_bytes": {
						JSONName:     "max_bytes",
						GoName:       "MaxBytes",
						Kind:         kindInteger,
						Required:     true,
						SemanticRule: "validatePlanArtifacts(MaxBytes)",
						Description:  "Inclusive upper bound on the recorded byte count.",
						ExampleValue: 1048576,
					},
					"media_type": {
						JSONName:     "media_type",
						GoName:       "MediaType",
						Kind:         kindString,
						Required:     true,
						SemanticRule: "validatePlanArtifacts(MediaType)",
						Description:  "IANA media type recorded with the artifact.",
						ExampleValue: "application/json",
					},
					"role": {
						JSONName:      "role",
						GoName:        "Role",
						Kind:          kindEnum,
						Required:      false,
						EnumAuthority: enumAuthorityArtifactRole(),
						SemanticRule:  "ArtifactRoleFor",
						Description:   "Lifecycle role for the artifact.",
						ExampleValue:  string(ArtifactRoleGeneratedOutput),
					},
				},
			},
		},
	}
}
