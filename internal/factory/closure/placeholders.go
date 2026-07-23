package closure

import "strings"

var exactClosurePlaceholders = map[string]struct{}{
	"TBD":            {},
	"TODO":           {},
	"UNKNOWN":        {},
	"RUNNING":        {},
	"TO BE RECORDED": {},
}

var embeddedClosurePlaceholders = []string{
	"(SEE GIT REV-PARSE)",
	"<COMMIT>",
	"<TREE>",
	"<HASH>",
}

func containsClosurePlaceholder(value string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if _, found := exactClosurePlaceholders[normalized]; found {
		return true
	}
	for _, marker := range embeddedClosurePlaceholders {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
