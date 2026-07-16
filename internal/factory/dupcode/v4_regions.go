// Package dupcode provides V4 syntax-region construction.
//
// V4 chains are bounded by parser-derived executable regions. Each
// region represents a Go function declaration, method, or function
// literal that owns a contiguous range of normalized tokens. Chains
// cannot cross region boundaries, so two independent function bodies in
// the same file cannot conflate into one chain even if their sliding
// windows share fingerprints.
//
// The AST supplies byte-level source positions. The scanner tokenizes
// the same source using the same FileSet so AST positions map 1:1 to
// normalized token indexes. Comments do not shift normalized token
// indexes; auto-inserted semicolons map consistently with the rest of
// the token stream.
//
// Region ownership policy (innermost wins):
//
//   - TokenOwner[i] is the innermost executable region whose
//     normalized-token range covers Tokens[i].
//   - Tokens that belong to no executable region (package
//     declaration, top-level var/const/type declarations, blank
//     lines, comments that survive tokenization, inter-function
//     gaps) carry the zero value (Path=="").
//   - Window validation rejects any window that crosses the boundary
//     between regions OR between owned and unowned token ranges.
package dupcode

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
)

// v4SyntaxRegionKind enumerates the executable regions recognized by
// the V4 region inventory.
type v4SyntaxRegionKind uint8

const (
	// v4FunctionDeclarationRegion is a top-level func or method.
	v4FunctionDeclarationRegion v4SyntaxRegionKind = iota + 1
	// v4FunctionLiteralRegion is a func literal (closure or IIFE).
	v4FunctionLiteralRegion
)

// String returns a human-readable region-kind label used in tests and
// diagnostics.
func (k v4SyntaxRegionKind) String() string {
	switch k {
	case v4FunctionDeclarationRegion:
		return "func-decl"
	case v4FunctionLiteralRegion:
		return "func-lit"
	default:
		return fmt.Sprintf("region-kind(%d)", uint8(k))
	}
}

// v4SyntaxRegion describes one executable region in one source file.
//
// StartPos is the normalized-token index of the first token of the
// region (the `func` keyword for declarations and literals alike).
// EndPos is the normalized-token index of the inclusive final token
// belonging to the region.
//
// Region identity for chain construction is (Path, Ordinal). Kind is
// retained for diagnostics but does NOT participate in identity so that
// the chain-key comparator remains total and unambiguous.
type v4SyntaxRegion struct {
	Path      string
	Kind      v4SyntaxRegionKind
	Ordinal   int
	StartPos  int // normalized-token offset, inclusive
	EndPos    int // normalized-token offset, inclusive
	StartLine int
	EndLine   int
}

// v4SyntaxRegionID is the chain-construction identity of a region.
type v4SyntaxRegionID struct {
	Path    string
	Ordinal int
}

// String returns a deterministic identifier for logging and tests.
func (id v4SyntaxRegionID) String() string {
	return fmt.Sprintf("%s#%d", id.Path, id.Ordinal)
}

// v4TokenEntry records the scanner position of one normalized token.
// Positions are kept in a parallel slice so AST positions can be mapped
// back to token indexes with a single linear scan.
type v4TokenEntry struct {
	Pos token.Pos
	Tok token.Token
}

// v4FileAnalysis bundles the normalized token stream of one file
// together with the AST-derived region inventory.
//
// TokenOwner is the parallel innermost-owner array used by window
// validation. Every token index i in [0, len(Tokens)) has a
// TokenOwner[i] value; zero value means "no executable-region owner"
// (typically the package declaration, top-level var/const/type
// declarations, or inter-function gaps).
type v4FileAnalysis struct {
	Path       string
	Tokens     []token.Token
	Lines      []int // line number of each token; Lines[i] is the line of Tokens[i]
	Entries    []v4TokenEntry
	Regions    []v4SyntaxRegion
	TokenOwner []v4SyntaxRegionID

	// NormalizedTokens is the exact canonical token projection used for
	// content identity. It is parallel to Tokens and TokenOwner.
	NormalizedTokens []string
}

