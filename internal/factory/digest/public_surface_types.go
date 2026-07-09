// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"fmt"
)

// symbolKey represents a unique symbol identifier.
type symbolKey struct {
	Package  string
	Name     string
	Kind     string // "func", "type", "const", "var", "method", "field"
	Receiver string // for methods
}

// String returns a canonical string for the symbol.
func (sk symbolKey) String() string {
	if sk.Receiver != "" {
		return fmt.Sprintf("%s.%s(%s)", sk.Name, sk.Kind, sk.Receiver)
	}
	return fmt.Sprintf("%s(%s)", sk.Name, sk.Kind)
}

// symbolKeyString converts a symbol key to its string representation.
func symbolKeyString(sk symbolKey) string {
	return sk.String()
}

// deduplicateStrings removes duplicate strings from a slice.
func deduplicateStrings(strs []string) []string {
	if len(strs) <= 1 {
		return strs
	}
	seen := make(map[string]bool)
	result := make([]string, 0, len(strs))
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
