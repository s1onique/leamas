package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/witness/claim"
	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// claimShowOptions is the dependency-injection contract used by the
// witness dispatcher's ShowClaim hook (see witness.go). It is not used
// by the production runWitnessClaimShow path; that path is driven by
// runRecordShow in record_show.go. Keeping the type preserves the
// existing public dependency surface while letting the production path
// share its orchestration.
type claimShowOptions struct {
	Root  string
	RunID string
	JSON  bool
}

// ============================================================================
// Claim create
// ============================================================================

type claimCreateOptions struct {
	Root      string
	RunID     string
	ID        string
	Statement string
	Notes     string
	JSON      bool
}

func runWitnessClaimCreate(args []string) int {
	var opts claimCreateOptions
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.Root, "root", defaultRunBundleRoot, "root directory for run bundles")
	fs.StringVar(&opts.RunID, "run-id", "", "run bundle ID (required)")
	fs.StringVar(&opts.ID, "id", "", "claim ID (required)")
	fs.StringVar(&opts.Statement, "statement", "", "claim statement (required)")
	fs.StringVar(&opts.Notes, "notes", "", "claim notes")
	fs.BoolVar(&opts.JSON, "json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if opts.RunID == "" {
		fmt.Fprintln(os.Stderr, "ERROR: claim create requires --run-id")
		fs.Usage()
		return 1
	}
	if opts.ID == "" {
		fmt.Fprintln(os.Stderr, "ERROR: claim create requires --id")
		fs.Usage()
		return 1
	}
	if opts.Statement == "" {
		fmt.Fprintln(os.Stderr, "ERROR: claim create requires --statement")
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

	claimID := claim.ClaimID(opts.ID)
	if err := claim.ValidateClaimID(claimID); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid claim ID: %v\n", err)
		return 1
	}

	bundle, _, err := runbundle.Open(opts.Root, runID)
	if err != nil {
		printRunBundleError(opts.Root, runID, err)
		return 1
	}

	store := claim.NewStore(bundle)
	clm, err := claim.NewClaim(claimID, runID, opts.Statement, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to create claim: %v\n", err)
		return 1
	}

	if opts.Notes != "" {
		clm.Notes = opts.Notes
	}

	if err := store.WriteClaim(clm); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to write claim: %v\n", err)
		return 1
	}

	if opts.JSON {
		output := struct {
			OK      bool   `json:"ok"`
			RunID   string `json:"run_id"`
			ClaimID string `json:"claim_id"`
			Path    string `json:"path"`
		}{
			OK:      true,
			RunID:   string(runID),
			ClaimID: string(claimID),
			Path:    filepath.Join(bundle.Path, "claims", string(claimID)+".json"),
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("created claim: %s\n", claimID)
		fmt.Printf("run bundle: %s\n", runID)
		fmt.Printf("path: %s\n", filepath.Join(bundle.Path, "claims", string(claimID)+".json"))
	}

	return 0
}

// ============================================================================
// Claim list
// ============================================================================

type claimListOptions struct {
	Root  string
	RunID string
	JSON  bool
}

func runWitnessClaimList(args []string) int {
	var opts claimListOptions
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.Root, "root", defaultRunBundleRoot, "root directory for run bundles")
	fs.StringVar(&opts.RunID, "run-id", "", "run bundle ID (required)")
	fs.BoolVar(&opts.JSON, "json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if opts.RunID == "" {
		fmt.Fprintln(os.Stderr, "ERROR: claim list requires --run-id")
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

	bundle, _, err := runbundle.Open(opts.Root, runID)
	if err != nil {
		printRunBundleError(opts.Root, runID, err)
		return 1
	}

	store := claim.NewStore(bundle)
	claims, err := listClaims(store)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to list claims: %v\n", err)
		return 1
	}

	if opts.JSON {
		type claimSummary struct {
			ID            string `json:"id"`
			Status        string `json:"status"`
			Verdict       string `json:"verdict"`
			Statement     string `json:"statement"`
			EvidenceCount int    `json:"evidence_count"`
		}
		arr := make([]claimSummary, 0, len(claims))
		for _, c := range claims {
			arr = append(arr, claimSummary{
				ID:            string(c.ID),
				Status:        string(c.Status),
				Verdict:       string(c.Verdict),
				Statement:     c.Statement,
				EvidenceCount: len(c.EvidenceIDs),
			})
		}
		output := struct {
			OK     bool           `json:"ok"`
			Root   string         `json:"root"`
			RunID  string         `json:"run_id"`
			Claims []claimSummary `json:"claims"`
		}{
			OK:     true,
			Root:   opts.Root,
			RunID:  string(runID),
			Claims: arr,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		if len(claims) == 0 {
			fmt.Println("no claims found")
			return 0
		}
		for _, c := range claims {
			fmt.Printf("%s  %s  %s  %s\n", c.ID, c.Status, c.Verdict, c.Statement)
		}
	}

	return 0
}

func listClaims(store claim.Store) ([]claim.Claim, error) {
	claimsDir := filepath.Join(store.Bundle.Path, "claims")

	info, err := os.Stat(claimsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(claimsDir)
	if err != nil {
		return nil, err
	}

	var claims []claim.Claim
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		idStr := strings.TrimSuffix(name, ".json")
		claimID := claim.ClaimID(idStr)

		c, err := store.ReadClaim(claimID)
		if err != nil {
			continue
		}
		claims = append(claims, c)
	}

	return claims, nil
}

// ============================================================================
// Claim show
// ============================================================================

// runWitnessClaimShow implements `leamas witness claim show <claim-id>`.
// The shared orchestration (flag parsing, arg/run-id/root validation,
// run bundle opening, error formatting, exit-code selection, JSON/text
// dispatch) lives in runRecordShow in record_show.go. This function
// only owns the claim-specific policy decisions.
func runWitnessClaimShow(args []string) int {
	return runRecordShow(args, recordShowSpec{
		KindName:    "claim",
		PosArgLabel: "<claim-id>",
		ValidateID: func(rawID string) error {
			return claim.ValidateClaimID(claim.ClaimID(rawID))
		},
		NotFoundErr: claim.ErrClaimNotFound,
		ReadRecord: func(store claim.Store, rawID string) (any, error) {
			clm, err := store.ReadClaim(claim.ClaimID(rawID))
			if err != nil {
				return nil, err
			}
			return &clm, nil
		},
		RenderText: renderClaimRecordText,
		RenderJSON: renderClaimRecordJSON,
	})
}
