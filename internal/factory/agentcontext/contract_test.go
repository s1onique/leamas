package agentcontext

import (
	"strings"
	"testing"
)

// minimalValidBlock returns a contract block containing every
// required field with the doctrinally valid value.
func minimalValidBlock() string {
	return ContractBeginMarker + "\n" +
		"authority_source=act\n" +
		"edit=act\n" +
		"focused_verification=act\n" +
		"affected_verification=act\n" +
		"expensive_verification=explicit_exact\n" +
		"commit=explicit\n" +
		"push=explicit\n" +
		"tag=explicit\n" +
		"history_rewrite=forbidden\n" +
		"checkpoint_is_git_publication=false\n" +
		"unauthorized_expensive_result=NOT_RUN\n" +
		"frozen_closure_plan_authority=true\n" +
		ContractEndMarker + "\n"
}

func TestParseContractBlock_Valid(t *testing.T) {
	c, err := ParseContractBlock(minimalValidBlock())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.IsValidContractSemantics() {
		t.Fatalf("expected valid contract semantics")
	}
}

func TestParseContractBlock_MissingBegin(t *testing.T) {
	body := strings.ReplaceAll(minimalValidBlock(), ContractBeginMarker, "")
	if _, err := ParseContractBlock(body); err == nil {
		t.Fatalf("expected error for missing BEGIN marker")
	} else if cerr, ok := err.(*ContractError); !ok || cerr.Kind != ErrMissingBegin {
		t.Fatalf("expected ErrMissingBegin, got %v", err)
	}
}

func TestParseContractBlock_MissingEnd(t *testing.T) {
	body := strings.ReplaceAll(minimalValidBlock(), ContractEndMarker, "")
	if _, err := ParseContractBlock(body); err == nil {
		t.Fatalf("expected error for missing END marker")
	} else if cerr, ok := err.(*ContractError); !ok || cerr.Kind != ErrMissingEnd {
		t.Fatalf("expected ErrMissingEnd, got %v", err)
	}
}

func TestParseContractBlock_DuplicateKey(t *testing.T) {
	body := strings.ReplaceAll(minimalValidBlock(),
		"commit=explicit\n",
		"commit=explicit\ncommit=explicit\n",
	)
	if _, err := ParseContractBlock(body); err == nil {
		t.Fatalf("expected error for duplicate key")
	} else if cerr, ok := err.(*ContractError); !ok || cerr.Kind != ErrDuplicateKey {
		t.Fatalf("expected ErrDuplicateKey, got %v", err)
	}
}

func TestParseContractBlock_UnknownKey(t *testing.T) {
	body := strings.ReplaceAll(minimalValidBlock(),
		"commit=explicit\n",
		"commit=explicit\nrogue_field=true\n",
	)
	if _, err := ParseContractBlock(body); err == nil {
		t.Fatalf("expected error for unknown key")
	} else if cerr, ok := err.(*ContractError); !ok || cerr.Kind != ErrUnknownKey {
		t.Fatalf("expected ErrUnknownKey, got %v", err)
	}
}

func TestParseContractBlock_MissingRequired(t *testing.T) {
	body := strings.ReplaceAll(minimalValidBlock(), "commit=explicit\n", "")
	if _, err := ParseContractBlock(body); err == nil {
		t.Fatalf("expected error for missing required key")
	} else if cerr, ok := err.(*ContractError); !ok || cerr.Kind != ErrMissingRequired {
		t.Fatalf("expected ErrMissingRequired, got %v", err)
	}
}

func TestParseContractBlock_InvalidMode(t *testing.T) {
	body := strings.ReplaceAll(minimalValidBlock(), "commit=explicit\n", "commit=always\n")
	if _, err := ParseContractBlock(body); err == nil {
		t.Fatalf("expected error for invalid mode value")
	} else if cerr, ok := err.(*ContractError); !ok || cerr.Kind != ErrInvalidModeValue {
		t.Fatalf("expected ErrInvalidModeValue, got %v", err)
	}
}

func TestParseContractBlock_InvalidBool(t *testing.T) {
	body := strings.ReplaceAll(minimalValidBlock(), "checkpoint_is_git_publication=false\n", "checkpoint_is_git_publication=nope\n")
	if _, err := ParseContractBlock(body); err == nil {
		t.Fatalf("expected error for invalid bool value")
	} else if cerr, ok := err.(*ContractError); !ok || cerr.Kind != ErrInvalidBoolValue {
		t.Fatalf("expected ErrInvalidBoolValue, got %v", err)
	}
}

func TestParseContractBlock_MalformedLine(t *testing.T) {
	body := strings.ReplaceAll(minimalValidBlock(),
		"commit=explicit\n",
		"this_is_not_a_valid_line\n",
	)
	if _, err := ParseContractBlock(body); err == nil {
		t.Fatalf("expected error for malformed line")
	} else if cerr, ok := err.(*ContractError); !ok || cerr.Kind != ErrMalformedLine {
		t.Fatalf("expected ErrMalformedLine, got %v", err)
	}
}

