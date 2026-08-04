package closure

import (
	"sort"
	"strings"
)

// planContractV1Descriptor is the private, immutable declaration of
// the Closure Protocol v1 plan wire contract. The descriptor is the
// authoritative source consumed by:
//
//   - structural validation (PlanValidationResult);
//   - typed decoding (the schema, the Go Plan struct);
//   - semantic validation (validatePlan* helpers);
//   - future schema generation (schema/example CLI commands);
//   - future example generation (schema/example CLI commands);
//   - diagnostics (instance paths, enum authority, required sets).
type planContractV1Descriptor struct {
	ContractVersion int

	// Root describes the top-level plan object.
	Root planObjectDescriptor

	// TopLevelAliases are JSON names that LOOK like plan keys but
	// MUST be rejected as unknown_property diagnostics.
	TopLevelAliases []string

	// AliasSubpaths names the locations where aliases have been
	// historically observed (for example `policy.mode`).
	AliasSubpaths []string

	// HistoricalRejectedModes is the ordered set of literal strings
	// that producers have historically tried to pass as execution
	// modes (for example `exitcode`, `gate`).
	HistoricalRejectedModes []string
}

// planContractV1 is the single source of truth for Closure Protocol
// v1. planContractV1() returns a fresh, fully resolved copy on every
// call so the descriptor cannot be mutated by callers.
func planContractV1() planContractV1Descriptor {
	descriptor := planContractV1Descriptor{
		ContractVersion: ContractVersionV1,
		TopLevelAliases: []string{
			"mode",
			"contractVersion",
			"closure_plan",
		},
		AliasSubpaths: []string{
			"/policy/mode",
			"/policy/execution",
			"/policy/execution_mode",
			"/runner_authority/tool_release_exact_v1",
		},
		HistoricalRejectedModes: []string{
			"exitcode",
			"gate",
			"parallel",
			"serial",
			"fail_fast",
			"exit_code",
		},
	}
	descriptor.Root = buildPlanRootV1()
	return descriptor
}

func buildPlanRootV1() planObjectDescriptor {
	root := planObjectDescriptor{
		Path: "",
		Kind: objectClosed,
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
			"baseline":  planContractV1BaselineField(),
			"execution": planContractV1ExecutionField(),
			"checks":    planContractV1ChecksField(),
			"artifacts": planContractV1ArtifactsField(),
			"policy":    planContractV1PolicyField(),
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

// planContractV1BaselineField returns the descriptor for the
// /baseline subtree. The baseline carries the OIDs the runtime
// validators use to anchor the closure window.
func planContractV1BaselineField() planFieldDescriptor {
	return planFieldDescriptor{
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
			Path:     "/baseline",
			Kind:     objectClosed,
			Required: []string{"commit_oid", "tree_oid"},
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
	}
}

// planContractV1ExecutionField returns the descriptor for the
// /execution subtree. /execution.mode is the only required field
// and is closed against every historical rejected alias.
func planContractV1ExecutionField() planFieldDescriptor {
	return planFieldDescriptor{
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
			Kind:     objectClosed,
			Required: []string{"mode"},
			Fields: map[string]planFieldDescriptor{
				"mode": {
					JSONName:      "mode",
					GoName:        "Mode",
					Kind:          kindEnum,
					Required:      true,
					Pointer:       true,
					Nullable:      false,
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
	}
}

// PolicyFieldOrder returns the ordered, closed set of /policy
// sibling names that the structural validator must report as
// missing. The order is read from the descriptor so the descriptor
// remains the single authority: there is no mutable package-global
// field-name slice parallel to it.
func PolicyFieldOrder() []string {
	contract := planContractV1()
	policyField, exists := contract.Root.Fields["policy"]
	if !exists || policyField.Children == nil {
		return nil
	}
	out := make([]string, 0, len(policyField.Children.Required))
	out = append(out, policyField.Children.Required...)
	return out
}

func (o planObjectDescriptor) fieldNamesSorted() []string {
	names := make([]string, 0, len(o.Fields))
	for name := range o.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (o planObjectDescriptor) hasField(jsonName string) bool {
	_, ok := o.Fields[jsonName]
	return ok
}

func canonicalJSONPointer(parent, name string) string {
	if name == "" {
		return parent
	}
	if parent == "" {
		return "/" + name
	}
	return parent + "/" + name
}

func isClineMMDerived(identifier string) bool {
	lower := strings.ToLower(identifier)
	return strings.HasPrefix(lower, "clinemm_") ||
		strings.Contains(lower, "_clinemm") ||
		strings.HasPrefix(lower, "cline_mm_") ||
		strings.Contains(lower, "_cline_mm_")
}

func descriptorIsClean(d planContractV1Descriptor) bool {
	for _, alias := range d.TopLevelAliases {
		if isClineMMDerived(alias) {
			return false
		}
	}
	for _, name := range d.AliasSubpaths {
		if isClineMMDerived(name) {
			return false
		}
	}
	for _, mode := range d.HistoricalRejectedModes {
		if isClineMMDerived(mode) {
			return false
		}
	}
	return walkDescriptorClean(d.Root)
}

func walkDescriptorClean(o planObjectDescriptor) bool {
	for name, field := range o.Fields {
		if isClineMMDerived(name) || isClineMMDerived(field.GoName) {
			return false
		}
		for _, alias := range field.RejectedAliases {
			if isClineMMDerived(alias) {
				return false
			}
		}
		if field.Children != nil && !walkDescriptorClean(*field.Children) {
			return false
		}
		if field.ItemDescriptor != nil && field.ItemDescriptor.Children != nil &&
			!walkDescriptorClean(*field.ItemDescriptor.Children) {
			return false
		}
	}
	return true
}
