package coverage

import (
	"fmt"
	"strings"
)

// ModuleSummary represents coverage summary for a single module.
type ModuleSummary struct {
	Module            string  `json:"module"`
	Percent           float64 `json:"percent"`
	Packages          int     `json:"packages"`
	CoveredStatements int     `json:"covered_statements,omitempty"`
	TotalStatements   int     `json:"total_statements,omitempty"`
}

// Report represents the coverage report with module breakdown.
type Report struct {
	SchemaVersion int             `json:"schema_version"`
	TotalPercent  float64         `json:"total_percent"`
	Modules       []ModuleSummary `json:"modules"`
}

// ClassifyModule maps an import path to a module name.
func ClassifyModule(importPath string) string {
	// Map import path prefix to module name
	switch {
	case strings.HasPrefix(importPath, "github.com/s1onique/leamas/cmd/leamas"):
		return "cmd/leamas"
	case strings.HasPrefix(importPath, "github.com/s1onique/leamas/internal/factory"):
		return "internal/factory"
	case strings.HasPrefix(importPath, "github.com/s1onique/leamas/internal/hulk"):
		return "internal/hulk"
	case strings.HasPrefix(importPath, "github.com/s1onique/leamas/internal/witness"):
		return "internal/witness"
	case strings.HasPrefix(importPath, "github.com/s1onique/leamas/internal/web"):
		return "internal/web"
	default:
		return "other"
	}
}

// roundToOneDecimal rounds a float to one decimal place.
func roundToOneDecimal(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}

// ToJSON serializes a Report to JSON.
func (r *Report) ToJSON() ([]byte, error) {
	modules := make([]ModuleSummary, len(r.Modules))
	for i, m := range r.Modules {
		modules[i] = ModuleSummary{
			Module:            m.Module,
			Percent:           roundToOneDecimal(m.Percent),
			Packages:          m.Packages,
			CoveredStatements: m.CoveredStatements,
			TotalStatements:   m.TotalStatements,
		}
	}
	report := &Report{
		SchemaVersion: r.SchemaVersion,
		TotalPercent:  roundToOneDecimal(r.TotalPercent),
		Modules:       modules,
	}
	return formatJSON(report)
}

// formatJSON formats data as indented JSON.
func formatJSON(v interface{}) ([]byte, error) {
	// Simple JSON formatting without external dependencies
	var sb strings.Builder
	formatValue(&sb, v, 0)
	return []byte(sb.String()), nil
}

func formatValue(sb *strings.Builder, v interface{}, indent int) {
	prefix := strings.Repeat("  ", indent)
	switch val := v.(type) {
	case map[string]interface{}:
		sb.WriteString("{\n")
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		for i, k := range keys {
			sb.WriteString(prefix + "  \"" + k + "\": ")
			formatValue(sb, val[k], indent+1)
			if i < len(keys)-1 {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
		}
		sb.WriteString(prefix + "}")
	case []interface{}:
		sb.WriteString("[\n")
		for i, item := range val {
			sb.WriteString(prefix + "  ")
			formatValue(sb, item, indent+1)
			if i < len(val)-1 {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
		}
		sb.WriteString(prefix + "]")
	case string:
		sb.WriteString("\"" + val + "\"")
	case float64:
		sb.WriteString(fmt.Sprintf("%.1f", val))
	case int:
		sb.WriteString(fmt.Sprintf("%d", val))
	case bool:
		if val {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	default:
		sb.WriteString("null")
	}
}

// PrintModuleTable prints the module breakdown in a formatted table.
func (r *Report) PrintModuleTable() {
	fmt.Println("Coverage by module:")
	fmt.Println("module                  coverage")
	for _, m := range r.Modules {
		fmt.Printf("%-22s %.1f%%\n", m.Module, roundToOneDecimal(m.Percent))
	}
}

// Threshold represents coverage threshold configuration.
type Threshold struct {
	MinTotalPercent float64
}

// Summary represents a simple coverage summary.
type Summary struct {
	TotalPercent float64
}

// Error represents a coverage error with kind and message.
type Error struct {
	Kind    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// Analyze runs full coverage analysis with threshold checking.
func Analyze(profilePath string, threshold *Threshold) (*Report, error) {
	profile, err := ParseProfile(profilePath)
	if err != nil {
		return nil, err
	}

	report := ProfileReportToReport(profile)

	if err := CheckThreshold(report, threshold); err != nil {
		return nil, err
	}

	return report, nil
}

// CheckThreshold checks if coverage meets the threshold.
func CheckThreshold(report *Report, threshold *Threshold) error {
	if report.TotalPercent < threshold.MinTotalPercent {
		return &Error{
			Kind: "threshold_fail",
			Message: fmt.Sprintf("total coverage %.1f%% is below minimum %.1f%%",
				report.TotalPercent, threshold.MinTotalPercent),
		}
	}
	return nil
}
