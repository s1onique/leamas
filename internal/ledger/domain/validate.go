package domain

import (
	"fmt"
)

// Finding is a single validation finding.
type Finding struct {
	// Path is the dotted record path the finding belongs to.
	Path string
	// Code is a stable machine-readable diagnostic code.
	Code string
	// Message is the human-readable explanation.
	Message string
}

// ValidationResult carries every finding produced by a validation pass.
//
// Findings are REJECTIONS: a candidate state that carries any finding is not
// an acceptable canonical state.
//
// Advisories are REPORTS: they describe a state that is legitimate but
// noteworthy, and they never reject the candidate. An advisory exists so
// that a preserved historical fact can be reported without blocking the
// transaction that records it.
type ValidationResult struct {
	Findings []Finding
	// Advisories are reported observations. They never reject a candidate.
	Advisories []Finding
}

// OK reports whether the validation passed.
func (r ValidationResult) OK() bool { return len(r.Findings) == 0 }

// HasAdvisories reports whether the validation produced advisories.
func (r ValidationResult) HasAdvisories() bool { return len(r.Advisories) > 0 }

// Codes emitted by the ledger semantic validator.
const (
	CodeDuplicateIdentity    = "duplicate_identity"
	CodeUnknownReference     = "unknown_reference"
	CodeCycle                = "dependency_cycle"
	CodeInvalidEnum          = "invalid_enum"
	CodeInvalidIdentity      = "invalid_identity"
	CodeMissingField         = "missing_field"
	CodeInsufficientEvidence = "insufficient_evidence"
	CodeLiveGateUnsatisfied  = "live_gate_unsatisfied"
	CodeStaleRevision        = "stale_revision"
	CodePathTraversal        = "path_traversal"
	CodeInvalidDigest        = "invalid_digest"
	CodeDuplicateEvidence    = "duplicate_evidence"
	CodeSelfDependency       = "self_dependency"
	CodeAmbiguousImport      = "ambiguous_import"
	CodeUnresolvedReference  = "unresolved_reference"
	// CodeAcceptanceAuthorityInvalid is an ADVISORY code. It reports that
	// an accepted work item is no longer backed by admissible evidence,
	// typically because the evidence artifact was invalidated after the
	// acceptance was recorded. The historical acceptance is preserved; the
	// report keeps the invalidation visible.
	CodeAcceptanceAuthorityInvalid = "acceptance_authority_invalid"
)

// Error renders the first finding as an error message, or "" when OK.
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

// add appends a finding.
func (r *ValidationResult) add(path, code, message string) {
	r.Findings = append(r.Findings, Finding{Path: path, Code: code, Message: message})
}

// advise appends an advisory. An advisory reports a noteworthy state and
// never rejects the candidate.
func (r *ValidationResult) advise(path, code, message string) {
	r.Advisories = append(r.Advisories, Finding{Path: path, Code: code, Message: message})
}

// ValidateLedger validates the whole canonical ledger state.
//
// The rules are the single authority for ledger semantics:
//
//   - identities are unique and well-formed;
//   - dependency references resolve;
//   - blocking dependency graphs are acyclic;
//   - acceptance never outruns the available evidence;
//   - a live qualification gate is never satisfied structurally;
//   - evidence paths never escape the repository.
func ValidateLedger(l Ledger) ValidationResult {
	var res ValidationResult

	if l.SchemaVersion != LedgerSchemaVersion {
		res.add("ledger.schema_version", CodeInvalidEnum,
			fmt.Sprintf("unsupported ledger schema version %d", l.SchemaVersion))
	}

	validateEpics(l, &res)
	validateItems(l, &res)
	validateActs(l, &res)
	validateEvidence(l, &res)
	validateDecisions(l, &res)
	validateKnowledge(l, &res)
	validateDependencyGraph(l, &res)
	validateAcceptanceAuthority(l, &res)

	sortFindings(res.Findings)
	sortFindings(res.Advisories)
	return res
}

