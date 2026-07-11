package doctrinecompiler

import (
	"bytes"
	"fmt"
	"strings"
)

// SelectorPath is the canonical path to the project selector.
const SelectorPath = TargetPath(".factory/project.json")

// ProjectSelector is the decoded contents of `.factory/project.json`.
//
// The selector is target-owned: a repository may change its declared
// pack and profile by editing this file. The compiler treats unknown
// fields as a hard error.
type ProjectSelector struct {
	SchemaVersion int    `json:"schema_version"`
	Pack          string `json:"pack"`
	Profile       string `json:"profile"`
}

// readSelector decodes a project selector from disk and validates it
// strictly. It rejects unknown fields and any trailing data.
func readSelector(path string) (*ProjectSelector, error) {
	data, err := readFS(path)
	if err != nil {
		return nil, newError("verify", "selector", err.Error())
	}
	var sel ProjectSelector
	if err := strictDecode(bytes.NewReader(data), &sel); err != nil {
		return nil, newError("verify", "selector", err.Error())
	}
	if err := validateSelector(&sel); err != nil {
		return nil, err
	}
	return &sel, nil
}

// validateSelector enforces selector-level invariants.
func validateSelector(s *ProjectSelector) error {
	if s.SchemaVersion != LockSchemaVersion {
		return newError("verify", "selector.schema_version",
			fmt.Sprintf("unsupported selector schema_version %d", s.SchemaVersion))
	}
	if strings.TrimSpace(s.Pack) == "" {
		return newError("verify", "selector.pack", "empty pack")
	}
	if strings.TrimSpace(s.Profile) == "" {
		return newError("verify", "selector.profile", "empty profile")
	}
	return nil
}

// ResolveSelection picks the (pack, profile) pair to use, preferring
// the explicit CLI flag when supplied and otherwise loading the
// committed selector. When fallback is true, the explicit flag is
// optional and the selector is used; otherwise the explicit flag is
// required.
//
// The returned SelectorPath matches the actual on-disk selector.
func ResolveSelection(explicitPack, explicitProfile string, target string, fallback bool) (string, ProfileId, error) {
	if explicitProfile != "" {
		return "", ProfileId(explicitProfile), nil
	}
	if !fallback {
		return "", "", nil
	}
	resolver, err := NewResolver(target)
	if err != nil {
		return "", "", err
	}
	sel, err := readSelector(resolver.Resolve(SelectorPath))
	if err != nil {
		return "", "", err
	}
	return string(sel.Pack), ProfileId(sel.Profile), nil
}
