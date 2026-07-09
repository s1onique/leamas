// Package main provides the factory digest command handler.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/s1onique/leamas/internal/factory/digest"
)

// digestArgs holds parsed arguments for the digest command.
type digestArgs struct {
	mode      digest.Mode
	hasDirty  bool
	hasStaged bool
	hasRange  bool
	output    string
	rangeSpec string
}

// parseDigestArgs parses command-line arguments for the digest command.
func parseDigestArgs(args []string) (digestArgs, error) {
	result := digestArgs{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dirty":
			result.hasDirty = true
			result.mode = digest.ModeDirty
		case "--staged":
			result.hasStaged = true
			result.mode = digest.ModeStaged
		case "--range":
			if i+1 >= len(args) {
				return digestArgs{}, errors.New("--range requires a revision range argument")
			}
			if strings.HasPrefix(args[i+1], "-") {
				return digestArgs{}, errors.New("--range requires a revision range argument")
			}
			result.hasRange = true
			result.rangeSpec = args[i+1]
			result.mode = digest.ModeRange
			i++
		case "--output":
			if i+1 >= len(args) {
				return digestArgs{}, errors.New("--output requires a path argument")
			}
			result.output = args[i+1]
			i++
		default:
			return digestArgs{}, fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	// Validate mutual exclusivity
	if result.hasDirty && result.hasStaged {
		return digestArgs{}, errors.New("cannot specify both --dirty and --staged")
	}
	if result.hasDirty && result.hasRange {
		return digestArgs{}, errors.New("cannot specify both --dirty and --range")
	}
	if result.hasStaged && result.hasRange {
		return digestArgs{}, errors.New("cannot specify both --staged and --range")
	}

	// Default to auto mode if no mode specified
	if result.mode == "" {
		result.mode = digest.ModeAuto
	}

	// Validate required --output
	if result.output == "" {
		return digestArgs{}, errors.New("--output is required")
	}

	return result, nil
}

// runFactoryDigest runs the digest command with the given arguments.
// It returns 0 on success, non-zero on failure.
func runFactoryDigest(args []string, stdout, stderr io.Writer, writeDigest func(digest.Options) error) int {
	parsed, err := parseDigestArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %s\n", err)
		printDigestUsageTo(stderr)
		return 1
	}

	opts := digest.Options{
		Mode:   parsed.mode,
		Output: parsed.output,
	}
	if parsed.hasRange {
		opts.Range = parsed.rangeSpec
	}

	if err := writeDigest(opts); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, opts.Output)
	return 0
}

// handleFactoryDigest handles the `leamas factory digest` command.
func handleFactoryDigest() {
	os.Exit(runFactoryDigest(os.Args[3:], os.Stdout, os.Stderr, digest.Write))
}

// printDigestUsage prints usage information to stdout.
func printDigestUsage() {
	printDigestUsageTo(os.Stdout)
}

// printDigestUsageTo prints usage information to the given writer.
func printDigestUsageTo(w io.Writer) {
	fmt.Fprintln(w, "Usage: leamas factory digest [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --dirty             Include unstaged, staged, and untracked changes")
	fmt.Fprintln(w, "  --staged            Include only staged changes")
	fmt.Fprintln(w, "  --range <rev-range> Include changes in revision range")
	fmt.Fprintln(w, "  --output <path>     Output path (required)")
}
