package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/s1onique/leamas/internal/witness/claim"
	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// ============================================================================
// Attach evidence to claim
// ============================================================================

type claimAttachEvidenceOptions struct {
	Root       string
	RunID      string
	ClaimID    string
	EvidenceID string
	JSON       bool
}

func runWitnessClaimAttachEvidence(args []string) int {
	var opts claimAttachEvidenceOptions
	fs := flag.NewFlagSet("attach-evidence", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.Root, "root", defaultRunBundleRoot, "root directory for run bundles")
	fs.StringVar(&opts.RunID, "run-id", "", "run bundle ID (required)")
	fs.StringVar(&opts.ClaimID, "claim-id", "", "claim ID (required)")
	fs.StringVar(&opts.EvidenceID, "evidence-id", "", "evidence ID (required)")
	fs.BoolVar(&opts.JSON, "json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if opts.RunID == "" {
		fmt.Fprintln(os.Stderr, "ERROR: attach-evidence requires --run-id")
		fs.Usage()
		return 1
	}
	if opts.ClaimID == "" {
		fmt.Fprintln(os.Stderr, "ERROR: attach-evidence requires --claim-id")
		fs.Usage()
		return 1
	}
	if opts.EvidenceID == "" {
		fmt.Fprintln(os.Stderr, "ERROR: attach-evidence requires --evidence-id")
		fs.Usage()
		return 1
	}
	if opts.Root == "" {
		fmt.Fprintln(os.Stderr, "ERROR: run bundle root must be non-empty")
		return 1
	}

	runID := runbundle.RunID(opts.RunID)
	if err := runbundle.ValidateRunID(runID); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid run ID: %v\n", err)
		return 1
	}

	clmID := claim.ClaimID(opts.ClaimID)
	if err := claim.ValidateClaimID(clmID); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid claim ID: %v\n", err)
		return 1
	}

	evID := claim.EvidenceID(opts.EvidenceID)
	if err := claim.ValidateEvidenceID(evID); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid evidence ID: %v\n", err)
		return 1
	}

	bundle, _, err := runbundle.Open(opts.Root, runID)
	if err != nil {
		printRunBundleError(opts.Root, runID, err)
		return 1
	}

	store := claim.NewStore(bundle)

	// Verify evidence exists
	_, err = store.ReadEvidence(evID)
	if err != nil {
		if errors.Is(err, claim.ErrEvidenceNotFound) {
			fmt.Fprintf(os.Stderr, "ERROR: evidence not found: %s\n", evID)
		} else {
			fmt.Fprintf(os.Stderr, "ERROR: failed to read evidence: %v\n", err)
		}
		return 1
	}

	// Check if claim exists and if evidence is already attached
	existingClaim, err := store.ReadClaim(clmID)
	if err != nil {
		if errors.Is(err, claim.ErrClaimNotFound) {
			fmt.Fprintf(os.Stderr, "ERROR: claim not found: %s\n", clmID)
		} else {
			fmt.Fprintf(os.Stderr, "ERROR: failed to read claim: %v\n", err)
		}
		return 1
	}

	alreadyAttached := existingClaim.HasEvidence(evID)

	// Attach evidence (idempotent)
	updatedClaim, err := store.AddEvidenceToClaim(clmID, evID, time.Now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to attach evidence: %v\n", err)
		return 1
	}

	newlyAttached := !alreadyAttached

	if opts.JSON {
		output := struct {
			OK         bool   `json:"ok"`
			RunID      string `json:"run_id"`
			ClaimID    string `json:"claim_id"`
			EvidenceID string `json:"evidence_id"`
			Attached   bool   `json:"attached"`
		}{
			OK:         true,
			RunID:      string(runID),
			ClaimID:    string(clmID),
			EvidenceID: string(evID),
			Attached:   newlyAttached,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		if newlyAttached {
			fmt.Printf("attached evidence %s to claim %s\n", evID, clmID)
			fmt.Printf("run bundle: %s\n", runID)
		} else {
			fmt.Printf("evidence %s already attached to claim %s\n", evID, clmID)
		}
		_ = updatedClaim
	}

	return 0
}
