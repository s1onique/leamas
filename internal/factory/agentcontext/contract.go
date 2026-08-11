// Authority contract: a structured key=value block that encodes the
// persistent agent authority-delegation contract as typed values
// rather than as prose substrings.
//
// The contract is a small canonical Markdown comment block placed in
// both AGENTS.md and .clinerules/leamas.md:
//
//   <!-- LEAMAS:AUTHORITY-CONTRACT:BEGIN -->
//   authority_source=current_act
//   edit=act
//   ...
//   <!-- LEAMAS:AUTHORITY-CONTRACT:END -->
//
// The parser is strict: each persistent agent context file MUST
// contain exactly one BEGIN and one END marker. Multiple contract
// blocks, missing markers, unknown keys, duplicate keys, invalid
// enum values, missing required keys, and malformed key=value pairs
// all fail closed with a parser error.

package agentcontext

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// DelegationMode enumerates how a capability is delegated.
type DelegationMode string

const (
	ModeAct           DelegationMode = "act"
	ModeExplicit      DelegationMode = "explicit"
	ModeExplicitExact DelegationMode = "explicit_exact"
	ModeForbidden     DelegationMode = "forbidden"
)

var AllDelegationModes = map[DelegationMode]struct{}{
	ModeAct:           {},
	ModeExplicit:      {},
	ModeExplicitExact: {},
	ModeForbidden:     {},
}

func ValidDelegationMode(m DelegationMode) bool {
	_, ok := AllDelegationModes[m]
	return ok
}

// AuthorityContract is the structured representation of the
// LEAMAS:AUTHORITY-CONTRACT block.
type AuthorityContract struct {
	AuthoritySource             DelegationMode
	Edit                        DelegationMode
	FocusedVerification         DelegationMode
	AffectedVerification        DelegationMode
	ExpensiveVerification       DelegationMode
	Commit                      DelegationMode
	Push                        DelegationMode
	Tag                         DelegationMode
	HistoryRewrite              DelegationMode
	CheckpointIsGitPublication  bool
	UnauthorizedExpensiveResult string
	FrozenPlanAuthority         bool
}

type FieldSpec struct {
	Key      string
	Required bool
	Kind     string // "mode", "bool", "string"
}

var FieldSpecs = []FieldSpec{
	{Key: "authority_source", Required: true, Kind: "mode"},
	{Key: "edit", Required: true, Kind: "mode"},
	{Key: "focused_verification", Required: true, Kind: "mode"},
	{Key: "affected_verification", Required: true, Kind: "mode"},
	{Key: "expensive_verification", Required: true, Kind: "mode"},
	{Key: "commit", Required: true, Kind: "mode"},
	{Key: "push", Required: true, Kind: "mode"},
	{Key: "tag", Required: true, Kind: "mode"},
	{Key: "history_rewrite", Required: true, Kind: "mode"},
	{Key: "checkpoint_is_git_publication", Required: true, Kind: "bool"},
	{Key: "unauthorized_expensive_result", Required: true, Kind: "string"},
	{Key: "frozen_closure_plan_authority", Required: true, Kind: "bool"},
}

const (
	ContractBeginMarker = "<!-- LEAMAS:AUTHORITY-CONTRACT:BEGIN -->"
	ContractEndMarker   = "<!-- LEAMAS:AUTHORITY-CONTRACT:END -->"
)

type ContractErrorKind string

const (
	ErrMissingBegin     ContractErrorKind = "missing_begin"
	ErrMissingEnd       ContractErrorKind = "missing_end"
	ErrDuplicateBegin   ContractErrorKind = "duplicate_begin"
	ErrDuplicateEnd     ContractErrorKind = "duplicate_end"
	ErrMultipleBlocks   ContractErrorKind = "multiple_contract_blocks"
	ErrInvertedMarkers  ContractErrorKind = "inverted_markers"
	ErrDuplicateKey     ContractErrorKind = "duplicate_key"
	ErrUnknownKey       ContractErrorKind = "unknown_key"
	ErrMissingRequired  ContractErrorKind = "missing_required_key"
	ErrInvalidModeValue ContractErrorKind = "invalid_mode_value"
	ErrInvalidBoolValue ContractErrorKind = "invalid_bool_value"
	ErrMalformedLine    ContractErrorKind = "malformed_line"
)

