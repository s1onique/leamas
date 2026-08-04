package closure

// PlanSchema returns the Closure Protocol v1 plan contract descriptor
// as a JSON-serializable map. This is the single authority for the
// descriptor command - do not hand-maintain a second schema inventory.
func PlanSchema() map[string]interface{} {
	contract := planContractV1()
	return descriptorToMap(contract)
}

func descriptorToMap(d planContractV1Descriptor) map[string]interface{} {
	return map[string]interface{}{
		"contract_version":          d.ContractVersion,
		"root":                      objectDescriptorToMap(d.Root),
		"top_level_aliases":         d.TopLevelAliases,
		"alias_subpaths":            d.AliasSubpaths,
		"historical_rejected_modes": d.HistoricalRejectedModes,
	}
}

func objectDescriptorToMap(o planObjectDescriptor) map[string]interface{} {
	fields := make(map[string]interface{}, len(o.Fields))
	for name, field := range o.Fields {
		fields[name] = fieldDescriptorToMap(field)
	}
	return map[string]interface{}{
		"path":     o.Path,
		"kind":     o.Kind.String(),
		"required": o.Required,
		"fields":   fields,
	}
}

func fieldDescriptorToMap(f planFieldDescriptor) map[string]interface{} {
	m := map[string]interface{}{
		"json_name":   f.JSONName,
		"go_name":     f.GoName,
		"kind":        f.Kind.String(),
		"required":    f.Required,
		"description": f.Description,
	}
	if f.ConstantValue != nil {
		m["constant_value"] = f.ConstantValue
	}
	if f.ExampleValue != nil {
		m["example_value"] = f.ExampleValue
	}
	if len(f.EnumAuthority) > 0 {
		m["enum_authority"] = f.EnumAuthority
	}
	if len(f.RejectedAliases) > 0 {
		m["rejected_aliases"] = f.RejectedAliases
	}
	if len(f.ApplicabilityRules) > 0 {
		m["applicability_rules"] = f.ApplicabilityRules
	}
	if f.Children != nil {
		m["children"] = objectDescriptorToMap(*f.Children)
	}
	if f.ItemDescriptor != nil {
		m["item_descriptor"] = itemDescriptorToMap(*f.ItemDescriptor)
	}
	if f.MinItems > 0 {
		m["min_items"] = f.MinItems
	}
	return m
}

func itemDescriptorToMap(i planFieldDescriptor) map[string]interface{} {
	return fieldDescriptorToMap(i)
}
