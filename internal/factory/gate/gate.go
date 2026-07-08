// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/agentcontext"
	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/docs"
	"github.com/s1onique/leamas/internal/factory/doctrine"
	"github.com/s1onique/leamas/internal/factory/forbidden"
	"github.com/s1onique/leamas/internal/factory/githooks"
	"github.com/s1onique/leamas/internal/factory/github"
	"github.com/s1onique/leamas/internal/factory/language"
	"github.com/s1onique/leamas/internal/factory/llmfriendly"
	"github.com/s1onique/leamas/internal/factory/staticbinary"
	"github.com/s1onique/leamas/internal/factory/tooling"
)

// Verifier represents a Factory verifier.
type Verifier struct {
	Name string
	Run  func(root string) []checks.Finding
}

// AllVerifiers returns all Factory policy verifiers (for factorize).
// Note: "github" verifier requires network access and is not included by default.
// Run `leamas factory verify github` explicitly for remote policy verification.
func AllVerifiers() []Verifier {
	return []Verifier{
		{"doctrine", doctrine.CheckRepo},
		{"doctrine-agent-contracts", doctrine.CheckRepo},
		{"docs", docs.CheckRepo},
		{"forbidden-patterns", forbidden.CheckRepo},
		{"language", language.CheckRepo},
		{"static-binary", staticbinary.CheckRepo},
		{"tooling-boundaries", tooling.CheckRepo},
		{"llm-friendly", llmFriendlyVerifier},
		{"agent-context", agentContextVerifier},
		{"git-hooks", gitHooksVerifier},
	}
}

func llmFriendlyVerifier(root string) []checks.Finding {
	cfg := llmfriendly.DefaultConfig()
	findings, _ := llmfriendly.CheckRepo(root, cfg)
	return convertLLMFriendlyFindings(findings)
}

func agentContextVerifier(root string) []checks.Finding {
	findings, _ := agentcontext.CheckRepo(root)
	return convertAgentContextFindings(findings)
}

func gitHooksVerifier(root string) []checks.Finding {
	findings, _ := githooks.CheckRepo(root)
	return convertGitHooksFindings(findings)
}

func convertLLMFriendlyFindings(src []llmfriendly.Finding) []checks.Finding {
	result := make([]checks.Finding, len(src))
	for i, f := range src {
		result[i] = checks.Finding{
			Path:     f.Path,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: checks.SeverityError,
		}
	}
	return result
}

func convertAgentContextFindings(src []agentcontext.Finding) []checks.Finding {
	result := make([]checks.Finding, len(src))
	for i, f := range src {
		result[i] = checks.Finding{
			Path:     f.Path,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: checks.SeverityError,
		}
	}
	return result
}

func convertGitHooksFindings(src []githooks.Finding) []checks.Finding {
	result := make([]checks.Finding, len(src))
	for i, f := range src {
		result[i] = checks.Finding{
			Path:     f.Path,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: checks.SeverityError,
		}
	}
	return result
}

func githubVerifier(root string) []checks.Finding {
	findings, _ := github.CheckRepo(root)
	return convertGithubFindings(findings)
}

func convertGithubFindings(src []github.Finding) []checks.Finding {
	result := make([]checks.Finding, len(src))
	for i, f := range src {
		severity := checks.SeverityError
		if f.Severity == "info" {
			severity = checks.SeverityWarn
		}
		result[i] = checks.Finding{
			Path:     f.Path,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: severity,
		}
	}
	return result
}

// RunGate runs all verifiers and Go toolchain checks.
func RunGate(root string) int {
	failed := false

	// Run all verifiers in deterministic order
	verifiers := AllVerifiers()
	sort.Slice(verifiers, func(i, j int) bool {
		return verifiers[i].Name < verifiers[j].Name
	})

	for _, v := range verifiers {
		findings := v.Run(root)
		if len(findings) > 0 {
			failed = true
			fmt.Printf("\n--- %s FAILED ---\n", v.Name)
			for _, f := range findings {
				fmt.Printf("  %s: %s: %s\n", f.Path, f.Kind, f.Message)
			}
		} else {
			fmt.Printf("  %s: OK\n", v.Name)
		}
	}

	// Run Go toolchain checks
	fmt.Printf("\n--- Go toolchain ---\n")

	// go mod tidy
	fmt.Printf("  go mod tidy...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		fmt.Printf(" FAILED: %v\n", err)
		failed = true
	} else {
		fmt.Printf(" OK\n")
	}

	// Check go.mod/go.sum didn't change
	if checks.FileExists(filepath.Join(root, "go.sum")) {
		cmd = exec.Command("git", "diff", "--quiet", "go.mod", "go.sum")
		cmd.Dir = root
		if err := cmd.Run(); err != nil {
			fmt.Printf("  go.mod/go.sum changed after tidy\n")
			failed = true
		}
	} else {
		cmd = exec.Command("git", "diff", "--quiet", "go.mod")
		cmd.Dir = root
		if err := cmd.Run(); err != nil {
			fmt.Printf("  go.mod changed after tidy\n")
			failed = true
		}
	}

	// gofmt check
	fmt.Printf("  gofmt...")
	cmd = exec.Command("gofmt", "-l", ".")
	cmd.Dir = root
	output, _ := cmd.Output()
	if len(strings.TrimSpace(string(output))) > 0 {
		fmt.Printf(" FAILED\n")
		fmt.Printf("    Unformatted files:\n")
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, f := range lines {
			if f != "" {
				fmt.Printf("    - %s\n", f)
			}
		}
		failed = true
	} else {
		fmt.Printf(" OK\n")
	}

	// go vet
	fmt.Printf("  go vet ./...")
	cmd = exec.Command("go", "vet", "./...")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		fmt.Printf(" FAILED\n")
		failed = true
	} else {
		fmt.Printf(" OK\n")
	}

	// go test
	fmt.Printf("  go test ./...")
	cmd = exec.Command("go", "test", "./...")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		fmt.Printf(" FAILED\n")
		failed = true
	} else {
		fmt.Printf(" OK\n")
	}

	// CGO_ENABLED=0 build
	fmt.Printf("  static build...")
	cmd = exec.Command("go", "build", "-trimpath", "-o", "bin/leamas", "./cmd/leamas")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if err := cmd.Run(); err != nil {
		fmt.Printf(" FAILED\n")
		failed = true
	} else {
		fmt.Printf(" OK\n")
	}

	if failed {
		fmt.Printf("\n*** GATE FAILED ***\n")
		return 1
	}

	fmt.Printf("\n*** GATE PASSED ***\n")
	return 0
}

// RunFactorize runs all Factory policy verifiers without toolchain checks.
func RunFactorize(root string) int {
	failed := false

	verifiers := AllVerifiers()
	sort.Slice(verifiers, func(i, j int) bool {
		return verifiers[i].Name < verifiers[j].Name
	})

	for _, v := range verifiers {
		findings := v.Run(root)
		if len(findings) > 0 {
			failed = true
			fmt.Printf("\n--- %s FAILED ---\n", v.Name)
			for _, f := range findings {
				fmt.Printf("  %s: %s: %s\n", f.Path, f.Kind, f.Message)
			}
		} else {
			fmt.Printf("  %s: OK\n", v.Name)
		}
	}

	if failed {
		fmt.Printf("\n*** FACTORIZE FAILED ***\n")
		return 1
	}

	fmt.Printf("\n*** FACTORIZE PASSED ***\n")
	return 0
}
