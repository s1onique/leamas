// Leamas CLI - Local-first, single-binary tool orchestration
package main

import (
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/execution"
	"github.com/s1onique/leamas/internal/factory/agentcontext"
	"github.com/s1onique/leamas/internal/factory/boundary"
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

func main() {
	// Emergency re-entry fuse: prevent Leamas from running inside Leamas
	// This must be checked before any other operation
	_, err := execution.NewExecutionRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		fmt.Fprintln(os.Stderr, "Leamas cannot be started from within a Leamas execution.")
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		handleVersion()
	case "factory":
		handleFactory()
	case "doctor":
		fmt.Println("Leamas doctor: all systems operational")
	case "cockpit":
		handleCockpit()
	case "witness":
		handleWitness()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
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
	case "coverage":
		f := gate.CheckCoverage(".")
		for _, f := range f {
			findings = append(findings, struct {
				path    string
				kind    string
				message string
			}{f.Path, f.Kind, f.Message})
		}
	case "dupcode":
		handleFactoryVerifyDupcode()
	case "dupcode-baseline":
		handleFactoryVerifyDupcodeBaseline()
	case "act-doctrine-compiler":
		runDoctrineCompilerVerifier()
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

// usageText returns the main usage text for leamas CLI.
func usageText() string {
	return `Leamas - Local-first, single-binary tool orchestration

Usage:
  leamas [command]

Commands:
  leamas --help               Show this help
  leamas version              Show version
  leamas factory verify       Run factory verifiers
  leamas factory gate        Run quality gate
  leamas factory factorize   Run factory verifiers only
  leamas factory digest      Generate targeted digest
  leamas factory coverage    Check coverage threshold
  leamas doctor              Run diagnostics
  leamas cockpit             Local web cockpit
  leamas witness             Witness proxy commands`
}

// factoryUsageText returns the factory subcommand usage text.
func factoryUsageText() string {
	return `Factory commands:
  leamas factory verify <check>   Run a specific verifier
  leamas factory gate           Run full quality gate
  leamas factory factorize      Run verifiers only (no toolchain)
  leamas factory digest [flags] Generate targeted digest
  leamas factory coverage        Check coverage threshold`
}

func printUsage() {
	fmt.Print(usageText())
	fmt.Println()
}

func printFactoryUsage() {
	fmt.Print(factoryUsageText())
	fmt.Println()
	printFactoryVerifyUsage()
}

// knownFactoryVerifyChecks returns the list of known factory verify check names.
func knownFactoryVerifyChecks() []string {
	return []string{
		"doctrine",
		"doctrine-agent-contracts",
		"docs",
		"dupcode",
		"dupcode-baseline",
		"forbidden-patterns",
		"language",
		"static-binary",
		"tooling-boundaries",
		"llm-friendly",
		"agent-context",
		"git-hooks",
		"github",
		"domain-boundaries",
		"coverage",
		"act-doctrine-compiler",
	}
}

// isKnownFactoryVerifyCheck returns true if the given check name is known.
func isKnownFactoryVerifyCheck(check string) bool {
	for _, known := range knownFactoryVerifyChecks() {
		if check == known {
			return true
		}
	}
	return false
}

func printFactoryVerifyUsage() {
	fmt.Println("Available verifiers:")
	fmt.Println("  doctrine              Check doctrine documents exist")
	fmt.Println("  doctrine-agent-contracts  Check Agent Contract sections")
	fmt.Println("  docs                 Check factory documentation")
	fmt.Println("  dupcode               Check for duplicate code")
	fmt.Println("  dupcode-baseline      Check dupcode baseline integrity")
	fmt.Println("  forbidden-patterns   Check for forbidden patterns")
	fmt.Println("  language             Check Go-only enforcement")
	fmt.Println("  static-binary        Check static binary build")
	fmt.Println("  tooling-boundaries   Check tooling boundaries")
	fmt.Println("  llm-friendly         Check LLM-friendliness")
	fmt.Println("  agent-context        Check agent context files")
	fmt.Println("  git-hooks            Check Git hooks installation")
	fmt.Println("  github               Check GitHub policy compliance")
	fmt.Println("  domain-boundaries    Check domain boundary import policies")
	fmt.Println("  coverage             Check coverage threshold")
}
