package coverage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ProfileBlock represents a single coverage block from a raw coverage profile.
type ProfileBlock struct {
	FilePath      string
	StartLine     int
	StartCol      int
	EndLine       int
	EndCol        int
	NumStatements int
	Count         int
}

// WeightedModuleSummary represents statement-weighted coverage for a module.
type WeightedModuleSummary struct {
	Module            string  `json:"module"`
	Percent           float64 `json:"percent"`
	Packages          int     `json:"packages"`
	CoveredStatements int     `json:"covered_statements"`
	TotalStatements   int     `json:"total_statements"`
}

// ProfileReport represents the complete weighted coverage report.
type ProfileReport struct {
	SchemaVersion   int                     `json:"schema_version"`
	TotalPercent    float64                 `json:"total_percent"`
	TotalCovered    int                     `json:"total_covered"`
	TotalStatements int                     `json:"total_statements"`
	Modules         []WeightedModuleSummary `json:"modules"`
}

// ParseProfilePath parses a file path from a coverage profile block.
// Format: "github.com/s1onique/leamas/internal/foo/foo.go:10.1,20.2"
func ParseProfilePath(filePath string) (string, error) {
	// Remove line/column info by finding the last colon that's part of line:col
	// The format is: "path/to/file.go:startLine.startCol,endLine.endCol"

	// Find the comma that separates start and end positions
	commaIdx := strings.LastIndex(filePath, ",")
	if commaIdx == -1 {
		return "", fmt.Errorf("no comma in profile path: %s", filePath)
	}

	// Extract just the file path
	rawPath := filePath[:commaIdx]

	// Find the last colon in the path (this separates file from line:col)
	lastColon := strings.LastIndex(rawPath, ":")
	if lastColon == -1 {
		return "", fmt.Errorf("no colon in profile path: %s", rawPath)
	}

	file := rawPath[:lastColon]

	// Remove the file name to get the package path
	lastSlash := strings.LastIndex(file, "/")
	if lastSlash == -1 {
		return "", fmt.Errorf("no slash in path: %s", file)
	}

	return file[:lastSlash], nil
}

// ParseProfileBlock parses a single coverage block line.
// Format: "github.com/s1onique/leamas/internal/foo/foo.go:10.1,20.2 7 0"
// Returns: (numStatements, count)
func ParseProfileBlock(line string) (*ProfileBlock, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "mode:") {
		return nil, nil
	}

	// Format: "path/to/file.go:startLine.startCol,endLine.endCol numStatements count"
	parts := strings.Fields(trimmed)
	if len(parts) < 3 {
		return nil, fmt.Errorf("malformed profile block: %s", line)
	}

	// Parse file path and positions
	filePath := parts[0]

	// Find the last colon to separate file path from positions
	lastColon := strings.LastIndex(filePath, ":")
	if lastColon == -1 {
		return nil, fmt.Errorf("no colon in file path: %s", filePath)
	}

	fullFilePath := filePath[:lastColon]
	positions := filePath[lastColon+1:]

	// Positions is "startLine.startCol,endLine.endCol"
	// Split by comma to get start and end
	posParts := strings.Split(positions, ",")
	if len(posParts) != 2 {
		return nil, fmt.Errorf("malformed positions: %s", positions)
	}

	// Parse start position (line.col)
	startParts := strings.Split(posParts[0], ".")
	if len(startParts) != 2 {
		return nil, fmt.Errorf("malformed start position: %s", posParts[0])
	}

	startLine, err := strconv.Atoi(startParts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid start line: %s", startParts[0])
	}
	startCol, err := strconv.Atoi(startParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid start col: %s", startParts[1])
	}

	// Parse end position (line.col)
	endParts := strings.Split(posParts[1], ".")
	if len(endParts) != 2 {
		return nil, fmt.Errorf("malformed end position: %s", posParts[1])
	}

	endLine, err := strconv.Atoi(endParts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid end line: %s", endParts[0])
	}
	endCol, err := strconv.Atoi(endParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid end col: %s", endParts[1])
	}

	// Parse statement count and coverage count
	numStatements, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid numStatements: %s", parts[1])
	}

	count, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid count: %s", parts[2])
	}

	return &ProfileBlock{
		FilePath:      fullFilePath,
		StartLine:     startLine,
		StartCol:      startCol,
		EndLine:       endLine,
		EndCol:        endCol,
		NumStatements: numStatements,
		Count:         count,
	}, nil
}

// ParseProfile parses a raw Go coverage profile and computes statement-weighted module coverage.
func ParseProfile(profilePath string) (*ProfileReport, error) {
	file, err := os.Open(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open profile: %w", err)
	}
	defer file.Close()

	return ParseProfileReader(file)
}

