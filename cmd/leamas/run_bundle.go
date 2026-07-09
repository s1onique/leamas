// Package main provides the Leamas CLI.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

const defaultRunBundleRoot = ".leamas/runs"

// runBundleCreateOptions holds options for run bundle create command.
type runBundleCreateOptions struct {
	Root        string
	ID          string
	ToolVersion string
	JSON        bool
}

// runBundleListOptions holds options for run bundle list command.
type runBundleListOptions struct {
	Root           string
	JSON           bool
	IncludeInvalid bool
}

// runBundleShowOptions holds options for run bundle show command.
type runBundleShowOptions struct {
	Root string
	ID   string
	JSON bool
}

// runWitnessRunBundle is a testable dispatcher for run-bundle subcommands.
func runWitnessRunBundle(args []string) int {
	if len(args) < 1 {
		printRunBundleUsage()
		return 1
	}

	switch args[0] {
	case "create":
		return runWitnessRunBundleCreate(args[1:])
	case "list":
		return runWitnessRunBundleList(args[1:])
	case "show":
		return runWitnessRunBundleShow(args[1:])
	case "--help", "-h":
		printRunBundleUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown run-bundle subcommand: %s\n", args[0])
		printRunBundleUsage()
		return 1
	}
}

// handleWitnessRunBundle dispatches run-bundle subcommands from os.Args.
func handleWitnessRunBundle() {
	if len(os.Args) < 4 {
		printRunBundleUsage()
		os.Exit(1)
	}
	os.Exit(runWitnessRunBundle(os.Args[3:]))
}

func printRunBundleUsage() {
	fmt.Fprintln(os.Stderr, "Usage: leamas witness run-bundle <subcommand> [flags]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Subcommands:")
	fmt.Fprintln(os.Stderr, "  create         Create a new run bundle")
	fmt.Fprintln(os.Stderr, "  list           List run bundles")
	fmt.Fprintln(os.Stderr, "  show <run-id>  Show run bundle details")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --root <path>  Root directory for run bundles (default: .leamas/runs)")
	fmt.Fprintln(os.Stderr, "  --json         Output JSON format")
	fmt.Fprintln(os.Stderr, "  --help, -h     Show this help")
}

// runWitnessRunBundleCreate handles the create subcommand.
func runWitnessRunBundleCreate(args []string) int {
	var opts runBundleCreateOptions
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.Root, "root", defaultRunBundleRoot, "root directory for run bundles")
	fs.StringVar(&opts.ID, "id", "", "run ID (required)")
	fs.StringVar(&opts.ToolVersion, "tool-version", "", "tool version string")
	fs.BoolVar(&opts.JSON, "json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// Validate required --id
	if opts.ID == "" {
		fmt.Fprintln(os.Stderr, "ERROR: run bundle create requires --id")
		fs.Usage()
		return 1
	}

	// Validate root is non-empty
	if opts.Root == "" {
		fmt.Fprintln(os.Stderr, "ERROR: run bundle root must be non-empty")
		return 1
	}

	// Validate run ID
	runID := runbundle.RunID(opts.ID)
	if err := runbundle.ValidateRunID(runID); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid run ID: %v\n", err)
		return 1
	}

	// Create the bundle
	bundle, err := runbundle.Create(runbundle.CreateOptions{
		Root:     opts.Root,
		RunID:    runID,
		ToolName: "leamas",
		Version:  opts.ToolVersion,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to create run bundle: %v\n", err)
		return 1
	}

	// Output result
	if opts.JSON {
		output := struct {
			OK       bool   `json:"ok"`
			RunID    string `json:"run_id"`
			Path     string `json:"path"`
			Metadata struct {
				SchemaVersion string `json:"schema_version"`
			} `json:"metadata"`
		}{
			OK:    true,
			RunID: string(runID),
			Path:  bundle.Path,
		}
		output.Metadata.SchemaVersion = runbundle.SchemaVersion
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("created run bundle: %s\n", bundle.Path)
	}

	return 0
}

// bundleInfo holds parsed bundle metadata for output formatting.
type bundleInfo struct {
	runID     string
	createdAt string
	path      string
	schemaVer string
	ok        bool
}

