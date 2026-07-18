// Package main - record_show.go
//
// Shared orchestration for "show one record by ID" CLI commands. Both
// `leamas witness claim show <id>` and `leamas witness evidence show <id>`
// are driven by runRecordShow here; the per-entity differences live in
// recordShowSpec instances defined next to each command.
//
// Why one shared operation?
//
//   - The duplicated 504-token geometry (claim_commands.go:268-340 and
//     evidence_commands.go:310-382) covers flag parsing, arg validation,
//     run-id validation, run bundle opening, store creation, error
//     formatting, exit-code selection, and JSON/text dispatch. None of
//     those differ across commands.
//   - The differences (ID validation function, read function, not-found
//     sentinel, JSON envelope field, text rendering) are captured by
//     named fields on recordShowSpec. There is no boolean policy flag
//     that hides a semantic branch.
//   - The shared operation remains narrowly CLI-orchestration specific.
//     It does not own a reusable domain capability.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/s1onique/leamas/internal/witness/claim"
	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// recordShowSpec captures the entity-specific behaviour of a "show" CLI
// command. The shared orchestration in runRecordShow is responsible for
// flag parsing, arg validation, run-id validation, run bundle opening,
// error formatting, exit-code selection, and JSON/text dispatch.
//
// Each field name describes one named decision; no field is a generic
// `isClaim bool` style policy. The spec is constructed at the call site
// (claim_commands.go / evidence_commands.go) so each command file
// remains a readable orchestration boundary.
type recordShowSpec struct {
	// KindName is the lowercase noun used in error messages and the
	// flag-set name (e.g. "claim", "evidence").
	KindName string

	// PosArgLabel is the placeholder printed by the "<kind> show
	// requires <label>" error (e.g. "<claim-id>", "<evidence-id>").
	PosArgLabel string

	// ValidateID parses and validates the raw positional ID. The raw
	// string comes directly from fs.Arg(0); the helper owns its own
	// type coercion (e.g. wrapping into claim.ClaimID).
	ValidateID func(rawID string) error

	// NotFoundErr is the sentinel returned by ReadRecord when the
	// record is absent. runRecordShow uses errors.Is to translate it
	// into a "<kind> not found: <id>" error message.
	NotFoundErr error

	// ReadRecord loads the record from the store by raw ID. It returns
	// any concrete record (typically *claim.Claim or *claim.Evidence);
	// the renderer fields below know how to interpret the value.
	ReadRecord func(store claim.Store, rawID string) (any, error)

	// RenderText writes the human-readable rendering of record to w.
	// bundlePath is the absolute run-bundle directory and is currently
	// informational; commands that need it for output can use it.
	RenderText func(w io.Writer, bundlePath string, record any)

	// RenderJSON writes the JSON encoding of record to w. It returns
	// any error from marshalling; the caller prints a generic error
	// to stderr on failure.
	RenderJSON func(w io.Writer, record any) error
}

// runRecordShow executes the shared orchestration of "show" commands.
// It returns the process exit code. Any non-zero return indicates the
// command has already printed the appropriate error to stderr.
//
// runRecordShow deliberately has no domain knowledge beyond run bundles
// and claim stores; entity-specific behaviour flows entirely through
// the recordShowSpec argument.
func runRecordShow(args []string, spec recordShowSpec) int {
	var root, runIDRaw string
	var jsonMode bool

	fs := flag.NewFlagSet(spec.KindName+" show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&root, "root", defaultRunBundleRoot, "root directory for run bundles")
	fs.StringVar(&runIDRaw, "run-id", "", "run bundle ID (required)")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "ERROR: %s show requires %s\n", spec.KindName, spec.PosArgLabel)
		fs.Usage()
		return 1
	}
	if runIDRaw == "" {
		fmt.Fprintf(os.Stderr, "ERROR: %s show requires --run-id\n", spec.KindName)
		fs.Usage()
		return 1
	}
	if root == "" {
		fmt.Fprintln(os.Stderr, "ERROR: run bundle root must be non-empty")
		return 1
	}

	runID := runbundle.RunID(runIDRaw)
	if err := runbundle.ValidateRunID(runID); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid run ID: %v\n", err)
		return 1
	}

	rawID := fs.Arg(0)
	if err := spec.ValidateID(rawID); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid %s ID: %v\n", spec.KindName, err)
		return 1
	}

	bundle, _, err := runbundle.Open(root, runID)
	if err != nil {
		printRunBundleError(root, runID, err)
		return 1
	}

	store := claim.NewStore(bundle)
	record, err := spec.ReadRecord(store, rawID)
	if err != nil {
		if errors.Is(err, spec.NotFoundErr) {
			fmt.Fprintf(os.Stderr, "ERROR: %s not found: %s\n", spec.KindName, rawID)
		} else {
			fmt.Fprintf(os.Stderr, "ERROR: failed to read %s: %v\n", spec.KindName, err)
		}
		return 1
	}

	if jsonMode {
		if err := spec.RenderJSON(os.Stdout, record); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to marshal %s JSON: %v\n", spec.KindName, err)
			return 1
		}
		return 0
	}

	spec.RenderText(os.Stdout, bundle.Path, record)
	return 0
}