// ParseProfileReader parses a raw Go coverage profile from an io.Reader.
func ParseProfileReader(r io.Reader) (*ProfileReport, error) {
	scanner := bufio.NewScanner(r)

	// Track coverage data by module
	type moduleData struct {
		covered  int
		total    int
		packages map[string]bool
	}
	moduleStats := make(map[string]*moduleData)

	var totalCovered, totalStatements int

	for scanner.Scan() {
		line := scanner.Text()

		block, err := ParseProfileBlock(line)
		if err != nil {
			// Skip malformed blocks
			continue
		}
		if block == nil {
			continue
		}

		// Classify the file path to a module
		pkgPath, err := ParseProfilePath(block.FilePath)
		if err != nil {
			// Try to classify the file path directly
			pkgPath = block.FilePath
		}
		module := ClassifyModule(pkgPath)

		// Get or create module data
		md, exists := moduleStats[module]
		if !exists {
			md = &moduleData{packages: make(map[string]bool)}
			moduleStats[module] = md
		}

		// Track package
		md.packages[pkgPath] = true

		// Update module stats
		md.total += block.NumStatements
		if block.Count > 0 {
			md.covered += block.NumStatements
		}

		// Update total
		totalStatements += block.NumStatements
		if block.Count > 0 {
			totalCovered += block.NumStatements
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading profile: %w", err)
	}

	// Build module summaries
	var modules []WeightedModuleSummary
	for module, md := range moduleStats {
		var percent float64
		if md.total > 0 {
			percent = float64(md.covered) / float64(md.total) * 100
		}

		modules = append(modules, WeightedModuleSummary{
			Module:            module,
			Percent:           percent,
			Packages:          len(md.packages),
			CoveredStatements: md.covered,
			TotalStatements:   md.total,
		})
	}

	// Sort modules deterministically by module name
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Module < modules[j].Module
	})

	// Calculate total percent
	var totalPercent float64
	if totalStatements > 0 {
		totalPercent = float64(totalCovered) / float64(totalStatements) * 100
	}

	return &ProfileReport{
		SchemaVersion:   2,
		TotalPercent:    totalPercent,
		TotalCovered:    totalCovered,
		TotalStatements: totalStatements,
		Modules:         modules,
	}, nil
}

// ToJSON serializes a ProfileReport to JSON.
func (r *ProfileReport) ToJSON() ([]byte, error) {
	// Round percentages to avoid floating point artifacts
	modules := make([]WeightedModuleSummary, len(r.Modules))
	for i, m := range r.Modules {
		modules[i] = WeightedModuleSummary{
			Module:            m.Module,
			Percent:           roundToOneDecimal(m.Percent),
			Packages:          m.Packages,
			CoveredStatements: m.CoveredStatements,
			TotalStatements:   m.TotalStatements,
		}
	}
	report := &ProfileReport{
		SchemaVersion:   r.SchemaVersion,
		TotalPercent:    roundToOneDecimal(r.TotalPercent),
		TotalCovered:    r.TotalCovered,
		TotalStatements: r.TotalStatements,
		Modules:         modules,
	}
	return json.MarshalIndent(report, "", "  ")
}

// ProfileReportToReport converts a ProfileReport to the legacy Report format.
// This is used for backward compatibility with CLI output.
func ProfileReportToReport(pr *ProfileReport) *Report {
	modules := make([]ModuleSummary, len(pr.Modules))
	for i, m := range pr.Modules {
		modules[i] = ModuleSummary{
			Module:   m.Module,
			Percent:  m.Percent,
			Packages: m.Packages,
		}
	}
	return &Report{
		SchemaVersion: pr.SchemaVersion,
		TotalPercent:  pr.TotalPercent,
		Modules:       modules,
	}
}

// IsZeroStatementBlock returns true if a block has zero statements.
func IsZeroStatementBlock(line string) bool {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return false
	}
	numStmts, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	return numStmts == 0
}

// CountProfileBlocks counts total and covered statements in a profile.
func CountProfileBlocks(profilePath string) (covered, total int, err error) {
	file, err := os.Open(profilePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		block, err := ParseProfileBlock(line)
		if err != nil || block == nil {
			continue
		}
		total += block.NumStatements
		if block.Count > 0 {
			covered += block.NumStatements
		}
	}
	return covered, total, scanner.Err()
}

// PrintModuleTable prints the module breakdown in a formatted table to stdout.
func (r *ProfileReport) PrintModuleTable() {
	fmt.Println("Coverage by module:")
	fmt.Println("module                  coverage")
	for _, m := range r.Modules {
		fmt.Printf("%-22s %.1f%%\n", m.Module, roundToOneDecimal(m.Percent))
	}
}

// PrintModuleTableTo prints the module breakdown in a formatted table to the given writer.
func (r *ProfileReport) PrintModuleTableTo(w io.Writer) {
	fmt.Fprintln(w, "Coverage by module:")
	fmt.Fprintln(w, "module                  coverage")
	for _, m := range r.Modules {
		fmt.Fprintf(w, "%-22s %.1f%%\n", m.Module, roundToOneDecimal(m.Percent))
	}
}