// windowFitsRegion reports whether the [start, end] inclusive token
// interval lies entirely within a single AST-derived executable
// region of a, and every token in [start, end] belongs to that region
// (no crossing into or out of a nested literal, and no package-decl
// or inter-function token).
//
// Returns (regionID, true) when the window fits a single region.
// Returns (zero, false) when the window is empty, out of range, or
// crosses any region boundary.
func (a *v4FileAnalysis) windowFitsRegion(start, end int) (v4SyntaxRegionID, bool) {
	if len(a.Regions) == 0 || len(a.TokenOwner) == 0 {
		return v4SyntaxRegionID{}, false
	}
	if start < 0 || end >= len(a.Tokens) || start > end {
		return v4SyntaxRegionID{}, false
	}
	owner := a.TokenOwner[start]
	if owner.Path == "" {
		return v4SyntaxRegionID{}, false
	}
	for i := start; i <= end; i++ {
		if a.TokenOwner[i] != owner {
			return v4SyntaxRegionID{}, false
		}
	}
	return owner, true
}

// filterWindowsToRegions drops windows whose token interval is not
// entirely contained in a single AST-derived executable region of its
// file. Windows that fall inside the package declaration or in
// inter-function gaps are removed so chains cannot span across
// independent function bodies.
//
// filterWindowsToRegions preserves the original windowMap key (the
// fingerprint string) so the downstream chain construction only sees
// region-bounded windows. It allocates a fresh backing slice for the
// kept windows; the input windowMap and its slice values are NOT
// modified.
func filterWindowsToRegions(windowMap map[string][]rawWindow, analyses map[string]*v4FileAnalysis) map[string][]rawWindow {
	if len(windowMap) == 0 || len(analyses) == 0 {
		return windowMap
	}
	out := make(map[string][]rawWindow, len(windowMap))
	for fp, wins := range windowMap {
		kept := make([]rawWindow, 0, len(wins))
		for _, w := range wins {
			a, ok := analyses[w.Path]
			if !ok {
				continue
			}
			if _, ok := a.windowFitsRegion(w.StartPos, w.EndPos); ok {
				kept = append(kept, w)
			}
		}
		if len(kept) > 0 {
			out[fp] = kept
		}
	}
	return out
}

// analyzeV4File reads path, parses the Go source, and returns the
// normalized token stream together with the AST-derived region
// inventory and per-token innermost ownership map.
//
// analyzeV4File fails closed on parse error: a syntactically invalid
// file produces an error rather than a silent fallback to unbounded
// chain construction.
//
// analyzeV4File uses ONE shared FileSet for both the parser and the
// scanner. This guarantees AST positions and scanner positions share
// the same byte-offset base, which is required for AST-node to
// token-index mapping.
func analyzeV4File(path string) (v4FileAnalysis, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return v4FileAnalysis{}, fmt.Errorf("read %s: %w", path, err)
	}

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return v4FileAnalysis{}, fmt.Errorf("parse %s: %w", path, err)
	}

	var analysis v4FileAnalysis
	analysis.Path = path
	parserFile := fset.File(astFile.Pos())
	if parserFile == nil {
		return v4FileAnalysis{}, fmt.Errorf("parser FileSet is missing the parsed file")
	}
	scannerTokenize(fset, parserFile, src, &analysis)
	buildRegions(&analysis, astFile)
	buildTokenOwner(&analysis)
	return analysis, nil
}

// scannerTokenize scans src with fset's scanner and appends entries,
// tokens, and lines to analysis. The scanner is initialised with the
// file the parser already registered in fset, so AST positions and
// scanner positions share the same byte-offset base.
func scannerTokenize(fset *token.FileSet, parserFile *token.File, src []byte, analysis *v4FileAnalysis) {
	var s scanner.Scanner
	s.Init(parserFile, src, nil, 0)
	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.COMMENT {
			continue
		}
		analysis.Entries = append(analysis.Entries, v4TokenEntry{Pos: pos, Tok: tok})
		analysis.Tokens = append(analysis.Tokens, tok)
		analysis.Lines = append(analysis.Lines, parserFile.Line(pos))
		analysis.NormalizedTokens = append(analysis.NormalizedTokens, normalizeV4Token(tok))
	}
	_ = fset
}

// indexAtOrAfter returns the smallest token index whose position is
// greater than or equal to pos. Returns -1 if no such token exists.
func (a *v4FileAnalysis) indexAtOrAfter(pos token.Pos) int {
	for i, e := range a.Entries {
		if e.Pos >= pos {
			return i
		}
	}
	return -1
}

// indexAtOrBefore returns the largest token index whose position is
// less than or equal to pos. Returns -1 if no such token exists.
func (a *v4FileAnalysis) indexAtOrBefore(pos token.Pos) int {
	last := -1
	for i, e := range a.Entries {
		if e.Pos > pos {
			break
		}
		last = i
	}
	return last
}

