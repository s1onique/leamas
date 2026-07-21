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

func normalizeFingerprint(tokens []token.Token) string {
	// Pre-allocate with worst-case estimate: 7 bytes per token + separators
	// "IDENT", "STRING", "NUMBER" are the longest tokens at 6 chars, + 1 separator
	var buf strings.Builder
	buf.Grow(len(tokens)*7 + 1)
	for i, t := range tokens {
		switch t {
		case token.IDENT:
			buf.WriteString("IDENT")
		case token.STRING, token.CHAR:
			buf.WriteString("STRING")
		case token.INT, token.FLOAT, token.IMAG:
			buf.WriteString("NUMBER")
		case token.EOF:
			buf.WriteString("EOF")
		case token.BREAK:
			buf.WriteString("BREAK")
		case token.CASE:
			buf.WriteString("CASE")
		case token.CONST:
			buf.WriteString("CONST")
		case token.CONTINUE:
			buf.WriteString("CONTINUE")
		case token.DEFAULT:
			buf.WriteString("DEFAULT")
		case token.DEFER:
			buf.WriteString("DEFER")
		case token.ELSE:
			buf.WriteString("ELSE")
		case token.FALLTHROUGH:
			buf.WriteString("FALLTHROUGH")
		case token.FOR:
			buf.WriteString("FOR")
		case token.FUNC:
			buf.WriteString("FUNC")
		case token.GO:
			buf.WriteString("GO")
		case token.GOTO:
			buf.WriteString("GOTO")
		case token.IF:
			buf.WriteString("IF")
		case token.IMPORT:
			buf.WriteString("IMPORT")
		case token.INTERFACE:
			buf.WriteString("INTERFACE")
		case token.MAP:
			buf.WriteString("MAP")
		case token.PACKAGE:
			buf.WriteString("PACKAGE")
		case token.RANGE:
			buf.WriteString("RANGE")
		case token.RETURN:
			buf.WriteString("RETURN")
		case token.SELECT:
			buf.WriteString("SELECT")
		case token.STRUCT:
			buf.WriteString("STRUCT")
		case token.SWITCH:
			buf.WriteString("SWITCH")
		case token.TYPE:
			buf.WriteString("TYPE")
		case token.VAR:
			buf.WriteString("VAR")
		case token.ASSIGN:
			buf.WriteString("ASSIGN")
		case token.COLON:
			buf.WriteString("COLON")
		case token.COMMA:
			buf.WriteString("COMMA")
		case token.DEC:
			buf.WriteString("DEC")
		case token.ELLIPSIS:
			buf.WriteString("ELLIPSIS")
		case token.INC:
			buf.WriteString("INC")
		case token.LAND:
			buf.WriteString("LAND")
		case token.LOR:
			buf.WriteString("LOR")
		case token.NOT:
			buf.WriteString("NOT")
		case token.PERIOD:
			buf.WriteString("PERIOD")
		case token.ADD:
			buf.WriteString("ADD")
		case token.SUB:
			buf.WriteString("SUB")
		case token.MUL:
			buf.WriteString("MUL")
		case token.QUO:
			buf.WriteString("QUO")
		case token.REM:
			buf.WriteString("REM")
		case token.AND:
			buf.WriteString("AND")
		case token.OR:
			buf.WriteString("OR")
		case token.XOR:
			buf.WriteString("XOR")
		case token.SHL:
			buf.WriteString("SHL")
		case token.SHR:
			buf.WriteString("SHR")
		case token.AND_NOT:
			buf.WriteString("AND_NOT")
		case token.LSS:
			buf.WriteString("LSS")
		case token.GTR:
			buf.WriteString("GTR")
		case token.LEQ:
			buf.WriteString("LEQ")
		case token.GEQ:
			buf.WriteString("GEQ")
		case token.EQL:
			buf.WriteString("EQL")
		case token.NEQ:
			buf.WriteString("NEQ")
		case token.RBRACK:
			buf.WriteString("RBRACK")
		case token.RPAREN:
			buf.WriteString("RPAREN")
		case token.RBRACE:
			buf.WriteString("RBRACE")
		case token.LPAREN:
			buf.WriteString("LPAREN")
		case token.LBRACK:
			buf.WriteString("LBRACK")
		case token.LBRACE:
			buf.WriteString("LBRACE")
		case token.SEMICOLON:
			buf.WriteString("SEMICOLON")
		case token.DEFINE:
			buf.WriteString("DEFINE")
		case token.ARROW:
			buf.WriteString("ARROW")
		default:
			buf.WriteString(t.String())
		}
		if i < len(tokens)-1 {
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
