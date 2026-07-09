package main

import (
	"fmt"
	"io"
	"os"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// ============================================================================
// Claim dispatcher
// ============================================================================

func runWitnessClaim(args []string) int {
	if len(args) < 1 {
		printClaimUsage()
		return 1
	}

	switch args[0] {
	case "create":
		return runWitnessClaimCreate(args[1:])
	case "list":
		return runWitnessClaimList(args[1:])
	case "show":
		return runWitnessClaimShow(args[1:])
	case "attach-evidence":
		return runWitnessClaimAttachEvidence(args[1:])
	case "--help", "-h":
		printClaimUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown claim subcommand: %s\n", args[0])
		printClaimUsage()
		return 1
	}
}

func printClaimUsage() {
	printClaimUsageTo(os.Stderr)
}

func printClaimUsageTo(w io.Writer) {
	fmt.Fprintln(w, "Usage: leamas witness claim <subcommand> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  create                   Create a new claim")
	fmt.Fprintln(w, "  list                     List claims in a run bundle")
	fmt.Fprintln(w, "  show <claim-id>          Show a claim")
	fmt.Fprintln(w, "  attach-evidence          Attach evidence to a claim")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --root <path>    Root directory for run bundles (default: .leamas/runs)")
	fmt.Fprintln(w, "  --run-id <id>    Run bundle ID (required)")
	fmt.Fprintln(w, "  --json           Output JSON format")
	fmt.Fprintln(w, "  --help, -h       Show this help")
}

// ============================================================================
// Evidence dispatcher
// ============================================================================

func runWitnessEvidence(args []string) int {
	if len(args) < 1 {
		printEvidenceUsage()
		return 1
	}

	switch args[0] {
	case "create":
		return runWitnessEvidenceCreate(args[1:])
	case "list":
		return runWitnessEvidenceList(args[1:])
	case "show":
		return runWitnessEvidenceShow(args[1:])
	case "--help", "-h":
		printEvidenceUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown evidence subcommand: %s\n", args[0])
		printEvidenceUsage()
		return 1
	}
}

func printEvidenceUsage() {
	printEvidenceUsageTo(os.Stderr)
}

func printEvidenceUsageTo(w io.Writer) {
	fmt.Fprintln(w, "Usage: leamas witness evidence <subcommand> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  create              Create new evidence")
	fmt.Fprintln(w, "  list                List evidence in a run bundle")
	fmt.Fprintln(w, "  show <evidence-id>  Show evidence")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --root <path>    Root directory for run bundles (default: .leamas/runs)")
	fmt.Fprintln(w, "  --run-id <id>    Run bundle ID (required)")
	fmt.Fprintln(w, "  --json            Output JSON format")
	fmt.Fprintln(w, "  --help, -h        Show this help")
}

// ============================================================================
// Helper
// ============================================================================

func printRunBundleError(root string, runID runbundle.RunID, err error) {
	if err == runbundle.ErrMissingMetadata {
		fmt.Fprintf(os.Stderr, "ERROR: run bundle not found: %s\n", runID)
	} else if err == runbundle.ErrSchemaVersionMismatch {
		fmt.Fprintln(os.Stderr, "ERROR: run bundle metadata schema version mismatch")
	} else if err == runbundle.ErrRunIDMismatch {
		fmt.Fprintln(os.Stderr, "ERROR: run bundle metadata run ID does not match requested ID")
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	}
}