type ContractError struct {
	Kind ContractErrorKind
	Line int
	Key  string
	Msg  string
}

func (e *ContractError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("contract: %s (line %d): %s", e.Kind, e.Line, e.Msg)
	}
	if e.Key != "" {
		return fmt.Sprintf("contract: %s: %s (key=%s)", e.Kind, e.Msg, e.Key)
	}
	return fmt.Sprintf("contract: %s: %s", e.Kind, e.Msg)
}

// CountMarkers returns the number of BEGIN and END markers in
// content. Exported for tests.
func CountMarkers(content string) (beginCount, endCount int) {
	beginCount = strings.Count(content, ContractBeginMarker)
	endCount = strings.Count(content, ContractEndMarker)
	return
}

// findMarkerIndices returns the indices of all BEGIN and END markers
// in content. Returns nil if either marker is absent.
func findMarkerIndices(content string) (begins, ends []int) {
	i := 0
	for {
		idx := strings.Index(content[i:], ContractBeginMarker)
		if idx == -1 {
			break
		}
		begins = append(begins, i+idx)
		i += idx + len(ContractBeginMarker)
	}
	i = 0
	for {
		idx := strings.Index(content[i:], ContractEndMarker)
		if idx == -1 {
			break
		}
		ends = append(ends, i+idx)
		i += idx + len(ContractEndMarker)
	}
	return
}

// ParseContractBlock parses a single LEAMAS:AUTHORITY-CONTRACT block
// from raw text. The parser is strict: exactly one BEGIN and one END
// marker must exist; the BEGIN must precede the END; exactly the
// keys listed in FieldSpecs are accepted; duplicate keys are
// rejected; missing required keys are rejected; invalid mode
// values are rejected; invalid bool values are rejected.
func ParseContractBlock(content string) (*AuthorityContract, error) {
	begins, ends := findMarkerIndices(content)

	if len(begins) == 0 {
		return nil, &ContractError{Kind: ErrMissingBegin, Msg: "BEGIN marker not found"}
	}
	if len(ends) == 0 {
		return nil, &ContractError{Kind: ErrMissingEnd, Msg: "END marker not found"}
	}
	if len(begins) > 1 {
		return nil, &ContractError{
			Kind: ErrDuplicateBegin,
			Msg:  fmt.Sprintf("found %d BEGIN markers; exactly one is required", len(begins)),
		}
	}
	if len(ends) > 1 {
		return nil, &ContractError{
			Kind: ErrDuplicateEnd,
			Msg:  fmt.Sprintf("found %d END markers; exactly one is required", len(ends)),
		}
	}

	beginIdx := begins[0]
	endIdx := ends[0]

	if endIdx <= beginIdx {
		return nil, &ContractError{Kind: ErrInvertedMarkers, Msg: "END marker precedes BEGIN marker"}
	}

	block := content[beginIdx+len(ContractBeginMarker) : endIdx]

	// Parse line by line.
	parsed := make(map[string]string, len(FieldSpecs))
	seen := make(map[string]bool, len(FieldSpecs))
	scanner := bufio.NewScanner(bytes.NewReader([]byte(block)))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			return nil, &ContractError{
				Kind: ErrMalformedLine,
				Line: lineNum,
				Msg:  fmt.Sprintf("expected key=value, got %q", raw),
			}
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		if seen[key] {
			return nil, &ContractError{
				Kind: ErrDuplicateKey,
				Line: lineNum,
				Key:  key,
				Msg:  "key appears more than once",
			}
		}
		seen[key] = true
		parsed[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, &ContractError{Kind: ErrMalformedLine, Msg: err.Error()}
	}

	// Validate every key.
	for _, spec := range FieldSpecs {
		raw, present := parsed[spec.Key]
		if !present {
			return nil, &ContractError{
				Kind: ErrMissingRequired,
				Key:  spec.Key,
				Msg:  "required key is missing",
			}
		}
		switch spec.Kind {
		case "mode":
			if !ValidDelegationMode(DelegationMode(raw)) {
				return nil, &ContractError{
					Kind: ErrInvalidModeValue,
					Key:  spec.Key,
					Msg:  fmt.Sprintf("invalid mode value %q", raw),
				}
			}
		case "bool":
			if _, err := strconv.ParseBool(raw); err != nil {
				return nil, &ContractError{
					Kind: ErrInvalidBoolValue,
					Key:  spec.Key,
					Msg:  fmt.Sprintf("invalid bool value %q", raw),
				}
			}
		case "string":
			if raw == "" {
				return nil, &ContractError{
					Kind: ErrInvalidModeValue,
					Key:  spec.Key,
					Msg:  "string value must be non-empty",
				}
			}
		}
	}

	// Reject any unknown key.
	allowed := make(map[string]struct{}, len(FieldSpecs))
	for _, spec := range FieldSpecs {
		allowed[spec.Key] = struct{}{}
	}
	extraKeys := make([]string, 0)
	for k := range parsed {
		if _, ok := allowed[k]; !ok {
			extraKeys = append(extraKeys, k)
		}
	}
	if len(extraKeys) > 0 {
		sort.Strings(extraKeys)
		return nil, &ContractError{
			Kind: ErrUnknownKey,
			Key:  extraKeys[0],
			Msg:  fmt.Sprintf("unknown key %q (and %d more)", extraKeys[0], len(extraKeys)-1),
		}
	}

	// Build typed contract.
	contract := &AuthorityContract{
		AuthoritySource:             DelegationMode(parsed["authority_source"]),
		Edit:                        DelegationMode(parsed["edit"]),
		FocusedVerification:         DelegationMode(parsed["focused_verification"]),
		AffectedVerification:        DelegationMode(parsed["affected_verification"]),
		ExpensiveVerification:       DelegationMode(parsed["expensive_verification"]),
		Commit:                      DelegationMode(parsed["commit"]),
		Push:                        DelegationMode(parsed["push"]),
		Tag:                         DelegationMode(parsed["tag"]),
		HistoryRewrite:              DelegationMode(parsed["history_rewrite"]),
		UnauthorizedExpensiveResult: parsed["unauthorized_expensive_result"],
	}
	if v, err := strconv.ParseBool(parsed["checkpoint_is_git_publication"]); err == nil {
		contract.CheckpointIsGitPublication = v
	}
	if v, err := strconv.ParseBool(parsed["frozen_closure_plan_authority"]); err == nil {
		contract.FrozenPlanAuthority = v
	}
	return contract, nil
}

