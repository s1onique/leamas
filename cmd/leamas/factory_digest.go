// Package main provides the factory digest command handler.
package main

import (
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/factory/digest"
)

func handleFactoryDigest() {
	var mode digest.Mode
	var hasDirty, hasStaged, hasRange bool
	var output string
	var rangeSpec string

	args := os.Args[3:]
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--dirty":
			hasDirty = true
			mode = digest.ModeDirty
			i++
		case "--staged":
			hasStaged = true
			mode = digest.ModeStaged
			i++
		case "--range":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "ERROR: --range requires a revision range argument\n")
				printDigestUsage()
				os.Exit(1)
			}
			hasRange = true
			rangeSpec = args[i+1]
			mode = digest.ModeRange
			i += 2
		case "--output":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "ERROR: --output requires a path argument\n")
				printDigestUsage()
				os.Exit(1)
			}
			output = args[i+1]
			i += 2
		default:
			fmt.Fprintf(os.Stderr, "ERROR: unknown flag: %s\n", args[i])
			printDigestUsage()
			os.Exit(1)
		}
	}

	if hasDirty && hasStaged {
		fmt.Fprintf(os.Stderr, "ERROR: cannot specify both --dirty and --staged\n")
		printDigestUsage()
		os.Exit(1)
	}
	if hasDirty && hasRange {
		fmt.Fprintf(os.Stderr, "ERROR: cannot specify both --dirty and --range\n")
		printDigestUsage()
		os.Exit(1)
	}
	if hasStaged && hasRange {
		fmt.Fprintf(os.Stderr, "ERROR: cannot specify both --staged and --range\n")
		printDigestUsage()
		os.Exit(1)
	}

	if mode == "" {
		mode = digest.ModeAuto
	}

	if output == "" {
		fmt.Fprintf(os.Stderr, "ERROR: --output is required\n")
		printDigestUsage()
		os.Exit(1)
	}

	opts := digest.Options{
		Mode:   mode,
		Output: output,
	}
	if hasRange {
		opts.Range = rangeSpec
	}

	err := digest.Write(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(output)
}

func printDigestUsage() {
	fmt.Println("Usage: leamas factory digest [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --dirty             Include unstaged, staged, and untracked changes")
	fmt.Println("  --staged            Include only staged changes")
	fmt.Println("  --range <rev-range> Include changes in revision range")
	fmt.Println("  --output <path>     Output path (required)")
}
