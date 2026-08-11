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
// The parser is strict: missing markers, unknown keys, duplicate
// keys, invalid enum values, missing required keys, and malformed
// key=value pairs all fail closed with a parser error.

package agentcontext

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// DelegationMode enumerates how a capability is delegated. The
// parser rejects any value not in this enum.
type DelegationMode string

const (
	// ModeAct means the capability is delegated by the current ACT.
	ModeAct DelegationMode = "act"
	// ModeExplicit means the capability requires explicit ACT
	// delegation per invocation.
	ModeExplicit DelegationMode = "explicit"
	// ModeExplicitExact means the capability requires the exact
	// command or lane to be authorized by the current ACT.
	ModeExplicitExact DelegationMode = "explicit_exact"
	// ModeForbidden means the capability is forbidden by persistent
	// context, regardless of ACT delegation.
	ModeForbidden DelegationMode = "forbidden"
)

// AllDelegationModes is the set of legal DelegationMode values.
var AllDelegationModes = map[DelegationMode]struct{}{
	ModeAct:           {},
	ModeExplicit:      {},
	ModeExplicitExact: {},
	ModeForbidden:     {},
}

// ValidDelegationMode reports whether m is a recognized mode.
func ValidDelegationMode(m DelegationMode) bool {
	_, ok := AllDelegationModes[m]
	return ok
}

// AuthorityContract is the structured representation of the
// LEAMAS:AUTHORITY-CONTRACT block parsed from a persistent agent
// context file.
//
// The fields are intentionally narrow: authority_source identifies
// where authority comes from; the six DelegationMode fields encode
// how each capability is delegated; the boolean fields encode the
// checkpoint vs Git-publication distinction and the frozen-plan
// authority exception; and UnauthorizedExpensiveResult encodes the
// required result label for unauthorized Tier-3 commands.
type AuthorityContract struct {
	AuthoritySource            DelegationMode
	Edit                       DelegationMode
	FocusedVerification        DelegationMode
	AffectedVerification       DelegationMode
	ExpensiveVerification      DelegationMode
	Commit                     DelegationMode
	Push                       DelegationMode
	Tag                        DelegationMode
	HistoryRewrite             DelegationMode
	CheckpointIsGitPublication bool
	UnauthorizedExpensiveResult string
	FrozenPlanAuthority        bool
}

// FieldSpec describes one key in the contract block, including its
// expected value type and whether it is required.
type FieldSpec struct {
	Key      string
	Required bool
	Kind     string // "mode", "bool", "string"
}

// FieldSpecs is the canonical ordered list of allowed keys. The
// parser rejects any key not in this list.
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

// ContractMarkers are the BEGIN / END HTML comment markers that
// bracket a contract block.
const (
	ContractBeginMarker = "<!-- LEAMAS:AUTHORITY-CONTRACT:BEGIN -->"
	ContractEndMarker   = "<!-- LEAMAS:AUTHORITY-CONTRACT:END -->"
)

// ContractErrorKind classifies the failure mode returned by
// ParseContractBlock. It exists so callers and tests can distinguish
// structural failures from value-validation failures.
type ContractErrorKind string

const (
	ErrMissingBegin     ContractErrorKind = "missing_begin"
	ErrMissingEnd       ContractErrorKind = "missing_end"
	ErrInvertedMarkers  ContractErrorKind = "inverted_markers"
	ErrDuplicateKey     ContractErrorKind = "duplicate_key"
	ErrUnknownKey       ContractErrorKind = "unknown_key"
	ErrMissingRequired  ContractErrorKind = "missing_required_key"
	ErrInvalidModeValue ContractErrorKind = "invalid_mode_value"
	ErrInvalidBoolValue ContractErrorKind = "invalid_bool_value"
	ErrMalformedLine    ContractErrorKind = "malformed_line"
)

// ContractError is the structured error returned by ParseContractBlock.
// It implements the error interface.
type ContractError struct {
	Kind ContractErrorKind
	Line int    // 1-based line number within the contract block; 0 if not line-specific
	Key  string // the offending key, when applicable
	Msg  string
}

// Error returns the human-readable message.
func (e *ContractError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("contract: %s (line %d): %s", e.Kind, e.Line, e.Msg)
	}
	if e.Key != "" {
		return fmt.Sprintf("contract: %s: %s (key=%s)", e.Kind, e.Msg, e.Key)
	}
	return fmt.Sprintf("contract: %s: %s", e.Kind, e.Msg)
}

// ParseContractBlock parses a single LEAMAS:AUTHORITY-CONTRACT block
// from raw text. It returns a non-nil *AuthorityContract and nil
// error on success, or nil and a *ContractError on any failure.
//
// The parser is strict:
//   - exactly one BEGIN and one END marker are required;
//   - exactly the keys listed in FieldSpecs are accepted;
//   - duplicate keys are rejected;
//   - missing required keys are rejected;
//   - invalid mode values are rejected;
//   - invalid bool values are rejected.
// ParseContractBlock never returns a partial contract on failure.
func ParseContractBlock(content string) (*AuthorityContract, error) {
	beginIdx := strings.Index(content, ContractBeginMarker)
	endIdx := strings.Index(content, ContractEndMarker)

	if beginIdx == -1 {
		return nil, &ContractError{Kind: ErrMissingBegin, Msg: "BEGIN marker not found"}
	}
	if endIdx == -1 {
		return nil, &ContractError{Kind: ErrMissingEnd, Msg: "END marker not found"}
	}
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
// of two contracts agree. "Shared" semantics are the fields whose
// values MUST agree between AGENTS.md and .clinerules/leamas.md; the
// .clinerules/leamas.md contract may be stricter for editor-only
// fields but MUST NOT weaken the shared ones.
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
// doctrinally required defaults. A persistent agent context file
// MUST express these values; any other value is a doctrinal defect.
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