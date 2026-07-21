package exectest

import (
	"bytes"
	"os"
)

// mergeEnv merges provided environment variables with the current environment.
// Provided variables override current ones; other variables are inherited.
func mergeEnv(overrides []string) []string {
	base := os.Environ()
	if len(overrides) == 0 {
		return base
	}
	// Build a map from base environment
	env := make(map[string]string)
	for _, e := range base {
		if idx := bytes.IndexByte([]byte(e), '='); idx >= 0 {
			env[e[:idx]] = e[idx+1:]
		}
	}
	// Apply overrides
	for _, e := range overrides {
		if idx := bytes.IndexByte([]byte(e), '='); idx >= 0 {
			env[e[:idx]] = e[idx+1:]
		}
	}
	// Convert back to slice
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}
