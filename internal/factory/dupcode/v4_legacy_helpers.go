// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"sort"
	"strings"
)

func findCommonWindows(ft1, ft2 fileTokens, cfg Config,
	windowMap map[string][]rawWindow,
	fingerprintTokens map[string]int) {

	fp1 := make(map[string][]int)
	for i := 0; i <= len(ft1.tokens)-cfg.MinTokens; i++ {
		window := ft1.tokens[i : i+cfg.MinTokens]
		fp := normalizeFingerprint(window)
		fp1[fp] = append(fp1[fp], i)
	}

	fp2 := make(map[string][]int)
	for i := 0; i <= len(ft2.tokens)-cfg.MinTokens; i++ {
		window := ft2.tokens[i : i+cfg.MinTokens]
		fp := normalizeFingerprint(window)
		fp2[fp] = append(fp2[fp], i)
	}

	// Sort fingerprints for deterministic processing
	var fps []string
	for fp := range fp1 {
		fps = append(fps, fp)
	}
	sort.Strings(fps)

	for _, fp := range fps {
		pos1 := fp1[fp]
		pos2, ok := fp2[fp]
		if !ok {
			continue
		}

		// Record token count once per fingerprint
		if _, exists := fingerprintTokens[fp]; !exists {
			fingerprintTokens[fp] = cfg.MinTokens
		}

		// Add raw windows for file 1
		for _, startPos := range pos1 {
			startLine := ft1.lines[startPos]
			endLine := ft1.lines[startPos+cfg.MinTokens-1]
			if endLine-startLine+1 >= cfg.MinLines {
				windowMap[fp] = append(windowMap[fp], rawWindow{
					Path:      ft1.path,
					StartLine: startLine,
					EndLine:   endLine,
					StartPos:  startPos,
					EndPos:    startPos + cfg.MinTokens - 1,
				})
			}
		}

		// Add raw windows for file 2
		for _, startPos := range pos2 {
			startLine := ft2.lines[startPos]
			endLine := ft2.lines[startPos+cfg.MinTokens-1]
			if endLine-startLine+1 >= cfg.MinLines {
				windowMap[fp] = append(windowMap[fp], rawWindow{
					Path:      ft2.path,
					StartLine: startLine,
					EndLine:   endLine,
					StartPos:  startPos,
					EndPos:    startPos + cfg.MinTokens - 1,
				})
			}
		}
	}
}

func tokenizeFile(path string) (fileTokens, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileTokens{}, fmt.Errorf("reading %s: %w", path, err)
	}
	fset := token.NewFileSet()
	file := fset.AddFile(path, fset.Base(), len(data))
	var s scanner.Scanner
	s.Init(file, data, nil, 0)
	var tokens []token.Token
	var lines []int
	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.COMMENT {
			continue
		}
		tokens = append(tokens, tok)
		lines = append(lines, file.Line(pos))
	}
	return fileTokens{path: path, tokens: tokens, lines: lines}, nil
}

// normalizeFingerprint produces a canonical string representation for a token window.
// Only IDENT, STRING, CHAR, INT, FLOAT, IMAG are normalized to semantic categories;
// all other tokens use their standard library string representation.
func normalizeFingerprint(tokens []token.Token) string {
	// Heuristic pre-allocation: most tokens are short identifiers/operators.
	// GROW is a conservative estimate; under-allocation just causes reallocation.
	var buf strings.Builder
	buf.Grow(len(tokens) * 8)

	for i, tok := range tokens {
		switch tok {
		case token.IDENT:
			buf.WriteString("IDENT")
		case token.STRING, token.CHAR:
			buf.WriteString("STRING")
		case token.INT, token.FLOAT, token.IMAG:
			buf.WriteString("NUMBER")
		default:
			// Use standard library string representation for operators/keywords
			buf.WriteString(tok.String())
		}

		if i+1 < len(tokens) {
			buf.WriteByte(' ')
		}
	}

	return buf.String()
}

func truncateFingerprint(fp string) string {
	if len(fp) > 40 {
		return fp[:40] + "..."
	}
	return fp
}

func deduplicateOccurrences(occs []Occurrence) []Occurrence {
	if len(occs) < 2 {
		return occs
	}
	sort.Slice(occs, func(i, j int) bool {
		return occs[i].StartLine < occs[j].StartLine
	})
	var result []Occurrence
	current := occs[0]
	for i := 1; i < len(occs); i++ {
		next := occs[i]
		if next.StartLine <= current.EndLine+1 {
			if next.EndLine > current.EndLine {
				current.EndLine = next.EndLine
			}
		} else {
			result = append(result, current)
			current = next
		}
	}
	result = append(result, current)
	return result
}

// PrintFindings prints findings in human-readable format.
func PrintFindings(findings []Finding) {
	if len(findings) == 0 {
		fmt.Println("No duplicate code detected.")
		return
	}
	fmt.Printf("Found %d duplicate code blocks:\n\n", len(findings))
	for i, f := range findings {
		fmt.Printf("%d. Duplicate block (%d tokens, ~%d lines):\n", i+1, f.TokenCount, f.LineCount)
		for _, occ := range f.Occurrences {
			fmt.Printf("   - %s:%d-%d\n", occ.Path, occ.StartLine, occ.EndLine)
		}
		fmt.Println()
	}
}
