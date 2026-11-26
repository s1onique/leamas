package domain

import (
	"fmt"
	"sort"
)

func validateActs(l Ledger, res *ValidationResult) {
	seen := map[ActID]bool{}
	for i, a := range l.Acts {
		path := fmt.Sprintf("acts[%d]", i)
		if err := ValidateActID(a.ActID); err != nil {
			res.add(path+".act_id", CodeInvalidIdentity, err.Error())
		}
		if seen[a.ActID] {
			res.add(path+".act_id", CodeDuplicateIdentity,
				fmt.Sprintf("duplicate act identity %q", string(a.ActID)))
		}
		seen[a.ActID] = true
		if !IsValidActVerdict(a.Verdict) {
			res.add(path+".verdict", CodeInvalidEnum,
				fmt.Sprintf("invalid act verdict %q", string(a.Verdict)))
		}
		if a.Revision < 1 {
			res.add(path+".revision", CodeInvalidEnum, "act revision must be >= 1")
		}
		if a.ItemID != "" {
			if err := ValidateScopedItemID(a.ItemID); err != nil {
				res.add(path+".item_id", CodeInvalidIdentity, err.Error())
			} else if _, ok := l.ItemByScoped(a.ItemID); !ok {
				res.add(path+".item_id", CodeUnknownReference,
					fmt.Sprintf("unknown work item reference %q", a.ItemID.String()))
			}
		}
		seenEv := map[EvidenceID]bool{}
		for j, ev := range a.Evidence {
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
			if _, ok := l.EvidenceByID(ev.EvidenceID); !ok {
				res.add(epath+".evidence_id", CodeUnknownReference,
					fmt.Sprintf("unknown evidence reference %q", string(ev.EvidenceID)))
			}
		}
		if a.IsTerminal() && a.TerminalDisposition == "" {
			res.add(path+".terminal_disposition", CodeMissingField,
				"terminal act verdict requires a preserved terminal disposition")
		}
	}
}

func validateEvidence(l Ledger, res *ValidationResult) {
	seen := map[EvidenceID]bool{}
	for i, e := range l.Evidence {
		path := fmt.Sprintf("evidence[%d]", i)
		if err := ValidateEvidenceID(e.EvidenceID); err != nil {
			res.add(path+".evidence_id", CodeInvalidIdentity, err.Error())
		}
		if seen[e.EvidenceID] {
			res.add(path+".evidence_id", CodeDuplicateIdentity,
				fmt.Sprintf("duplicate evidence identity %q", string(e.EvidenceID)))
		}
		seen[e.EvidenceID] = true
		if e.Kind == "" {
			res.add(path+".kind", CodeMissingField, "evidence kind must not be empty")
		}
		if e.Title == "" {
			res.add(path+".title", CodeMissingField, "evidence title must not be empty")
		}
		if e.Revision < 1 {
			res.add(path+".revision", CodeInvalidEnum, "evidence revision must be >= 1")
		}
		if e.Quality != "" && !IsValidEvidenceQuality(e.Quality) {
			res.add(path+".quality", CodeInvalidEnum,
				fmt.Sprintf("invalid evidence quality %q", string(e.Quality)))
		}
		if e.RelativePath != "" {
			if err := ValidateRelativePath(e.RelativePath); err != nil {
				res.add(path+".relative_path", CodePathTraversal, err.Error())
			}
		}
		if e.SHA256 != "" {
			if err := ValidateSHA256(e.SHA256); err != nil {
				res.add(path+".sha256", CodeInvalidDigest, err.Error())
			}
		}
		if e.ItemID != "" {
			if err := ValidateScopedItemID(e.ItemID); err != nil {
				res.add(path+".item_id", CodeInvalidIdentity, err.Error())
			} else if _, ok := l.ItemByScoped(e.ItemID); !ok {
				res.add(path+".item_id", CodeUnknownReference,
					fmt.Sprintf("unknown work item reference %q", e.ItemID.String()))
			}
		}
	}
}

func validateDecisions(l Ledger, res *ValidationResult) {
	seen := map[DecisionID]bool{}
	for i, d := range l.Decisions {
		path := fmt.Sprintf("decisions[%d]", i)
		if err := ValidateDecisionID(d.DecisionID); err != nil {
			res.add(path+".decision_id", CodeInvalidIdentity, err.Error())
		}
		if seen[d.DecisionID] {
			res.add(path+".decision_id", CodeDuplicateIdentity,
				fmt.Sprintf("duplicate decision identity %q", string(d.DecisionID)))
		}
		seen[d.DecisionID] = true
		if d.Title == "" {
			res.add(path+".title", CodeMissingField, "decision title must not be empty")
		}
		if d.Revision < 1 {
			res.add(path+".revision", CodeInvalidEnum, "decision revision must be >= 1")
		}
		if d.ItemID != "" {
			if err := ValidateScopedItemID(d.ItemID); err != nil {
				res.add(path+".item_id", CodeInvalidIdentity, err.Error())
			}
		}
	}
}