// SharedSemanticsEqual reports whether the shared authority semantics
// of two contracts agree.
func SharedSemanticsEqual(a, b *AuthorityContract) bool {
	if a == nil || b == nil {
		return false
	}
	return a.AuthoritySource == b.AuthoritySource &&
		a.Edit == b.Edit &&
		a.FocusedVerification == b.FocusedVerification &&
		a.AffectedVerification == b.AffectedVerification &&
		a.ExpensiveVerification == b.ExpensiveVerification &&
		a.Commit == b.Commit &&
		a.Push == b.Push &&
		a.Tag == b.Tag &&
		a.HistoryRewrite == b.HistoryRewrite &&
		a.CheckpointIsGitPublication == b.CheckpointIsGitPublication &&
		a.UnauthorizedExpensiveResult == b.UnauthorizedExpensiveResult &&
		a.FrozenPlanAuthority == b.FrozenPlanAuthority
}

// IsValidContractSemantics checks that the contract values match the
// doctrinally required defaults.
func (c *AuthorityContract) IsValidContractSemantics() bool {
	if c == nil {
		return false
	}
	return c.AuthoritySource == ModeAct &&
		c.Edit == ModeAct &&
		c.FocusedVerification == ModeAct &&
		c.AffectedVerification == ModeAct &&
		c.ExpensiveVerification == ModeExplicitExact &&
		c.Commit == ModeExplicit &&
		c.Push == ModeExplicit &&
		c.Tag == ModeExplicit &&
		c.HistoryRewrite == ModeForbidden &&
		!c.CheckpointIsGitPublication &&
		c.UnauthorizedExpensiveResult == "NOT_RUN" &&
		c.FrozenPlanAuthority
}