func validateEpics(l Ledger, res *ValidationResult) {
	seen := map[EpicID]bool{}
	for i, e := range l.Epics {
		path := fmt.Sprintf("epics[%d]", i)
		if err := ValidateEpicID(e.EpicID); err != nil {
			res.add(path+".epic_id", CodeInvalidIdentity, err.Error())
		}
		if seen[e.EpicID] {
			res.add(path+".epic_id", CodeDuplicateIdentity,
				fmt.Sprintf("duplicate epic identity %q", string(e.EpicID)))
		}
		seen[e.EpicID] = true
		if e.Title == "" {
			res.add(path+".title", CodeMissingField, "epic title must not be empty")
		}
		if e.Revision < 1 {
			res.add(path+".revision", CodeInvalidEnum, "epic revision must be >= 1")
		}
		if e.Priority != "" && !IsValidPriority(e.Priority) {
			res.add(path+".priority", CodeInvalidEnum,
				fmt.Sprintf("invalid priority %q", string(e.Priority)))
		}
		validateSourceRefs(res, path+".source_refs", e.SourceRefs)
	}
}

func validateItems(l Ledger, res *ValidationResult) {
	seen := map[ScopedItemID]bool{}
	for i, it := range l.Items {
		path := fmt.Sprintf("items[%d]", i)
		if err := ValidateEpicID(it.EpicID); err != nil {
			res.add(path+".epic_id", CodeInvalidIdentity, err.Error())
		}
		if err := ValidateItemID(it.ItemID); err != nil {
			res.add(path+".item_id", CodeInvalidIdentity, err.Error())
		}
		scoped := it.Scoped()
		if seen[scoped] {
			res.add(path+".item_id", CodeDuplicateIdentity,
				fmt.Sprintf("duplicate work item identity %q", scoped.String()))
		}
		seen[scoped] = true
		if it.Title == "" {
			res.add(path+".title", CodeMissingField, "work item title must not be empty")
		}
		if !IsValidItemKind(it.Kind) {
			res.add(path+".kind", CodeInvalidEnum,
				fmt.Sprintf("invalid item kind %q", string(it.Kind)))
		}
		if !IsValidLifecycleState(it.Lifecycle) {
			res.add(path+".lifecycle", CodeInvalidEnum,
				fmt.Sprintf("invalid lifecycle state %q", string(it.Lifecycle)))
		}
		if !IsValidAcceptanceState(it.Acceptance) {
			res.add(path+".acceptance", CodeInvalidEnum,
				fmt.Sprintf("invalid acceptance state %q", string(it.Acceptance)))
		}
		if it.Priority != "" && !IsValidPriority(it.Priority) {
			res.add(path+".priority", CodeInvalidEnum,
				fmt.Sprintf("invalid priority %q", string(it.Priority)))
		}
		if it.Quality != "" && !IsValidEvidenceQuality(it.Quality) {
			res.add(path+".evidence_quality", CodeInvalidEnum,
				fmt.Sprintf("invalid evidence quality %q", string(it.Quality)))
		}
		if it.Revision < 1 {
			res.add(path+".revision", CodeInvalidEnum, "work item revision must be >= 1")
		}
		if it.Archived && it.Lifecycle != LifecycleArchived {
			res.add(path+".lifecycle", CodeInvalidEnum,
				"archived work item must carry lifecycle state 'archived'")
		}
		validateSourceRefs(res, path+".source_refs", it.SourceRefs)
		validateItemLinks(l, it, path, res)
	}
}

