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

// evidenceShowOptions is the dependency-injection contract used by the
// witness dispatcher's ShowEvidence hook (see witness.go). It is not
// used by the production runWitnessEvidenceShow path; that path is
// driven by runRecordShow in record_show.go. Keeping the type preserves
// the existing public dependency surface while letting the production
// path share its orchestration.
type evidenceShowOptions struct {
	Root  string
	RunID string
	JSON  bool
}

// ============================================================================
// Evidence create
// ============================================================================

type evidenceCreateOptions struct {
	Root         string
	RunID        string
	ID           string
	Kind         string
	Role         string
	Title        string
	RelativePath string
	Summary      string
	JSON         bool
}

func runWitnessEvidenceCreate(args []string) int {
	var opts evidenceCreateOptions
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.Root, "root", defaultRunBundleRoot, "root directory for run bundles")
	fs.StringVar(&opts.RunID, "run-id", "", "run bundle ID (required)")
	fs.StringVar(&opts.ID, "id", "", "evidence ID (required)")
	fs.StringVar(&opts.Kind, "kind", "", "evidence kind (required)")
	fs.StringVar(&opts.Role, "role", "", "evidence role (required)")
	fs.StringVar(&opts.Title, "title", "", "evidence title (required)")
	fs.StringVar(&opts.RelativePath, "relative-path", "", "relative path to evidence artifact")
	fs.StringVar(&opts.Summary, "summary", "", "evidence summary")
	fs.BoolVar(&opts.JSON, "json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if opts.RunID == "" {
		fmt.Fprintln(os.Stderr, "ERROR: evidence create requires --run-id")
		fs.Usage()
		return 1
	}
	if opts.ID == "" {
		fmt.Fprintln(os.Stderr, "ERROR: evidence create requires --id")
		fs.Usage()
		return 1
	}
	if opts.Kind == "" {
		fmt.Fprintln(os.Stderr, "ERROR: evidence create requires --kind")
		fs.Usage()
		return 1
	}
	if opts.Role == "" {
		fmt.Fprintln(os.Stderr, "ERROR: evidence create requires --role")
		fs.Usage()
		return 1
	}
	if opts.Title == "" {
		fmt.Fprintln(os.Stderr, "ERROR: evidence create requires --title")
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

	evidenceID := claim.EvidenceID(opts.ID)
	if err := claim.ValidateEvidenceID(evidenceID); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid evidence ID: %v\n", err)
		return 1
	}

	kind := claim.EvidenceKind(opts.Kind)
	if !claim.IsValidEvidenceKind(kind) {
		validKinds := []string{
			"command_output", "digest", "log", "file", "trace", "verifier_result",
		}
		fmt.Fprintf(os.Stderr, "ERROR: invalid evidence kind: %s\n", opts.Kind)
		fmt.Fprintf(os.Stderr, "Valid kinds: %s\n", strings.Join(validKinds, ", "))
		return 1
	}

	role := claim.EvidenceRole(opts.Role)
	if !claim.IsValidEvidenceRole(role) {
		validRoles := []string{"primary", "supporting", "contradicting", "context"}
		fmt.Fprintf(os.Stderr, "ERROR: invalid evidence role: %s\n", opts.Role)
		fmt.Fprintf(os.Stderr, "Valid roles: %s\n", strings.Join(validRoles, ", "))
		return 1
	}

	if opts.RelativePath != "" {
		if err := claim.ValidateRelativePath(opts.RelativePath); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: invalid relative path: %v\n", err)
			return 1
		}
	}

	bundle, _, err := runbundle.Open(opts.Root, runID)
	if err != nil {
		printRunBundleError(opts.Root, runID, err)
		return 1
	}

	store := claim.NewStore(bundle)
	ev, err := claim.NewEvidence(evidenceID, runID, kind, role, opts.Title, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to create evidence: %v\n", err)
		return 1
	}

	if opts.RelativePath != "" {
		ev.RelativePath = opts.RelativePath
	}
	if opts.Summary != "" {
		ev.Summary = opts.Summary
	}

	if err := store.WriteEvidence(ev); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to write evidence: %v\n", err)
		return 1
	}

	if opts.JSON {
		output := struct {
			OK         bool   `json:"ok"`
			RunID      string `json:"run_id"`
			EvidenceID string `json:"evidence_id"`
			Path       string `json:"path"`
		}{
			OK:         true,
			RunID:      string(runID),
			EvidenceID: string(evidenceID),
			Path:       filepath.Join(bundle.Path, "evidence", string(evidenceID)+".json"),
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("created evidence: %s\n", evidenceID)
		fmt.Printf("run bundle: %s\n", runID)
		fmt.Printf("path: %s\n", filepath.Join(bundle.Path, "evidence", string(evidenceID)+".json"))
	}

	return 0
}

