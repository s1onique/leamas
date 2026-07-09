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

// Threshold represents coverage threshold configuration.
type Threshold struct {
	MinTotalPercent   float64
	MinModulePercents map[string]float64
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
// It checks both total and module thresholds.
// Module thresholds are checked in deterministic order.
func CheckThreshold(report *Report, threshold *Threshold) error {
	// Check total threshold first
	if report.TotalPercent < threshold.MinTotalPercent {
		return &Error{
			Kind: "threshold_fail",
			Message: fmt.Sprintf("total coverage %.1f%% is below minimum %.1f%%",
				report.TotalPercent, threshold.MinTotalPercent),
		}
	}

	// Build module lookup from report
	moduleMap := make(map[string]float64)
	for _, m := range report.Modules {
		moduleMap[m.Module] = m.Percent
	}

	// Deterministic order for module threshold checking
	moduleOrder := []string{
		"cmd/leamas",
		"internal/factory",
		"internal/hulk",
		"internal/web",
		"internal/witness",
		"other",
	}

	// Check each module threshold (only for modules in MinModulePercents)
	for _, moduleName := range moduleOrder {
		minPercent, hasThreshold := threshold.MinModulePercents[moduleName]
		if !hasThreshold {
			continue
		}

		actualPercent, exists := moduleMap[moduleName]
		if !exists {
			// Fail closed for missing enforced modules (except "other" which is report-only)
			return &Error{
				Kind: "module_threshold_fail",
				Message: fmt.Sprintf("module %s coverage is missing but minimum %.1f%% is required",
					moduleName, minPercent),
			}
		}

		if actualPercent < minPercent {
			return &Error{
				Kind: "module_threshold_fail",
				Message: fmt.Sprintf("module %s coverage %.1f%% is below minimum %.1f%%",
					moduleName, actualPercent, minPercent),
			}
		}
	}

	return nil
}
