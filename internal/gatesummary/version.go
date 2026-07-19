package gatesummary

import (
	"encoding/json"
	"regexp"
)

// versionDecision is the Stage 4 + Stage 5 outcome.
type versionDecision struct {
	version Version
	raw     string
	code    string
}

// integerLexicalRe matches the JSON integer lexical form
//
//	^-?(0|[1-9][0-9]*)$
//
// as required by gate-summary-schema-version-translation.md §2.
var integerLexicalRe = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

// classifyVersion consumes the schema_version json.Token captured by
// Stage 4 and returns the dispatch decision. It performs lexical,
// type, and value classification without narrowing to a machine-sized
// integer.
func classifyVersion(tok json.Token) versionDecision {
	switch v := tok.(type) {
	case nil:
		return versionDecision{code: CodeInvalidVersionType, raw: "null"}
	case string, bool, json.Delim:
		return versionDecision{code: CodeInvalidVersionType, raw: encodeTokenAsString(v)}
	case json.Number:
		s := v.String()
		if !integerLexicalRe.MatchString(s) {
			return versionDecision{code: CodeInvalidVersionType, raw: s}
		}
		if s == "1" {
			return versionDecision{version: Version1, raw: s}
		}
		if s == "2" {
			return versionDecision{version: Version2, raw: s}
		}
		return versionDecision{code: CodeUnsupportedVersion, raw: s}
	}
	return versionDecision{code: CodeInvalidVersionType, raw: ""}
}

// encodeTokenAsString returns a deterministic string spelling for
// diagnostic purposes.
func encodeTokenAsString(tok json.Token) string {
	switch v := tok.(type) {
	case json.Number:
		return v.String()
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	case json.Delim:
		if v == '{' {
			return "{"
		}
		return "["
	}
	return ""
}
