package main

import (
	"os"
	"strings"

	"github.com/s1onique/leamas/internal/execution"
)

// withoutLeamasEnv returns a copy of the process environment with
// the Leamas re-entry markers removed. This prevents test subprocesses
// from being rejected as nested executions.
func withoutLeamasEnv() []string {
	blocked := map[string]struct{}{
		execution.EnvRootID:     {},
		execution.EnvParentPID:  {},
		execution.EnvGeneration: {},
	}

	env := os.Environ()
	out := make([]string, 0, len(env))

	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, remove := blocked[key]; !remove {
			out = append(out, entry)
		}
	}

	return out
}
