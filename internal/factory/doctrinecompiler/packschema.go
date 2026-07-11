package doctrinecompiler

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// PackSchema is the JSON wire format for a doctrine pack.
//
// Strict decoding rejects unknown fields. The schema is intentionally
// declarative; projection and validation behaviour live in ordinary Go
// code outside this file.
type PackSchema struct {
	SchemaVersion   int              `json:"schema_version"`
	PackID          string           `json:"pack_id"`
	PackVersion     string           `json:"pack_version"`
	CompilerVersion string           `json:"compiler_version"`
	Doctrines       []DoctrineSchema `json:"doctrines"`
	Profiles        []ProfileSchema  `json:"target_profiles"`
}

// DoctrineSchema is one doctrine entry in the pack inventory.
type DoctrineSchema struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

// ProfileSchema describes a single target profile.
type ProfileSchema struct {
	ID                string                   `json:"id"`
	Summary           string                   `json:"summary"`
	Outputs           []OutputSchema           `json:"outputs"`
	Seeds             []SeedSchema             `json:"seeds"`
	ObservedContracts []ObservedContractSchema `json:"observed_contracts"`
	FactorizeChecks   []FactorizeCheckSchema   `json:"factorize_checks"`
	ExtensionPoints   []ExtensionPointSchema   `json:"extension_points"`
	EnabledDoctrines  []string                 `json:"enabled_doctrines,omitempty"`
}

// OutputSchema is one managed output declaration.
type OutputSchema struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Ownership string `json:"ownership"`
	ContentID string `json:"content_id"`
}

// SeedSchema is one seeded (target-owned-after-creation) declaration.
type SeedSchema struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Ownership string `json:"ownership"`
	ContentID string `json:"content_id"`
}

// ObservedContractSchema is one runtime invariant to assert on the
// target repository.
type ObservedContractSchema struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Path       string `json:"path"`
	Target     string `json:"target,omitempty"`
	Dependency string `json:"dependency,omitempty"`
}

// FactorizeCheckSchema is one entry of the factorize chain.
type FactorizeCheckSchema struct {
	ID      string   `json:"id"`
	Command []string `json:"command"`
}

// ExtensionPointSchema is one target-owned extension point declared
// by the pack.
type ExtensionPointSchema struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DecodePack parses raw pack bytes into a PackSchema and validates
// structural invariants that must hold regardless of profile content:
//
//   - schema_version equals SupportedPackSchemaVersion
//   - pack_id, pack_version are non-empty
//   - doctrine ids are unique
//   - profile ids are unique
//
// Validation of profile internals (paths, ownership, references) is
// performed by validateSchema. Keeping the two passes separate makes
// failure reports clearer.
func DecodePack(data []byte) (*PackSchema, []byte, error) {
	var schema PackSchema
	if err := strictDecode(bytes.NewReader(data), &schema); err != nil {
		return nil, data, newError("decode", "pack", err.Error())
	}
	if err := validatePackSchema(&schema); err != nil {
		return nil, data, err
	}
	return &schema, data, nil
}