// validateSourceRefs validates the byte-exact source references of a record.
//
// A source reference binds a claim to an exact byte range of an exact
// source revision. A reference without a one-based range or without a
// digest does not bind anything.
func validateSourceRefs(res *ValidationResult, path string, refs []SourceRef) {
	for j, ref := range refs {
		rpath := fmt.Sprintf("%s[%d]", path, j)
		if ref.Path == "" {
			res.add(rpath+".path", CodeMissingField,
				"the source reference must name the source path")
		}
		if ref.StartLine < 1 {
			res.add(rpath+".start_line", CodeInvalidEnum,
				fmt.Sprintf("the source reference start line %d is not a one-based line", ref.StartLine))
		}
		if ref.EndLine < ref.StartLine {
			res.add(rpath+".end_line", CodeInvalidEnum,
				fmt.Sprintf("the source reference end line %d precedes its start line %d",
					ref.EndLine, ref.StartLine))
		}
		if ref.SHA256 != "" {
			if err := ValidateSHA256(ref.SHA256); err != nil {
				res.add(rpath+".sha256", CodeInvalidDigest, err.Error())
			}
		}
	}
}

func validateItemLinks(l Ledger, it WorkItem, path string, res *ValidationResult) {
	seenDeps := map[ScopedItemID]bool{}
	for j, dep := range it.Dependencies {
		dpath := fmt.Sprintf("%s.dependencies[%d]", path, j)
		if !IsValidDependencyKind(dep.Kind) {
			res.add(dpath+".kind", CodeInvalidEnum,
				fmt.Sprintf("invalid dependency kind %q", string(dep.Kind)))
		}
		if err := ValidateScopedItemID(dep.Ref); err != nil {
			res.add(dpath+".ref", CodeInvalidIdentity, err.Error())
			continue
		}
		if dep.Ref == it.Scoped() {
			res.add(dpath+".ref", CodeSelfDependency,
				fmt.Sprintf("work item %q depends on itself", it.Scoped().String()))
			continue
		}
		if seenDeps[dep.Ref] {
			res.add(dpath+".ref", CodeDuplicateIdentity,
				fmt.Sprintf("duplicate dependency reference %q", dep.Ref.String()))
		}
		seenDeps[dep.Ref] = true
		if _, ok := l.ItemByScoped(dep.Ref); !ok {
			if !dep.Kind.IsResolvable() {
				// A durable blocker is ALLOWED to dangle: the
				// dangling reference is exactly what keeps the
				// dependent item non-executable.
				continue
			}
			res.add(dpath+".ref", CodeUnknownReference,
				fmt.Sprintf("dangling dependency reference %q", dep.Ref.String()))
		}
	}

	seenEv := map[EvidenceID]bool{}
	for j, ev := range it.Evidence {
		epath := fmt.Sprintf("%s.evidence[%d]", path, j)
		if err := ValidateEvidenceID(ev.EvidenceID); err != nil {
			res.add(epath+".evidence_id", CodeInvalidIdentity, err.Error())
			continue
		}
		if seenEv[ev.EvidenceID] {
			res.add(epath+".evidence_id", CodeDuplicateEvidence,
				fmt.Sprintf("duplicate evidence reference %q", string(ev.EvidenceID)))
		}
		seenEv[ev.EvidenceID] = true
		if !IsValidEvidenceQuality(ev.Quality) {
			res.add(epath+".quality", CodeInvalidEnum,
				fmt.Sprintf("invalid evidence quality %q", string(ev.Quality)))
		}
		if _, ok := l.EvidenceByID(ev.EvidenceID); !ok {
			res.add(epath+".evidence_id", CodeUnknownReference,
				fmt.Sprintf("unknown evidence reference %q", string(ev.EvidenceID)))
		}
	}

	seenSucc := map[ScopedItemID]bool{}
	for j, succ := range it.Successors {
		spath := fmt.Sprintf("%s.successors[%d]", path, j)
		if err := ValidateScopedItemID(succ); err != nil {
			res.add(spath, CodeInvalidIdentity, err.Error())
			continue
		}
		if seenSucc[succ] {
			res.add(spath, CodeDuplicateIdentity,
				fmt.Sprintf("duplicate successor reference %q", succ.String()))
		}
		seenSucc[succ] = true
		if _, ok := l.ItemByScoped(succ); !ok {
			res.add(spath, CodeUnknownReference,
				fmt.Sprintf("unknown successor reference %q", succ.String()))
		}
	}
}