// renderClaimRecordJSON writes the canonical JSON envelope for a single
// claim. The envelope shape `{ "ok": true, "claim": <record> }` is part
// of the public witness claim CLI contract.
func renderClaimRecordJSON(w io.Writer, record any) error {
	clm, ok := record.(*claim.Claim)
	if !ok {
		return fmt.Errorf("renderClaimRecordJSON: expected *claim.Claim, got %T", record)
	}
	out := struct {
		OK    bool         `json:"ok"`
		Claim *claim.Claim `json:"claim"`
	}{
		OK:    true,
		Claim: clm,
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(data))
	return nil
}

// renderClaimRecordText writes the canonical text rendering for a single
// claim. The line ordering and labels (Claim/Run/Status/Verdict/
// Statement/Evidence/Notes) are part of the public witness claim CLI
// contract and must remain stable across the refactor.
func renderClaimRecordText(w io.Writer, _ string, record any) {
	clm, ok := record.(*claim.Claim)
	if !ok {
		fmt.Fprintf(w, "ERROR: renderClaimRecordText: expected *claim.Claim, got %T\n", record)
		return
	}
	fmt.Fprintf(w, "Claim: %s\n", clm.ID)
	fmt.Fprintf(w, "Run: %s\n", clm.RunID)
	fmt.Fprintf(w, "Status: %s\n", clm.Status)
	fmt.Fprintf(w, "Verdict: %s\n", clm.Verdict)
	fmt.Fprintf(w, "Statement: %s\n", clm.Statement)
	if len(clm.EvidenceIDs) > 0 {
		fmt.Fprintln(w, "Evidence:")
		for _, eid := range clm.EvidenceIDs {
			fmt.Fprintf(w, "  %s\n", eid)
		}
	}
	if clm.Notes != "" {
		fmt.Fprintf(w, "Notes: %s\n", clm.Notes)
	}
}

// renderEvidenceRecordJSON writes the canonical JSON envelope for a
// single evidence record. The envelope shape `{ "ok": true, "evidence":
// <record> }` is part of the public witness evidence CLI contract.
func renderEvidenceRecordJSON(w io.Writer, record any) error {
	ev, ok := record.(*claim.Evidence)
	if !ok {
		return fmt.Errorf("renderEvidenceRecordJSON: expected *claim.Evidence, got %T", record)
	}
	out := struct {
		OK       bool            `json:"ok"`
		Evidence *claim.Evidence `json:"evidence"`
	}{
		OK:       true,
		Evidence: ev,
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(data))
	return nil
}

// renderEvidenceRecordText writes the canonical text rendering for a
// single evidence record. The line ordering and labels (Evidence/Run/
// Kind/Role/Title/Path/Summary) are part of the public witness evidence
// CLI contract and must remain stable across the refactor.
func renderEvidenceRecordText(w io.Writer, _ string, record any) {
	ev, ok := record.(*claim.Evidence)
	if !ok {
		fmt.Fprintf(w, "ERROR: renderEvidenceRecordText: expected *claim.Evidence, got %T\n", record)
		return
	}
	fmt.Fprintf(w, "Evidence: %s\n", ev.ID)
	fmt.Fprintf(w, "Run: %s\n", ev.RunID)
	fmt.Fprintf(w, "Kind: %s\n", ev.Kind)
	fmt.Fprintf(w, "Role: %s\n", ev.Role)
	fmt.Fprintf(w, "Title: %s\n", ev.Title)
	if ev.RelativePath != "" {
		fmt.Fprintf(w, "Path: %s\n", ev.RelativePath)
	}
	if ev.Summary != "" {
		fmt.Fprintf(w, "Summary: %s\n", ev.Summary)
	}
}
