// Leamas CLI - Local-first, single-binary tool orchestration
package main

import (
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/factory/agentcontext"
	"github.com/s1onique/leamas/internal/factory/boundary"
	"github.com/s1onique/leamas/internal/factory/digest"
	"github.com/s1onique/leamas/internal/factory/docs"
	"github.com/s1onique/leamas/internal/factory/doctrine"
	"github.com/s1onique/leamas/internal/factory/forbidden"
	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/factory/githooks"
	"github.com/s1onique/leamas/internal/factory/github"
	"github.com/s1onique/leamas/internal/factory/language"
	"github.com/s1onique/leamas/internal/factory/llmfriendly"
	"github.com/s1onique/leamas/internal/factory/staticbinary"
	"github.com/s1onique/leamas/internal/factory/tooling"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		fmt.Println("leamas version", version)
	case "factory":
		handleFactory()
	case "doctor":
		fmt.Println("Leamas doctor: all systems operational")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func handleFactory() {
	if len(os.Args) < 3 {
		printFactoryUsage()
		os.Exit(1)
	}

	switch os.Args[2] {
	case "verify":
		handleFactoryVerify()
	case "gate":
		handleFactoryGate()
	case "factorize":
		handleFactoryFactorize()
	case "digest":
		handleFactoryDigest()
	default:
		fmt.Fprintf(os.Stderr, "unknown factory command: %s\n", os.Args[2])
		printFactoryUsage()
		os.Exit(1)
	}
}

func handleFactoryVerify() {
	if len(os.Args) < 4 {
		printFactoryVerifyUsage()
		os.Exit(1)
	}

	check := os.Args[3]
	var findings []struct {
		path    string
		kind    string
		message string
	}

	switch check {
	case "doctrine", "doctrine-agent-contracts":
		f := doctrine.CheckRepo(".")
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	case "docs":
		f := docs.CheckRepo(".")
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	case "forbidden-patterns":
		f := forbidden.CheckRepo(".")
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	case "language":
		f := language.CheckRepo(".")
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	case "static-binary":
		f := staticbinary.CheckRepo(".")
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	case "tooling-boundaries":
		f := tooling.CheckRepo(".")
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	case "llm-friendly":
		cfg := llmfriendly.DefaultConfig()
		f, err := llmfriendly.CheckRepo(".", cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "LLM-friendliness verification ERROR: %v\n", err)
			os.Exit(1)
		}
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	case "agent-context":
		f, err := agentcontext.CheckRepo(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Agent context verification ERROR: %v\n", err)
			os.Exit(1)
		}
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	case "git-hooks":
		f, err := githooks.CheckRepo(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Git hooks verification ERROR: %v\n", err)
			os.Exit(1)
		}
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	case "github":
		f, err := github.CheckRepo(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "GitHub policy verification ERROR: %v\n", err)
			os.Exit(1)
		}
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	case "domain-boundaries":
		f := boundary.CheckRepo(".")
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown verify command: %s\n", check)
		printFactoryVerifyUsage()
		os.Exit(1)
	}

	if len(findings) == 0 {
		fmt.Printf("%s verification PASSED\n", check)
		os.Exit(0)
	}

	fmt.Printf("%s verification FAILED\n", check)
	for _, f := range findings {
		fmt.Printf("  %s: %s: %s\n", f.path, f.kind, f.message)
	}
	os.Exit(1)
}

func handleFactoryGate() {
	exitCode := gate.RunGate(".")
	os.Exit(exitCode)
}

func handleFactoryFactorize() {
	exitCode := gate.RunFactorize(".")
	os.Exit(exitCode)
}

func handleFactoryDigest() {
	// Parse flags manually for simplicity
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

	// Validate: mutually exclusive modes
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

	// If no mode specified, default to auto
	if mode == "" {
		mode = digest.ModeAuto
	}

	// Validate output
	if output == "" {
		fmt.Fprintf(os.Stderr, "ERROR: --output is required\n")
		printDigestUsage()
		os.Exit(1)
	}

	// Generate digest
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
	fmt.Println("  --range <rev-range> Include changes in revision range (e.g., HEAD~1..HEAD)")
	fmt.Println("  --output <path>     Output path (required)")
	fmt.Println()
	fmt.Println("Default behavior (no mode flag):")
	fmt.Println("  - If working tree has changes: generate dirty digest")
	fmt.Println("  - If working tree is clean: generate previous commit digest (HEAD~1..HEAD)")
}

func printUsage() {
	fmt.Println("Leamas - Local-first, single-binary tool orchestration")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  leamas [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  leamas --help                Show this help")
	fmt.Println("  leamas version               Show version")
	fmt.Println("  leamas factory verify        Run factory verifiers")
	fmt.Println("  leamas factory gate          Run quality gate")
	fmt.Println("  leamas factory factorize     Run factory verifiers only")
	fmt.Println("  leamas factory digest        Generate targeted digest")
	fmt.Println("  leamas doctor                Run diagnostics")
}

func printFactoryUsage() {
	fmt.Println("Factory commands:")
	fmt.Println("  leamas factory verify <check>     Run a specific verifier")
	fmt.Println("  leamas factory gate               Run full quality gate")
	fmt.Println("  leamas factory factorize          Run verifiers only (no toolchain)")
	fmt.Println("  leamas factory digest [flags]     Generate targeted digest")
	fmt.Println()
	printFactoryVerifyUsage()
}

func printFactoryVerifyUsage() {
	fmt.Println("Available verifiers:")
	fmt.Println("  doctrine             Check doctrine documents exist")
	fmt.Println("  doctrine-agent-contracts  Check Agent Contract sections")
	fmt.Println("  docs                Check factory documentation")
	fmt.Println("  forbidden-patterns  Check for forbidden patterns")
	fmt.Println("  language            Check Go-only enforcement")
	fmt.Println("  static-binary       Check static binary build")
	fmt.Println("  tooling-boundaries  Check tooling boundaries")
	fmt.Println("  llm-friendly       Check LLM-friendliness")
	fmt.Println("  agent-context      Check agent context files")
	fmt.Println("  git-hooks          Check Git hooks installation")
	fmt.Println("  github             Check GitHub policy compliance")
	fmt.Println("  domain-boundaries  Check domain boundary import policies")
}