func TestParseContractBlock_DoctorinalDefaultsRejected(t *testing.T) {
	wrong := strings.ReplaceAll(minimalValidBlock(),
		"expensive_verification=explicit_exact\n", "expensive_verification=act\n")
	c, err := ParseContractBlock(wrong)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if c.IsValidContractSemantics() {
		t.Fatalf("expected contract to be rejected as doctrinally invalid")
	}
}

func TestParseContractBlock_AllDoctrinalValuesRejected(t *testing.T) {
	// Each doctrinal violation must be rejected by EITHER the parser
	// (invalid enum) or the semantic check (parsed-but-wrong).
	violations := []struct {
		Name    string
		Key     string
		Replace string
		CheckAs string // "mode_invalid", "bool_semantic", "string_semantic"
	}{
		{"expensive_verification=always", "expensive_verification=explicit_exact\n", "expensive_verification=always\n", "mode_invalid"},
		{"commit=implicit", "commit=explicit\n", "commit=implicit\n", "mode_invalid"},
		{"push=automatic", "push=explicit\n", "push=automatic\n", "mode_invalid"},
		{"tag=implicit", "tag=explicit\n", "tag=implicit\n", "mode_invalid"},
		{"history_rewrite=allowed", "history_rewrite=forbidden\n", "history_rewrite=allowed\n", "mode_invalid"},
		{"checkpoint_is_git_publication=true", "checkpoint_is_git_publication=false\n", "checkpoint_is_git_publication=true\n", "bool_semantic"},
		{"unauthorized_expensive_result=PASS", "unauthorized_expensive_result=NOT_RUN\n", "unauthorized_expensive_result=PASS\n", "string_semantic"},
		{"frozen_closure_plan_authority=false", "frozen_closure_plan_authority=true\n", "frozen_closure_plan_authority=false\n", "bool_semantic"},
	}
	for _, v := range violations {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			body := strings.ReplaceAll(minimalValidBlock(), v.Key, v.Replace)
			c, err := ParseContractBlock(body)
			switch v.CheckAs {
			case "mode_invalid":
				if err == nil {
					t.Fatalf("expected parse error for %s", v.Name)
				}
			case "bool_semantic", "string_semantic":
				if err != nil {
					t.Fatalf("unexpected parse error for %s: %v", v.Name, err)
				}
				if c.IsValidContractSemantics() {
					t.Fatalf("expected %s to be rejected as doctrinally invalid", v.Name)
				}
			}
		})
	}
}

func TestSharedSemanticsEqual(t *testing.T) {
	c, err := ParseContractBlock(minimalValidBlock())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !SharedSemanticsEqual(c, c) {
		t.Fatalf("contract must be equal to itself")
	}

	broken := strings.ReplaceAll(minimalValidBlock(), "commit=explicit\n", "commit=act\n")
	c2, err := ParseContractBlock(broken)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if SharedSemanticsEqual(c, c2) {
		t.Fatalf("expected inequality after commit mutation")
	}

	if SharedSemanticsEqual(c, nil) {
		t.Fatalf("expected inequality with nil")
	}
}

func TestParseContractBlock_DuplicateBegin(t *testing.T) {
	body := ContractBeginMarker + "\ncommit=explicit\n" + ContractBeginMarker + "\ncommit=act\n" + ContractEndMarker + "\n"
	_, err := ParseContractBlock(body)
	if err == nil {
		t.Fatalf("expected error for duplicate BEGIN marker")
	} else if cerr, ok := err.(*ContractError); !ok || cerr.Kind != ErrDuplicateBegin {
		t.Fatalf("expected ErrDuplicateBegin, got %v", err)
	}
}

func TestParseContractBlock_DuplicateEnd(t *testing.T) {
	body := ContractBeginMarker + "\ncommit=explicit\n" + ContractEndMarker + "\n" + ContractEndMarker + "\n"
	_, err := ParseContractBlock(body)
	if err == nil {
		t.Fatalf("expected error for duplicate END marker")
	} else if cerr, ok := err.(*ContractError); !ok || cerr.Kind != ErrDuplicateEnd {
		t.Fatalf("expected ErrDuplicateEnd, got %v", err)
	}
}

func TestParseContractBlock_TwoCompleteBlocks(t *testing.T) {
	body := ContractBeginMarker + "\ncommit=explicit\n" + ContractEndMarker +
		"\nprose\n" +
		ContractBeginMarker + "\ncommit=act\n" + ContractEndMarker + "\n"
	_, err := ParseContractBlock(body)
	if err == nil {
		t.Fatalf("expected error for two complete contract blocks")
	} else if cerr, ok := err.(*ContractError); !ok || cerr.Kind != ErrDuplicateBegin {
		t.Fatalf("expected ErrDuplicateBegin, got %v", err)
	}
}
