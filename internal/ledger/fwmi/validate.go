package fwmi

import (
	"fmt"
	"sort"
)

// Finding is one strict-validation finding of an interchange package.
//
// A finding is a REJECTION: an interchange package that carries a finding
// is not an acceptable migration input.
type Finding struct {
	// Path is the dotted path of the offending record.
	Path string `json:"path"`
	// Code is the stable machine-readable diagnostic code.
	Code string `json:"code"`
	// Message is the human-readable explanation.
	Message string `json:"message"`
}

// Diagnostic codes emitted by the interchange validator.
const (
	CodeDuplicateIdentity = "duplicate_identity"
	CodeMissingField      = "missing_field"
	CodeInvalidEnum       = "invalid_enum"
	CodeInvalidRange      = "invalid_source_range"
	CodeMissingCoverage   = "missing_coverage"
)

// ValidationResult carries every finding of a validation pass.
type ValidationResult struct {
	Findings []Finding `json:"findings"`
}

// OK reports whether the package passed the strict validation.
func (r ValidationResult) OK() bool { return len(r.Findings) == 0 }

// Error renders the first finding as an error message.
func (r ValidationResult) Error() string {
	if r.OK() {
		return ""
	}
	f := r.Findings[0]
	return fmt.Sprintf("%s: %s", f.Code, f.Message)
}

// Errors renders every finding as an error slice.
func (r ValidationResult) Errors() []error {
	out := make([]error, 0, len(r.Findings))
	for _, f := range r.Findings {
		out = append(out, fmt.Errorf("%s: %s", f.Code, f.Message))
	}
	return out
}

func (r *ValidationResult) add(path, code, message string) {
	r.Findings = append(r.Findings, Finding{Path: path, Code: code, Message: message})
}

// Validate validates the interchange package strictly.
//
// The validation is the DETERMINISTIC half of the migration: the semantic
// interpretation was produced elsewhere, and every load-bearing fact of that
// interpretation is checked here before it may reach the canonical ledger.
func (p *Package) Validate() error {
	res := p.Validation()
	if !res.OK() {
		return res
	}
	return nil
}

// Validation computes the strict validation findings of the package.
func (p *Package) Validation() ValidationResult {
	var res ValidationResult
	if p.Manifest.SchemaVersion != SchemaVersion {
		res.add("source_manifest.schema_version", CodeInvalidEnum,
			fmt.Sprintf("unsupported source manifest schema version %d", p.Manifest.SchemaVersion))
	}
	if p.Manifest.SourceHead == "" {
		res.add("source_manifest.source_head", CodeMissingField,
			"the source manifest must name the source HEAD it describes")
	}
	if p.Manifest.Repository == "" {
		// A MISSING source repository must not establish a source binding.
		// The repository is the identity of the examined source: without it
		// the records would be bound to an empty source identity, and the
		// binding would be indistinguishable from a verified one.
		res.add("source_manifest.repository", CodeMissingField,
			"the source manifest must name the source repository it describes")
	}

	epicIDs := map[string]bool{}
	scopedIDs := map[string]bool{}
	for i, epic := range p.Epics {
		path := fmt.Sprintf("epics[%d]", i)
		if epic.EpicID == "" {
			res.add(path+".epic_id", CodeMissingField, "the epic record must name its epic identity")
		}
		if epicIDs[epic.EpicID] {
			res.add(path+".epic_id", CodeDuplicateIdentity,
				fmt.Sprintf("duplicate epic identity %q", epic.EpicID))
		}
		epicIDs[epic.EpicID] = true
		if !IsValidLifecycle(epic.Lifecycle) {
			res.add(path+".lifecycle", CodeInvalidEnum,
				fmt.Sprintf("invalid lifecycle %q", epic.Lifecycle))
		}
		if epic.SchemaVersion != SchemaVersion {
			res.add(path+".schema_version", CodeInvalidEnum,
				fmt.Sprintf("unsupported epic schema version %d", epic.SchemaVersion))
		}
		validateSourceRefs(&res, path, epic.SourceRefs)
		for j, item := range epic.Items {
			validateItem(&res, fmt.Sprintf("%s.items[%d]", path, j), epic, item, scopedIDs)
		}
	}

	actIDs := map[string]bool{}
	for i, rec := range p.Historical.Records {
		path := fmt.Sprintf("historical_index.records[%d]", i)
		if rec.ACTID == "" {
			res.add(path+".act_id", CodeMissingField, "the historical record must name its ACT identity")
		}
		if actIDs[rec.ACTID] {
			res.add(path+".act_id", CodeDuplicateIdentity,
				fmt.Sprintf("duplicate historical ACT identity %q", rec.ACTID))
		}
		actIDs[rec.ACTID] = true
		if rec.SchemaVersion != SchemaVersion {
			res.add(path+".schema_version", CodeInvalidEnum,
				fmt.Sprintf("unsupported historical record schema version %d", rec.SchemaVersion))
		}
		validateSourceRefs(&res, path, rec.SourceRefs)
	}

	for i, rec := range p.Reconciliation.Records {
		path := fmt.Sprintf("reconciliation.records[%d]", i)
		if rec.Kind == "" {
			res.add(path+".kind", CodeMissingField, "the reconciliation record must name its kind")
		}
		if len(rec.Identities) == 0 {
			res.add(path+".identities", CodeMissingField,
				"the reconciliation record must name at least one identity")
		}
		if rec.SchemaVersion != SchemaVersion {
			res.add(path+".schema_version", CodeInvalidEnum,
				fmt.Sprintf("unsupported reconciliation record schema version %d", rec.SchemaVersion))
		}
		validateSourceRefs(&res, path, rec.SourceRefs)
	}

	sort.SliceStable(res.Findings, func(i, j int) bool {
		if res.Findings[i].Path != res.Findings[j].Path {
			return res.Findings[i].Path < res.Findings[j].Path
		}
		return res.Findings[i].Code < res.Findings[j].Code
	})
	return res
}
