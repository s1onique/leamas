package fwmi

import "fmt"

// validateItem validates one reconciled work-item record.
//
// The COVERAGE rules are load-bearing: an empty criteria list with an
// UNKNOWN coverage is a fact about incomplete recovery, not a fact about an
// obligation without criteria.
func validateItem(res *ValidationResult, path string, epic EpicRecord, item WorkItem, scoped map[string]bool) {
	if item.ItemID == "" {
		res.add(path+".item_id", CodeMissingField, "the work item must name its item identity")
	}
	if item.SchemaVersion != SchemaVersion {
		res.add(path+".schema_version", CodeInvalidEnum,
			fmt.Sprintf("unsupported work item schema version %d", item.SchemaVersion))
	}
	scopedID := epic.EpicID + "/" + item.ItemID
	if scoped[scopedID] {
		res.add(path+".item_id", CodeDuplicateIdentity,
			fmt.Sprintf("duplicate scoped work item identity %q", scopedID))
	}
	scoped[scopedID] = true
	if item.Title == "" {
		res.add(path+".title", CodeMissingField, "the work item must name its title")
	}
	if !IsValidLifecycle(item.Lifecycle) {
		res.add(path+".lifecycle", CodeInvalidEnum,
			fmt.Sprintf("invalid lifecycle %q", item.Lifecycle))
	}
	if !IsValidQualification(item.Qualification) {
		res.add(path+".qualification", CodeInvalidEnum,
			fmt.Sprintf("invalid qualification %q", item.Qualification))
	}
	if !IsValidCoverage(item.CriteriaCoverage) {
		res.add(path+".criteria_coverage", CodeInvalidEnum,
			fmt.Sprintf("invalid criteria coverage %q", item.CriteriaCoverage))
	}
	if !IsValidCoverage(item.DependencyCoverage) {
		res.add(path+".dependency_coverage", CodeInvalidEnum,
			fmt.Sprintf("invalid dependency coverage %q", item.DependencyCoverage))
	}
	// The COVERAGE semantics are EXPLICIT and are not inferred from the
	// size of the recovered collection:
	//
	//   - COMPLETE means the collection is known to be empty when it is
	//     empty. An empty collection with a COMPLETE coverage is a fact
	//     about the obligation, not a contradiction.
	//   - UNKNOWN means the collection has NOT been recovered. It is
	//     never an assertion that there are no criteria or no
	//     dependencies.
	//
	// The importer never invents criteria or dependencies to make a
	// coverage consistent.
	for j, dep := range item.Dependencies {
		dpath := fmt.Sprintf("%s.dependencies[%d]", path, j)
		if !IsValidRelationshipKind(dep.Kind) {
			res.add(dpath+".kind", CodeInvalidEnum,
				fmt.Sprintf("invalid relationship kind %q", dep.Kind))
		}
		if dep.Target == "" {
			res.add(dpath+".target", CodeMissingField,
				"the relationship must name its target identity")
		}
		validateSourceRefs(res, dpath, dep.SourceRefs)
	}
	validateSourceRefs(res, path, item.SourceRefs)
}

// isLowerHex reports whether the value is a lowercase hexadecimal string.
//
// A SHA-256 digest is bound by its EXACT bytes: an uppercase digest and a
// malformed digest are not the same binding, so neither is accepted.
func isLowerHex(value string) bool {
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}

// validateSourceRefs validates the source references of a record.
//
// A source reference binds a claim to an exact byte range of an exact
// source revision. A reference without a valid range or without a digest
// does not bind anything.
func validateSourceRefs(res *ValidationResult, path string, refs []SourceRef) {
	for j, ref := range refs {
		rpath := fmt.Sprintf("%s.source_refs[%d]", path, j)
		if ref.Path == "" {
			res.add(rpath+".path", CodeMissingField,
				"the source reference must name the source path")
		}
		if ref.StartLine < 1 {
			res.add(rpath+".start_line", CodeInvalidRange,
				fmt.Sprintf("the source reference start line %d is not a one-based line", ref.StartLine))
		}
		if ref.EndLine < ref.StartLine {
			res.add(rpath+".end_line", CodeInvalidRange,
				fmt.Sprintf("the source reference end line %d precedes its start line %d",
					ref.EndLine, ref.StartLine))
		}
		if len(ref.SHA256) != 64 {
			res.add(rpath+".sha256", CodeInvalidEnum,
				"the source reference must carry a SHA-256 digest of its byte range")
		} else if !isLowerHex(ref.SHA256) {
			res.add(rpath+".sha256", CodeInvalidEnum,
				"the source reference must carry a lowercase hexadecimal SHA-256 digest")
		}
	}
}