func validateKnowledge(l Ledger, res *ValidationResult) {
	seen := map[KnowledgeID]bool{}
	for i, k := range l.Knowledge {
		path := fmt.Sprintf("knowledge[%d]", i)
		if err := ValidateKnowledgeID(k.KnowledgeID); err != nil {
			res.add(path+".knowledge_id", CodeInvalidIdentity, err.Error())
		}
		if seen[k.KnowledgeID] {
			res.add(path+".knowledge_id", CodeDuplicateIdentity,
				fmt.Sprintf("duplicate knowledge identity %q", string(k.KnowledgeID)))
		}
		seen[k.KnowledgeID] = true
		if k.Title == "" {
			res.add(path+".title", CodeMissingField, "knowledge title must not be empty")
		}
		if k.Revision < 1 {
			res.add(path+".revision", CodeInvalidEnum, "knowledge revision must be >= 1")
		}
		if k.RelativePath != "" {
			if err := ValidateRelativePath(k.RelativePath); err != nil {
				res.add(path+".relative_path", CodePathTraversal, err.Error())
			}
		}
	}
}

// validateDependencyGraph rejects blocking dependency cycles.
//
// Only blocking dependency kinds participate: a "relates" edge can never
// create a cycle, so it is excluded from the graph.
func validateDependencyGraph(l Ledger, res *ValidationResult) {
	graph := map[ScopedItemID][]ScopedItemID{}
	for _, it := range l.Items {
		for _, dep := range it.Dependencies {
			if !dep.Kind.IsBlocking() {
				continue
			}
			if _, ok := l.ItemByScoped(dep.Ref); !ok {
				continue
			}
			graph[it.Scoped()] = append(graph[it.Scoped()], dep.Ref)
		}
	}

	for _, start := range sortedItemIDs(l) {
		if cycle := findCycle(graph, start); len(cycle) > 0 {
			res.add("items."+start.String(), CodeCycle,
				"blocking dependency cycle: "+formatCycle(cycle))
		}
	}
}

// findCycle returns the cycle reachable from start, if any. The returned
// slice starts and ends with the same identity.
func findCycle(graph map[ScopedItemID][]ScopedItemID, start ScopedItemID) []ScopedItemID {
	const (
		unvisited = 0
		onPath    = 1
		done      = 2
	)
	state := map[ScopedItemID]int{}
	var path []ScopedItemID

	var visit func(node ScopedItemID) []ScopedItemID
	visit = func(node ScopedItemID) []ScopedItemID {
		state[node] = onPath
		path = append(path, node)
		for _, next := range graph[node] {
			switch state[next] {
			case onPath:
				for i, p := range path {
					if p == next {
						return append(append([]ScopedItemID{}, path[i:]...), next)
					}
				}
				return []ScopedItemID{next, next}
			case unvisited:
				if c := visit(next); len(c) > 0 {
					return c
				}
			}
		}
		path = path[:len(path)-1]
		state[node] = done
		return nil
	}

	return visit(start)
}

func formatCycle(cycle []ScopedItemID) string {
	out := ""
	for i, c := range cycle {
		if i > 0 {
			out += " -> "
		}
		out += c.String()
	}
	return out
}

func sortedItemIDs(l Ledger) []ScopedItemID {
	ids := make([]ScopedItemID, 0, len(l.Items))
	for _, it := range l.Items {
		ids = append(ids, it.Scoped())
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// validateAcceptanceAuthority enforces the evidence/acceptance separation.
//
// The rules are:
//
//   - an accepted item must carry at least "referenced" evidence quality
//     when it declares acceptance criteria;
//   - an accepted item that requires a live qualification gate must carry
//     "verified" evidence quality, because a structural pass is not a live
//     qualification;
//   - a terminal ACT verdict never retires an unfinished obligation.
func validateAcceptanceAuthority(l Ledger, res *ValidationResult) {
	for i, it := range l.Items {
		path := fmt.Sprintf("items[%d]", i)
		if it.Acceptance != AcceptanceAccepted {
			continue
		}
		if len(it.AcceptanceCriteria) > 0 && it.Quality != EvidenceQualityReferenced &&
			it.Quality != EvidenceQualityVerified {
			res.add(path+".evidence_quality", CodeInsufficientEvidence,
				"accepted work item with acceptance criteria requires at least 'referenced' evidence quality")
		}
		if it.RequiresLiveGate && it.Quality != EvidenceQualityVerified {
			res.add(path+".evidence_quality", CodeLiveGateUnsatisfied,
				"accepted work item requiring a live qualification gate requires 'verified' evidence quality")
		}
	}

	for i, a := range l.Acts {
		if !a.IsTerminal() || a.ItemID == "" {
			continue
		}
		it, ok := l.ItemByScoped(a.ItemID)
		if !ok {
			continue
		}
		if it.Acceptance == AcceptanceAccepted {
			continue
		}
		// A terminal ACT verdict never retires an unfinished obligation.
		// The record is therefore REQUIRED to state explicitly that the
		// obligation is retained. A terminal verdict that silently claims
		// to have closed the obligation is rejected.
		if a.RetainedObligation == "" {
			res.add(fmt.Sprintf("acts[%d].retained_obligation", i), CodeInsufficientEvidence,
				fmt.Sprintf("terminal act verdict %q must record that work item %q keeps its unfinished obligation",
					string(a.Verdict), a.ItemID.String()))
		}
	}
}

// sortFindings orders findings deterministically.
func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Message < findings[j].Message
	})
}
