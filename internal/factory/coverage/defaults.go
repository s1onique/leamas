package coverage

// DefaultMinTotalPercent is the canonical default total coverage threshold.
const DefaultMinTotalPercent = 64.0

// DefaultModuleFloorOrder is the deterministic order for module floor operations.
var DefaultModuleFloorOrder = []string{
	"cmd/leamas",
	"internal/factory",
	"internal/hulk",
	"internal/web",
	"internal/witness",
}

// defaultModuleFloors contains the canonical default module coverage floors.
var defaultModuleFloors = map[string]float64{
	"cmd/leamas":       50.0,
	"internal/factory": 67.0,
	"internal/hulk":    90.0,
	"internal/web":     70.0,
	"internal/witness": 80.0,
}

// DefaultModuleThresholds returns a defensive copy of the canonical default module floors.
// These values must match the Makefile COVERAGE_MIN_* variables.
func DefaultModuleThresholds() map[string]float64 {
	result := make(map[string]float64, len(defaultModuleFloors))
	for k, v := range defaultModuleFloors {
		result[k] = v
	}
	return result
}

// DefaultThreshold returns a Threshold with conservative default values.
// The total and module values are canonical and single-sourced.
func DefaultThreshold() *Threshold {
	return &Threshold{
		MinTotalPercent:   DefaultMinTotalPercent,
		MinModulePercents: DefaultModuleThresholds(),
	}
}

// KnownEnforcedModules returns the list of modules that can be enforced.
// Returns a defensive copy in deterministic order.
func KnownEnforcedModules() []string {
	result := make([]string, len(DefaultModuleFloorOrder))
	copy(result, DefaultModuleFloorOrder)
	return result
}

// IsKnownEnforcedModule returns true if the module is known and enforceable by default.
// "other" is not enforceable by default (report-only).
func IsKnownEnforcedModule(module string) bool {
	_, exists := defaultModuleFloors[module]
	return exists
}