// ============================================================================
// Evidence list
// ============================================================================

type evidenceListOptions struct {
	Root  string
	RunID string
	JSON  bool
}

func runWitnessEvidenceList(args []string) int {
	var opts evidenceListOptions
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.Root, "root", defaultRunBundleRoot, "root directory for run bundles")
	fs.StringVar(&opts.RunID, "run-id", "", "run bundle ID (required)")
	fs.BoolVar(&opts.JSON, "json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if opts.RunID == "" {
		fmt.Fprintln(os.Stderr, "ERROR: evidence list requires --run-id")
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
	evidenceList, err := listEvidence(store)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to list evidence: %v\n", err)
		return 1
	}

	if opts.JSON {
		type evidenceSummary struct {
			ID    string `json:"id"`
			Kind  string `json:"kind"`
			Role  string `json:"role"`
			Title string `json:"title"`
		}
		arr := make([]evidenceSummary, 0, len(evidenceList))
		for _, e := range evidenceList {
			arr = append(arr, evidenceSummary{
				ID:    string(e.ID),
				Kind:  string(e.Kind),
				Role:  string(e.Role),
				Title: e.Title,
			})
		}
		output := struct {
			OK       bool              `json:"ok"`
			Root     string            `json:"root"`
			RunID    string            `json:"run_id"`
			Evidence []evidenceSummary `json:"evidence"`
		}{
			OK:       true,
			Root:     opts.Root,
			RunID:    string(runID),
			Evidence: arr,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		if len(evidenceList) == 0 {
			fmt.Println("no evidence found")
			return 0
		}
		for _, e := range evidenceList {
			fmt.Printf("%s  %s  %s  %s\n", e.ID, e.Kind, e.Role, e.Title)
		}
	}

	return 0
}

func listEvidence(store claim.Store) ([]claim.Evidence, error) {
	evidenceDir := filepath.Join(store.Bundle.Path, "evidence")

	info, err := os.Stat(evidenceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(evidenceDir)
	if err != nil {
		return nil, err
	}

	var evidenceList []claim.Evidence
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		idStr := strings.TrimSuffix(name, ".json")
		evidenceID := claim.EvidenceID(idStr)

		e, err := store.ReadEvidence(evidenceID)
		if err != nil {
			continue
		}
		evidenceList = append(evidenceList, e)
	}

	return evidenceList, nil
}

// ============================================================================
// Evidence show
// ============================================================================

// runWitnessEvidenceShow implements `leamas witness evidence show <evidence-id>`.
// The shared orchestration (flag parsing, arg/run-id/root validation,
// run bundle opening, error formatting, exit-code selection, JSON/text
// dispatch) lives in runRecordShow in record_show.go. This function
// only owns the evidence-specific policy decisions.
func runWitnessEvidenceShow(args []string) int {
	return runRecordShow(args, recordShowSpec{
		KindName:    "evidence",
		PosArgLabel: "<evidence-id>",
		ValidateID: func(rawID string) error {
			return claim.ValidateEvidenceID(claim.EvidenceID(rawID))
		},
		NotFoundErr: claim.ErrEvidenceNotFound,
		ReadRecord: func(store claim.Store, rawID string) (any, error) {
			ev, err := store.ReadEvidence(claim.EvidenceID(rawID))
			if err != nil {
				return nil, err
			}
			return &ev, nil
		},
		RenderText: renderEvidenceRecordText,
		RenderJSON: renderEvidenceRecordJSON,
	})
}