// bundleJSON is the JSON output format for a single bundle.
type bundleJSON struct {
	RunID         string `json:"run_id,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	Path          string `json:"path"`
	SchemaVersion string `json:"schema_version,omitempty"`
}

// runWitnessRunBundleList handles the list subcommand.
func runWitnessRunBundleList(args []string) int {
	var opts runBundleListOptions
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.Root, "root", defaultRunBundleRoot, "root directory for run bundles")
	fs.BoolVar(&opts.JSON, "json", false, "output JSON")
	fs.BoolVar(&opts.IncludeInvalid, "include-invalid", false, "include invalid bundles")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// Validate root is non-empty
	if opts.Root == "" {
		fmt.Fprintln(os.Stderr, "ERROR: run bundle root must be non-empty")
		return 1
	}

	// Check if root exists
	info, err := os.Stat(opts.Root)
	if err != nil {
		if os.IsNotExist(err) {
			// Root doesn't exist - print empty list
			printRunBundleListJSON(opts.Root, nil)
			return 0
		}
		fmt.Fprintf(os.Stderr, "ERROR: cannot access root directory: %v\n", err)
		return 1
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "ERROR: root is not a directory: %s\n", opts.Root)
		return 1
	}

	// Read directory entries
	entries, err := os.ReadDir(opts.Root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot read root directory: %v\n", err)
		return 1
	}

	var bundles []bundleInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		candidateID := runbundle.RunID(entry.Name())

		// Try to open as a valid bundle
		_, meta, err := runbundle.Open(opts.Root, candidateID)
		if err != nil {
			if !opts.IncludeInvalid {
				continue // skip invalid bundles
			}
			// Include invalid bundles with partial info
			bundles = append(bundles, bundleInfo{
				runID: entry.Name(),
				path:  filepath.Join(opts.Root, entry.Name()),
				ok:    false,
			})
			continue
		}

		bundles = append(bundles, bundleInfo{
			runID:     string(meta.RunID),
			createdAt: meta.CreatedAt.Format(time.RFC3339),
			path:      filepath.Join(opts.Root, entry.Name()),
			schemaVer: meta.SchemaVersion,
			ok:        true,
		})
	}

	// Output result
	if opts.JSON {
		arr := make([]bundleJSON, 0, len(bundles))
		for _, b := range bundles {
			arr = append(arr, bundleJSON{
				RunID:         b.runID,
				CreatedAt:     b.createdAt,
				Path:          b.path,
				SchemaVersion: b.schemaVer,
			})
		}
		printRunBundleListJSON(opts.Root, arr)
	} else {
		if len(bundles) == 0 {
			fmt.Println("no run bundles found")
			return 0
		}
		for _, b := range bundles {
			if b.ok {
				fmt.Printf("%s  %s  %s\n", b.runID, b.createdAt, b.path)
			} else {
				fmt.Printf("%s  <invalid>  %s\n", b.runID, b.path)
			}
		}
	}

	return 0
}

// printRunBundleListJSON prints JSON output for list command.
func printRunBundleListJSON(root string, bundles []bundleJSON) {
	output := struct {
		OK      bool         `json:"ok"`
		Root    string       `json:"root"`
		Bundles []bundleJSON `json:"bundles"`
	}{
		OK:      true,
		Root:    root,
		Bundles: bundles,
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to marshal JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// runWitnessRunBundleShow handles the show subcommand.
func runWitnessRunBundleShow(args []string) int {
	var opts runBundleShowOptions
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.Root, "root", defaultRunBundleRoot, "root directory for run bundles")
	fs.BoolVar(&opts.JSON, "json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// Require exactly one positional argument (run-id)
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "ERROR: run bundle show requires <run-id>")
		fs.Usage()
		return 1
	}

	runID := runbundle.RunID(fs.Arg(0))

	// Validate run ID
	if err := runbundle.ValidateRunID(runID); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid run ID: %v\n", err)
		return 1
	}

	// Validate root is non-empty
	if opts.Root == "" {
		fmt.Fprintln(os.Stderr, "ERROR: run bundle root must be non-empty")
		return 1
	}

	// Open the bundle
	bundle, meta, err := runbundle.Open(opts.Root, runID)
	if err != nil {
		// Provide clear error messages for specific cases
		if errors.Is(err, runbundle.ErrMissingMetadata) {
			fmt.Fprintf(os.Stderr, "ERROR: metadata.json not found in %s\n", filepath.Join(opts.Root, string(runID)))
		} else if errors.Is(err, runbundle.ErrSchemaVersionMismatch) {
			fmt.Fprintln(os.Stderr, "ERROR: metadata schema version mismatch")
		} else if errors.Is(err, runbundle.ErrRunIDMismatch) {
			fmt.Fprintln(os.Stderr, "ERROR: metadata run ID does not match requested ID")
		} else {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		}
		return 1
	}

	// Output result
	if opts.JSON {
		output := struct {
			OK       bool                `json:"ok"`
			Path     string              `json:"path"`
			Metadata *runbundle.Metadata `json:"metadata"`
		}{
			OK:       true,
			Path:     bundle.Path,
			Metadata: meta,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		// Text format matching the ACT spec
		fmt.Printf("Run bundle: %s\n", meta.RunID)
		fmt.Printf("Path: %s\n", bundle.Path)
		fmt.Printf("Created: %s\n", meta.CreatedAt.Format(time.RFC3339))
		fmt.Printf("Schema: %s\n", meta.SchemaVersion)
		fmt.Printf("Tool: leamas %s\n", meta.Tool.Version)

		// Build doctrine string
		var doctrineParts []string
		if meta.Doctrine.LocalOnly {
			doctrineParts = append(doctrineParts, "local_only=true")
		}
		if meta.Doctrine.ReadOnly {
			doctrineParts = append(doctrineParts, "read_only=true")
		}
		if meta.Doctrine.NoDatabase {
			doctrineParts = append(doctrineParts, "no_database=true")
		}
		fmt.Printf("Doctrine: %s\n", strings.Join(doctrineParts, " "))
	}

	return 0
}