// validatePackSchema enforces the pack-level invariants.
func validatePackSchema(p *PackSchema) error {
	if p.SchemaVersion != int(SupportedPackSchemaVersion) {
		return newError("validate", "schema_version",
			fmt.Sprintf("unsupported schema_version %d (expected %d)",
				p.SchemaVersion, int(SupportedPackSchemaVersion)))
	}
	if strings.TrimSpace(p.PackID) == "" {
		return newError("validate", "pack_id", "empty pack_id")
	}
	if strings.TrimSpace(p.PackVersion) == "" {
		return newError("validate", "pack_version", "empty pack_version")
	}
	// compiler_version is optional. When empty, no compatibility
	// constraint is enforced at verify time.
	_ = p.CompilerVersion
	// Doctrines: non-empty, unique.
	if len(p.Doctrines) == 0 {
		return newError("validate", "doctrines", "doctrines must not be empty")
	}
	seen := make(map[string]struct{}, len(p.Doctrines))
	for _, d := range p.Doctrines {
		if strings.TrimSpace(d.ID) == "" {
			return newError("validate", "doctrine.id", "empty doctrine id")
		}
		if _, dup := seen[d.ID]; dup {
			return newError("validate", "doctrine.id",
				"duplicate doctrine id: "+d.ID)
		}
		seen[d.ID] = struct{}{}
	}
	// Profiles: unique, all referenced profile ids must match
	// declared profiles.
	if len(p.Profiles) == 0 {
		return newError("validate", "target_profiles", "target_profiles must not be empty")
	}
	pseen := make(map[string]struct{}, len(p.Profiles))
	for _, prof := range p.Profiles {
		if strings.TrimSpace(prof.ID) == "" {
			return newError("validate", "target_profile.id", "empty profile id")
		}
		if _, dup := pseen[prof.ID]; dup {
			return newError("validate", "target_profile.id",
				"duplicate profile id: "+prof.ID)
		}
		pseen[prof.ID] = struct{}{}
		if err := validateProfileSchema(p, &prof); err != nil {
			return err
		}
	}
	return nil
}

