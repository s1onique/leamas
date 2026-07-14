// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"fmt"
	"strings"
)

// ExitCodeFromCompareResult returns the appropriate exit code for a compare result.
func ExitCodeFromCompareResult(result CompareResult) int {
	if result.HasChanges {
		return 1
	}
	return 0
}

// NormalizeFingerprintForBaseline returns a stable, normalized fingerprint for baseline storage.
func NormalizeFingerprintForBaseline(tokens []string) string {
	normalized := strings.Join(tokens, " ")
	return StableFingerprintHash(normalized)
}

// PrintCompareResult prints the comparison result in human-readable format.
func PrintCompareResult(result CompareResult) {
	if !result.HasChanges {
		fmt.Println("No new or worsened duplicate code detected.")
		return
	}

	if len(result.NewFindings) > 0 {
		fmt.Printf("\nNew duplicate code blocks (%d):\n\n", len(result.NewFindings))
		for i, f := range result.NewFindings {
			fmt.Printf("%d. New duplicate block (%d tokens, ~%d lines):\n", i+1, f.TokenCount, f.LineCount)
			for _, occ := range f.Occurrences {
				fmt.Printf("   - %s:%d-%d\n", occ.Path, occ.StartLine, occ.EndLine)
			}
			fmt.Println()
		}
	}

	if len(result.WorsenedFindings) > 0 {
		fmt.Printf("\nWorsened duplicate code blocks (%d):\n\n", len(result.WorsenedFindings))
		for i, f := range result.WorsenedFindings {
			fmt.Printf("%d. Worsened fingerprint (now %d occurrences, was %d):\n", i+1, f.TotalNow, len(f.BaselineOccurrences))
			fmt.Printf("   Baseline locations (%d):\n", len(f.BaselineOccurrences))
			for _, occ := range f.BaselineOccurrences {
				fmt.Printf("     - %s:%d-%d\n", occ.Path, occ.StartLine, occ.EndLine)
			}
			fmt.Printf("   New locations (%d):\n", len(f.NewOccurrences))
			for _, occ := range f.NewOccurrences {
				fmt.Printf("     + %s:%d-%d\n", occ.Path, occ.StartLine, occ.EndLine)
			}
			fmt.Println()
		}
	}
}