// buildRegions walks the AST and populates analysis.Regions with one
// entry per function declaration and per nested function literal.
//
// Declarations with no body (e.g. assembly function declarations) are
// skipped because they have no executable token region. Nested
// function literals are recorded as independent regions: their tokens
// belong to the literal, not to any enclosing function body.
//
// The final token of each region is the inclusive normalized token
// index that ends at the auto-inserted SEMICOLON after the closing
// `}` (when the SEMICOLON exists in the token stream).
func buildRegions(analysis *v4FileAnalysis, root *ast.File) {
	ordinal := 0

	recordRegion := func(kind v4SyntaxRegionKind, fnLike interface {
		Pos() token.Pos
		End() token.Pos
	}, body *ast.BlockStmt) {
		if body == nil {
			return
		}
		startIdx := analysis.indexAtOrAfter(fnLike.Pos())
		if startIdx < 0 {
			return
		}
		endIdx := analysis.inclusiveRegionEnd(startIdx, fnLike.End(), kind == v4FunctionDeclarationRegion)
		if endIdx < startIdx {
			return
		}
		analysis.Regions = append(analysis.Regions, v4SyntaxRegion{
			Path:      analysis.Path,
			Kind:      kind,
			Ordinal:   ordinal,
			StartPos:  startIdx,
			EndPos:    endIdx,
			StartLine: analysis.Lines[startIdx],
			EndLine:   analysis.Lines[endIdx],
		})
		ordinal++
	}

	for _, decl := range root.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			recordRegion(v4FunctionDeclarationRegion, d, d.Body)
			if d.Body != nil {
				ast.Inspect(d.Body, func(n ast.Node) bool {
					lit, ok := n.(*ast.FuncLit)
					if !ok {
						return true
					}
					recordRegion(v4FunctionLiteralRegion, lit, lit.Body)
					return true
				})
			}
		case *ast.GenDecl:
			// Package-level var/const/type declarations may carry
			// function literals as initialisers or composite-literal
			// elements. Inventory those literals so their tokens
			// receive unambiguous innermost ownership.
			for _, spec := range d.Specs {
				ast.Inspect(spec, func(n ast.Node) bool {
					lit, ok := n.(*ast.FuncLit)
					if !ok {
						return true
					}
					recordRegion(v4FunctionLiteralRegion, lit, lit.Body)
					return true
				})
			}
		}
	}
}

// inclusiveRegionEnd converts the AST-exclusive End() position to the
// normalized-token inclusive final-token boundary for the region.
//
// The Go scanner emits explicit SEMICOLON tokens for auto-inserted
// semicolons that follow a closing `}`. The conversion picks the
// largest token index whose position is strictly less than the AST
// end position (the closing `}` itself), and then advances one more
// token if that token is a SEMICOLON. The SEMICOLON is part of the
// region's normalized token stream because it terminates the function
// body's last statement.
//
// Search window: [startIdx, endPos-1] inclusive. When the AST end
// position is at or beyond EOF, the function uses the last token
// index.
func (a *v4FileAnalysis) inclusiveRegionEnd(startIdx int, endPos token.Pos, includeTrailingSemicolon bool) int {
	last := a.indexAtOrBefore(endPos - 1)
	if last < startIdx {
		return startIdx
	}
	// Advance past an auto-inserted SEMICOLON that follows the
	// closing `}`. The SEMICOLON belongs to the region's normalized
	// token stream.
	if includeTrailingSemicolon && last+1 < len(a.Tokens) && a.Tokens[last+1] == token.SEMICOLON {
		return last + 1
	}
	return last
}

// buildTokenOwner populates analysis.TokenOwner as a per-token
// innermost-owner array. Innermost executable region wins; tokens
// that belong to no executable region carry the zero value.
func buildTokenOwner(analysis *v4FileAnalysis) {
	analysis.TokenOwner = make([]v4SyntaxRegionID, len(analysis.Tokens))
	if len(analysis.Regions) == 0 {
		return
	}
	// Iterate regions in reverse ordinal order so the innermost
	// (last-recorded) region wins. Each token gets the LAST region
	// that contains it.
	for ri := len(analysis.Regions) - 1; ri >= 0; ri-- {
		r := analysis.Regions[ri]
		rid := v4SyntaxRegionID{Path: r.Path, Ordinal: r.Ordinal}
		for i := r.StartPos; i <= r.EndPos && i < len(analysis.TokenOwner); i++ {
			if analysis.TokenOwner[i].Path == "" {
				analysis.TokenOwner[i] = rid
			}
		}
	}
}
