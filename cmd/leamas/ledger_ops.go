package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/ledger/importer"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/service"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// runLedgerAccept records an explicit engineering acceptance decision.
//
// The acceptance is only permitted when the linked evidence is admissible:
// the artifact identity was actually verified, the evidence is explicitly
// classified, and it is associated with the criterion it supports.
func runLedgerAccept(args []string) int {
	fs := flag.NewFlagSet("accept", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "work item identity (required)")
	acceptedBy := fs.String("accepted-by", "", "accepting actor (required)")
	basis := fs.String("basis", "", "engineering basis of the acceptance decision")
	sourceHead := fs.String("source-head", "", "source revision the acceptance is bound to")
	expectedRev := fs.Int("expect-revision", 0, "expected revision (required, positive)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" {
		fmt.Fprintln(os.Stderr, "ERROR: accept requires --id")
		return 1
	}
	if *acceptedBy == "" {
		fmt.Fprintln(os.Stderr, "ERROR: accept requires --accepted-by")
		return 1
	}
	if *expectedRev <= 0 {
		fmt.Fprintln(os.Stderr, "ERROR: accept requires a positive --expect-revision")
		return 1
	}
	accepted, err := ledgerService(*root).Accept("ledger-accept", service.AcceptOptions{
		ScopedID:    domain.ScopedItemID(*id),
		ExpectedRev: *expectedRev,
		AcceptedBy:  *acceptedBy,
		Basis:       *basis,
		SourceHead:  *sourceHead,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "item": accepted})
		return 0
	}
	fmt.Printf("accepted %s\n", accepted.Scoped())
	return 0
}

// runLedgerTransfer transfers the ownership of a work item.
//
// The transfer is explicit and bounded: the expected current owner must
// match exactly. Ownership is never replaced silently.
func runLedgerTransfer(args []string) int {
	fs := flag.NewFlagSet("transfer", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "work item identity (required)")
	owner := fs.String("owner", "", "new owner (required)")
	from := fs.String("from", "", "expected current owner (required)")
	expectedRev := fs.Int("expect-revision", 0, "expected revision (optimistic concurrency)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" || *owner == "" || *from == "" {
		fmt.Fprintln(os.Stderr, "ERROR: transfer requires --id, --owner and --from")
		return 1
	}
	updated, err := ledgerService(*root).TransferOwnership("ledger-transfer",
		service.TransferOwnershipOptions{
			ScopedID:    domain.ScopedItemID(*id),
			ExpectedRev: *expectedRev,
			Owner:       *owner,
			FromOwner:   *from,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "item": updated})
		return 0
	}
	fmt.Printf("transferred %s to %s\n", updated.Scoped(), updated.Owner)
	return 0
}

// runLedgerReopen explicitly reopens a work item.
//
// A reopen restores the live lifecycle of a work item. It is distinct from
// a restore, which brings an ARCHIVED record back into the live state.
//
// A reopen makes the work ACTIONABLE again: the lifecycle returns to open
// and the acceptance state returns to pending, so the obligation is live
// again. The historical acceptance DECISION is preserved.
func runLedgerReopen(args []string) int {
	fs := flag.NewFlagSet("reopen", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "work item identity (required)")
	expectedRev := fs.Int("expect-revision", 0, "expected revision (optimistic concurrency)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" {
		fmt.Fprintln(os.Stderr, "ERROR: reopen requires --id")
		return 1
	}
	updated, err := ledgerService(*root).Reopen("ledger-reopen", service.ReopenOptions{
		ScopedID:    domain.ScopedItemID(*id),
		ExpectedRev: *expectedRev,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "item": updated})
		return 0
	}
	fmt.Printf("reopened %s\n", updated.Scoped())
	return 0
}

// runLedgerReconcile produces the durable import reconciliation receipt.
//
// The reconcile path is a READ-ONLY comparison of the source board against
// the canonical state. It reports the conservation of nodes,
// relationships, blockers, original dispositions, criteria and the explicit
// reconciliation records. It never mutates the canonical state.
func runLedgerReconcile(args []string) int {
	fs := flag.NewFlagSet("reconcile", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	source := fs.String("source", "", "path to the source board JSON file (required)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *source == "" {
		fmt.Fprintln(os.Stderr, "ERROR: reconcile requires --source")
		return 1
	}
	data, err := os.ReadFile(*source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: read source board: %v\n", err)
		return 1
	}
	var board importer.SourceBoard
	if err := json.Unmarshal(data, &board); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: decode source board: %v\n", err)
		return 1
	}
	st := store.New(*root)
	existing, err := st.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	plan := importer.BuildPlan(board, existing)
	receipt := importer.BuildReceipt(existing, plan)
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "receipt": receipt})
		return 0
	}
	fmt.Printf("nodes: %d\n", receipt.Nodes)
	fmt.Printf("epics: %d\n", receipt.Epics)
	fmt.Printf("relationships: %d\n", receipt.Relationships)
	fmt.Printf("blockers: %d\n", receipt.Blockers)
	fmt.Printf("original dispositions: %d\n", receipt.OriginalDispositions)
	fmt.Printf("criteria: %d\n", receipt.Criteria)
	fmt.Printf("exit criteria: %d\n", receipt.ExitCriteria)
	for _, rec := range receipt.Reconciliations {
		fmt.Printf("reconciliation: %s: %s: %s\n", rec.ScopedID, rec.Outcome, rec.Detail)
	}
	for _, q := range receipt.Quarantined {
		fmt.Printf("quarantined: %s: %s\n", q.SourceID, q.Reason)
	}
	return 0
}

// runLedgerEvidenceLookup performs the historical lookup of an evidence
// record.
func runLedgerEvidenceLookup(args []string) int {
	fs := flag.NewFlagSet("evidence lookup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "evidence identity (required)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" {
		fmt.Fprintln(os.Stderr, "ERROR: evidence lookup requires --id")
		return 1
	}
	l, err := ledgerService(*root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	for _, e := range l.Evidence {
		if e.EvidenceID == domain.EvidenceID(*id) {
			if *jsonOut {
				emitJSON(map[string]any{"ok": true, "evidence": e})
				return 0
			}
			fmt.Printf("%s %s verified=%t class=%s\n", e.EvidenceID, e.Title, e.Verified, e.Class)
			return 0
		}
	}
	fmt.Fprintf(os.Stderr, "ERROR: record not found: %s\n", *id)
	return 1
}

// runLedgerEvidenceRevalidate re-checks the artifact identity of the
// evidence records against the repository.
//
// An evidence record whose artifact changed, was removed or was replaced
// loses its verified integrity.
func runLedgerEvidenceRevalidate(args []string) int {
	fs := flag.NewFlagSet("evidence revalidate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "evidence identity (re-verify every record when empty)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	results, err := ledgerService(*root).RevalidateEvidence("ledger-revalidate",
		service.RevalidateEvidenceOptions{EvidenceID: domain.EvidenceID(*id)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "validations": results})
		return 0
	}
	for _, v := range results {
		if v.Valid {
			fmt.Printf("%s: valid\n", v.EvidenceID)
			continue
		}
		fmt.Printf("%s: INVALID: %s\n", v.EvidenceID, v.Reason)
	}
	return 0
}