// validateProfileSchema enforces profile-level invariants. The parent
// pack is supplied so enabled-doctrine references can be cross-checked
// against the doctrine inventory.
func validateProfileSchema(parent *PackSchema, p *ProfileSchema) error {
	if parent == nil {
		return newError("validate", "profile", "nil parent pack")
	}
	_ = p // keep parameter list documented for clarity; below uses p
	// Outputs.
	if len(p.Outputs) == 0 {
		return newError("validate", "outputs",
			"profile must declare at least one output")
	}
	outIds := make(map[string]struct{}, len(p.Outputs))
	outPaths := make(map[TargetPath]struct{}, len(p.Outputs))
	for _, o := range p.Outputs {
		if strings.TrimSpace(o.ID) == "" {
			return newError("validate", "output.id", "empty output id")
		}
		if _, dup := outIds[o.ID]; dup {
			return newError("validate", "output.id",
				"duplicate output id: "+o.ID)
		}
		outIds[o.ID] = struct{}{}
		tp, err := NormalizeTargetPath(o.Path)
		if err != nil {
			return newError("validate", "output.path",
				fmt.Sprintf("output %q: %v", o.ID, err))
		}
		if _, dup := outPaths[tp]; dup {
			return newError("validate", "output.path",
				fmt.Sprintf("duplicate normalized output path: %s", tp))
		}
		outPaths[tp] = struct{}{}
		own, err := ParseOwnership(o.Ownership)
		if err != nil {
			return newError("validate", "output.ownership",
				fmt.Sprintf("output %q: %v", o.ID, err))
		}
		if own != OwnershipManaged {
			return newError("validate", "output.ownership",
				fmt.Sprintf("output %q: ownership must be 'managed'", o.ID))
		}
		if strings.TrimSpace(o.ContentID) == "" {
			return newError("validate", "output.content_id",
				fmt.Sprintf("output %q: empty content_id", o.ID))
		}
	}
	// Seeds.
	seedIds := make(map[string]struct{}, len(p.Seeds))
	seedPaths := make(map[TargetPath]struct{}, len(p.Seeds))
	for _, s := range p.Seeds {
		if strings.TrimSpace(s.ID) == "" {
			return newError("validate", "seed.id", "empty seed id")
		}
		if _, dup := seedIds[s.ID]; dup {
			return newError("validate", "seed.id",
				"duplicate seed id: "+s.ID)
		}
		seedIds[s.ID] = struct{}{}
		tp, err := NormalizeTargetPath(s.Path)
		if err != nil {
			return newError("validate", "seed.path",
				fmt.Sprintf("seed %q: %v", s.ID, err))
		}
		if _, dup := seedPaths[tp]; dup {
			return newError("validate", "seed.path",
				fmt.Sprintf("duplicate normalized seed path: %s", tp))
		}
		seedPaths[tp] = struct{}{}
		// Cross-check: seed path must not collide with output path.
		if _, dup := outPaths[tp]; dup {
			return newError("validate", "seed.path",
				fmt.Sprintf("seed path collides with output path: %s", tp))
		}
		own, err := ParseOwnership(s.Ownership)
		if err != nil {
			return newError("validate", "seed.ownership",
				fmt.Sprintf("seed %q: %v", s.ID, err))
		}
		if own != OwnershipSeeded {
			return newError("validate", "seed.ownership",
				fmt.Sprintf("seed %q: ownership must be 'seeded'", s.ID))
		}
		if strings.TrimSpace(s.ContentID) == "" {
			return newError("validate", "seed.content_id",
				fmt.Sprintf("seed %q: empty content_id", s.ID))
		}
	}
	// Factorize checks.
	for _, c := range p.FactorizeChecks {
		if strings.TrimSpace(c.ID) == "" {
			return newError("validate", "factorize_check.id", "empty factorize_check id")
		}
		if len(c.Command) == 0 {
			return newError("validate", "factorize_check.command",
				fmt.Sprintf("factorize_check %q: empty command", c.ID))
		}
	}
	// Observed contracts.
	for _, c := range p.ObservedContracts {
		if strings.TrimSpace(c.ID) == "" {
			return newError("validate", "observed_contract.id",
				"empty observed_contract id")
		}
		kind := ObservedContractKind(c.Kind)
		switch kind {
		case ObservedMakefileInclude, ObservedMakefileTargetDep,
			ObservedFileExists, ObservedFileDigestEquals:
		default:
			return newError("validate", "observed_contract.kind",
				fmt.Sprintf("contract %q: unknown kind %q", c.ID, c.Kind))
		}
		if kind == ObservedMakefileInclude || kind == ObservedFileExists ||
			kind == ObservedFileDigestEquals {
			if strings.TrimSpace(c.Path) == "" {
				return newError("validate", "observed_contract.path",
					fmt.Sprintf("contract %q: empty path", c.ID))
			}
		}
		if kind == ObservedMakefileTargetDep {
			if strings.TrimSpace(c.Target) == "" || strings.TrimSpace(c.Dependency) == "" {
				return newError("validate", "observed_contract",
					fmt.Sprintf("contract %q: target and dependency required", c.ID))
			}
		}
	}
	// Extension points.
	for _, ep := range p.ExtensionPoints {
		if strings.TrimSpace(ep.ID) == "" {
			return newError("validate", "extension_point.id",
				"empty extension_point id")
		}
		if strings.TrimSpace(ep.Kind) == "" {
			return newError("validate", "extension_point.kind",
				fmt.Sprintf("extension_point %q: empty kind", ep.ID))
		}
	}
	// Enabled doctrines: must reference declared doctrines, no
	// duplicates, no empty ids.
	known := make(map[string]struct{}, len(parent.Doctrines))
	for _, d := range parent.Doctrines {
		known[d.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(p.EnabledDoctrines))
	for _, did := range p.EnabledDoctrines {
		if strings.TrimSpace(did) == "" {
			return newError("validate", "enabled_doctrines",
				"empty doctrine reference")
		}
		if _, ok := known[did]; !ok {
			return newError("validate", "enabled_doctrines",
				fmt.Sprintf("unknown doctrine reference %q", did))
		}
		if _, dup := seen[did]; dup {
			return newError("validate", "enabled_doctrines",
				fmt.Sprintf("duplicate doctrine reference %q", did))
		}
		seen[did] = struct{}{}
	}
	return nil
}

// sortProfileSchema is a helper for tests that need a canonical
// comparison shape. It is not used by the decoder.
func sortProfileSchema(_ *ProfileSchema) {
	// Reserved for future profile-level sorting hooks. Profiles are
	// already validated and re-emitted deterministically elsewhere.
	_ = sort.Strings
}
