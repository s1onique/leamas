// Leamas CLI - Local-first, single-binary tool orchestration
package main

import (
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/factory/agentcontext"
	"github.com/s1onique/leamas/internal/factory/githooks"
	"github.com/s1onique/leamas/internal/factory/llmfriendly"
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
		if len(os.Args) < 3 {
			printFactoryUsage()
			os.Exit(1)
		}
		switch os.Args[2] {
		case "verify":
			if len(os.Args) < 4 {
				printFactoryVerifyUsage()
				os.Exit(1)
			}
			switch os.Args[3] {
			case "llm-friendly":
				runLLMFriendlyCheck()
			case "agent-context":
				runAgentContextCheck()
			case "git-hooks":
				runGitHooksCheck()
			default:
				fmt.Fprintf(os.Stderr, "unknown verify command: %s\n", os.Args[3])
				printFactoryVerifyUsage()
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown factory command: %s\n", os.Args[2])
			printFactoryUsage()
			os.Exit(1)
		}
	case "doctor":
		fmt.Println("Leamas doctor: all systems operational")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Leamas - Local-first, single-binary tool orchestration")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  leamas [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  leamas --help           Show this help")
	fmt.Println("  leamas version          Show version")
	fmt.Println("  leamas factory verify   Run factory verifiers")
	fmt.Println("  leamas doctor           Run diagnostics")
}

func printFactoryUsage() {
	fmt.Println("Factory commands:")
	fmt.Println("  leamas factory verify <check>   Run a specific verifier")
	fmt.Println()
	fmt.Println("Available verifiers:")
	fmt.Println("  llm-friendly    Check repository LLM-friendliness")
	fmt.Println("  agent-context   Check agent context files")
	fmt.Println("  git-hooks       Check Git hook installation")
}

func printFactoryVerifyUsage() {
	fmt.Println("Usage: leamas factory verify <check>")
	fmt.Println()
	fmt.Println("Available checks:")
	fmt.Println("  llm-friendly    Check repository LLM-friendliness")
	fmt.Println("  agent-context   Check agent context files")
	fmt.Println("  git-hooks       Check Git hook installation")
}

func runLLMFriendlyCheck() {
	cfg := llmfriendly.DefaultConfig()
	findings, err := llmfriendly.CheckRepo(".", cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LLM-friendliness verification ERROR: %v\n", err)
		os.Exit(1)
	}

	if len(findings) == 0 {
		fmt.Println("LLM-friendliness verification PASSED")
		os.Exit(0)
	}

	fmt.Println("LLM-friendliness verification FAILED")
	// Sort and print findings deterministically
	for _, f := range findings {
		fmt.Printf("%s: %s: %s\n", f.Path, f.Kind, f.Message)
	}
	os.Exit(1)
}

func runAgentContextCheck() {
	findings, err := agentcontext.CheckRepo(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Agent context verification ERROR: %v\n", err)
		os.Exit(1)
	}

	if len(findings) == 0 {
		fmt.Println("Agent context verification PASSED")
		os.Exit(0)
	}

	fmt.Println("Agent context verification FAILED")
	for _, f := range findings {
		fmt.Printf("%s: %s: %s\n", f.Path, f.Kind, f.Message)
	}
	os.Exit(1)
}

func runGitHooksCheck() {
	findings, err := githooks.CheckRepo(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Git hooks verification ERROR: %v\n", err)
		os.Exit(1)
	}

	if len(findings) == 0 {
		fmt.Println("Git hooks verification PASSED")
		os.Exit(0)
	}

	fmt.Println("Git hooks verification FAILED")
	for _, f := range findings {
		fmt.Printf("%s: %s: %s\n", f.Path, f.Kind, f.Message)
	}
	os.Exit(1)
}
