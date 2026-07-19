package gatesummary

import (
	"net/url"
	"slices"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

var testTotalAnyOfTokens = []string{"$defs", "check", "anyOf"}

// keywordIdentity returns the parsed base schema URL and the complete
// decoded keyword-token tuple. The schema fragment is an RFC 6901 pointer;
// ErrorKind.KeywordPath tokens identify the keyword inside that object.
func keywordIdentity(node *jsonschema.ValidationError) (string, []string) {
	if node == nil || node.ErrorKind == nil {
		return "", nil
	}
	base, fragmentTokens := parseSchemaURL(node.SchemaURL)
	keywordTokens := node.ErrorKind.KeywordPath()
	if _, ok := node.ErrorKind.(*kind.Not); ok {
		keywordTokens = []string{"not"}
	}
	tokens := make([]string, 0, len(fragmentTokens)+len(keywordTokens))
	tokens = append(tokens, fragmentTokens...)
	tokens = append(tokens, keywordTokens...)
	return base, tokens
}

func parseSchemaURL(raw string) (string, []string) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw, nil
	}
	fragment := parsed.Fragment
	parsed.Fragment = ""
	parsed.RawFragment = ""
	base := parsed.String()
	tokens, ok := decodePointerFragment(fragment)
	if !ok {
		return base, nil
	}
	return base, tokens
}

func decodePointerFragment(fragment string) ([]string, bool) {
	if fragment == "" {
		return nil, true
	}
	if fragment[0] != '/' {
		return nil, false
	}
	rawTokens := strings.Split(fragment[1:], "/")
	tokens := make([]string, len(rawTokens))
	for i, raw := range rawTokens {
		decoded, ok := decodePointerToken(raw)
		if !ok {
			return nil, false
		}
		tokens[i] = decoded
	}
	return tokens, true
}

func decodePointerToken(token string) (string, bool) {
	if !strings.Contains(token, "~") {
		return token, true
	}
	var b strings.Builder
	b.Grow(len(token))
	for i := 0; i < len(token); i++ {
		if token[i] != '~' {
			b.WriteByte(token[i])
			continue
		}
		if i+1 >= len(token) {
			return "", false
		}
		i++
		switch token[i] {
		case '0':
			b.WriteByte('~')
		case '1':
			b.WriteByte('/')
		default:
			return "", false
		}
	}
	return b.String(), true
}

func isTestTotalAnyOf(baseURL string, tokens []string) bool {
	return baseURL == v2SchemaID && slices.Equal(tokens, testTotalAnyOfTokens)
}

func instanceLocationToPointer(location []string) string {
	if len(location) == 0 {
		return ""
	}
	var b strings.Builder
	for _, token := range location {
		b.WriteByte('/')
		b.WriteString(escapePointer(token))
	}
	return b.String()
}

func isSchemaVersionLocation(location []string) bool {
	return slices.Equal(location, []string{"schema_version"})
}

func isGeneratedAtLocation(location []string) bool {
	return slices.Equal(location, []string{"generated_at"})
}

func isStatusLocation(location []string) bool {
	if len(location) == 1 {
		switch location[0] {
		case "overall_status", "scope_status", "parent_status":
			return true
		}
	}
	return len(location) == 3 && location[0] == "checks" &&
		isArrayIndexToken(location[1]) && location[2] == "status"
}

func isOIDLocation(location []string) bool {
	if len(location) != 1 {
		return false
	}
	switch location[0] {
	case "execution_head_oid", "execution_tree_oid", "subject_tree_oid":
		return true
	default:
		return false
	}
}

func isOutputHashLocation(location []string) bool {
	if len(location) != 4 || location[0] != "checks" ||
		!isArrayIndexToken(location[1]) || location[2] != "extras" {
		return false
	}
	return location[3] == "stdout_sha256" || location[3] == "stderr_sha256"
}

func isDurationLocation(location []string) bool {
	return len(location) == 4 && location[0] == "checks" &&
		isArrayIndexToken(location[1]) && location[2] == "extras" &&
		location[3] == "duration_ms"
}

func isArrayIndexToken(token string) bool {
	if token == "0" {
		return true
	}
	if token == "" || token[0] < '1' || token[0] > '9' {
		return false
	}
	for i := 1; i < len(token); i++ {
		if token[i] < '0' || token[i] > '9' {
			return false
		}
	}
	return true
}
