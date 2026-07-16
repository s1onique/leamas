package dupcode

import (
	"fmt"
	"go/token"
)

// v4AnalyzedFile is the single per-file lexical inventory used by V4.
// FileTokens, Analysis.Tokens, and NormalizedTokens share one scanner
// result; AST positions are mapped back into those same token entries.
type v4AnalyzedFile struct {
	FileTokens       fileTokens
	Analysis         v4FileAnalysis
	NormalizedTokens []string
}

// analyzeV4AnalyzedFile parses and scans a file once, then exposes the
// authoritative token inventory to both window discovery and region-aware
// content materialization.
func analyzeV4AnalyzedFile(path string) (v4AnalyzedFile, error) {
	analysis, err := analyzeV4File(path)
	if err != nil {
		return v4AnalyzedFile{}, err
	}
	file := v4AnalyzedFile{
		FileTokens: fileTokens{
			path:   path,
			tokens: analysis.Tokens,
			lines:  analysis.Lines,
		},
		Analysis:         analysis,
		NormalizedTokens: analysis.NormalizedTokens,
	}
	if err := validateV4AnalyzedFile(file); err != nil {
		return v4AnalyzedFile{}, fmt.Errorf("validate %s: %w", path, err)
	}
	return file, nil
}

// validateV4AnalyzedFile fails closed when a token stream has lost alignment
// between the lexical, AST-owner, and normalized-content projections.
func validateV4AnalyzedFile(file v4AnalyzedFile) error {
	n := len(file.FileTokens.tokens)
	if len(file.FileTokens.lines) != n {
		return fmt.Errorf("file token/line alignment: tokens=%d lines=%d", n, len(file.FileTokens.lines))
	}
	if len(file.Analysis.Tokens) != n || len(file.Analysis.Entries) != n {
		return fmt.Errorf("analysis token alignment: file=%d analysis=%d entries=%d",
			n, len(file.Analysis.Tokens), len(file.Analysis.Entries))
	}
	if len(file.Analysis.Lines) != n || len(file.Analysis.TokenOwner) != n {
		return fmt.Errorf("analysis geometry alignment: tokens=%d lines=%d owners=%d",
			n, len(file.Analysis.Lines), len(file.Analysis.TokenOwner))
	}
	if len(file.Analysis.NormalizedTokens) != n || len(file.NormalizedTokens) != n {
		return fmt.Errorf("normalized token alignment: tokens=%d analysis=%d file=%d",
			n, len(file.Analysis.NormalizedTokens), len(file.NormalizedTokens))
	}
	for i := range file.Analysis.Tokens {
		if file.Analysis.Tokens[i] != file.FileTokens.tokens[i] {
			return fmt.Errorf("token mismatch at index %d: analysis=%s file=%s",
				i, file.Analysis.Tokens[i], file.FileTokens.tokens[i])
		}
		if file.Analysis.NormalizedTokens[i] != file.NormalizedTokens[i] {
			return fmt.Errorf("normalized token mismatch at index %d", i)
		}
	}
	return nil
}

// rebaseV4AnalysisPath atomically changes the path carried by the analysis
// and every embedded region and nonzero token-owner identity.
func rebaseV4AnalysisPath(analysis *v4FileAnalysis, normalizedPath string) {
	if analysis == nil {
		return
	}
	analysis.Path = normalizedPath
	for i := range analysis.Regions {
		analysis.Regions[i].Path = normalizedPath
	}
	for i := range analysis.TokenOwner {
		if analysis.TokenOwner[i].Path != "" {
			analysis.TokenOwner[i].Path = normalizedPath
		}
	}
}

func rebaseV4AnalyzedFilePath(file *v4AnalyzedFile, normalizedPath string) {
	if file == nil {
		return
	}
	rebaseV4AnalysisPath(&file.Analysis, normalizedPath)
	file.FileTokens.path = normalizedPath
}

// normalizeV4Token is the canonical normalized token value used for seed
// discovery and exact-content identity. Source spelling of identifiers and
// literals is intentionally normalized, while operators and keywords retain
// their token spelling.
func normalizeV4Token(tok token.Token) string {
	switch tok {
	case token.IDENT:
		return "IDENT"
	case token.STRING, token.CHAR:
		return "STRING"
	case token.INT, token.FLOAT, token.IMAG:
		return "NUMBER"
	default:
		return tok.String()
	}
}
