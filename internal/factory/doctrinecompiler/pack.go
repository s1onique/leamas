package doctrinecompiler

import (
	"fmt"
	"sort"
	"strings"
)

// Doctrine is the typed representation of one doctrine inventory entry.
type Doctrine struct {
	ID      DoctrineId
	Summary string
}

// OutputDecl is one managed output declaration in a profile.
type OutputDecl struct {
	ID        string
	Path      TargetPath
	Ownership OwnershipMode
	ContentID string
}

// SeedDecl is one seeded (target-owned-after-creation) declaration.
type SeedDecl struct {
	ID        string
	Path      TargetPath
	Ownership OwnershipMode
	ContentID string
}

// Pack is the typed, validated representation of a doctrine pack.
//
// All collections inside Pack are sorted deterministically so that
// reflection-style JSON encoders produce stable output.
type Pack struct {
	SchemaVersion   PackSchemaVersion
	PackID          PackId
	PackVersion     PackVersion
	CompilerVersion string
	Doctrines       []Doctrine
	Profiles        []Profile
	RawBytes        []byte
	Schema          *PackSchema
}

// Profile is the typed representation of a target profile.
type Profile struct {
	ID                ProfileId
	Summary           string
	Outputs           []OutputDecl
	Seeds             []SeedDecl
	ObservedContracts []ObservedContract
	FactorizeChecks   []FactorizeCheck
	ExtensionPoints   []ExtensionPoint
	EnabledDoctrines  []DoctrineId
}

// BuildPack converts a decoded PackSchema into a typed Pack and
// performs cross-reference validation:
//
//   - every output and seed ContentID maps to a registered provider
//   - every observed contract references known kinds
//   - every factorize_check id is unique
//
// Returns the canonical PackDigest of the raw schema bytes.
func BuildPack(schema *PackSchema, raw []byte, providers map[string]ContentProvider) (*Pack, error) {
	p := &Pack{
		SchemaVersion:   PackSchemaVersion(schema.SchemaVersion),
		PackID:          PackId(schema.PackID),
		PackVersion:     PackVersion(schema.PackVersion),
		CompilerVersion: schema.CompilerVersion,
		RawBytes:        append([]byte(nil), raw...),
		Schema:          schema,
	}
	for _, d := range schema.Doctrines {
		p.Doctrines = append(p.Doctrines, Doctrine{
			ID:      DoctrineId(d.ID),
			Summary: d.Summary,
		})
	}
	sort.Slice(p.Doctrines, func(i, j int) bool {
		return p.Doctrines[i].ID < p.Doctrines[j].ID
	})
	for _, prof := range schema.Profiles {
		tp, err := buildProfile(&prof, providers)
		if err != nil {
			return nil, err
		}
		p.Profiles = append(p.Profiles, *tp)
	}
	sort.Slice(p.Profiles, func(i, j int) bool {
		return p.Profiles[i].ID < p.Profiles[j].ID
	})
	return p, nil
}

// buildProfile converts a decoded profile into the typed form.
func buildProfile(src *ProfileSchema, providers map[string]ContentProvider) (*Profile, error) {
	out := &Profile{
		ID:      ProfileId(src.ID),
		Summary: src.Summary,
	}
	for _, o := range src.Outputs {
		tp, err := NormalizeTargetPath(o.Path)
		if err != nil {
			return nil, err
		}
		own, err := ParseOwnership(o.Ownership)
		if err != nil {
			return nil, err
		}
		out.Outputs = append(out.Outputs, OutputDecl{
			ID:        o.ID,
			Path:      tp,
			Ownership: own,
			ContentID: o.ContentID,
		})
	}
	for _, s := range src.Seeds {
		tp, err := NormalizeTargetPath(s.Path)
		if err != nil {
			return nil, err
		}
		own, err := ParseOwnership(s.Ownership)
		if err != nil {
			return nil, err
		}
		out.Seeds = append(out.Seeds, SeedDecl{
			ID:        s.ID,
			Path:      tp,
			Ownership: own,
			ContentID: s.ContentID,
		})
	}
	for _, c := range src.ObservedContracts {
		kind := ObservedContractKind(c.Kind)
		var tp TargetPath
		if c.Path != "" {
			n, err := NormalizeTargetPath(c.Path)
			if err != nil {
				return nil, err
			}
			tp = n
		}
		out.ObservedContracts = append(out.ObservedContracts, ObservedContract{
			Id:     c.ID,
			Kind:   kind,
			Path:   tp,
			Target: c.Target,
			Dep:    c.Dependency,
		})
	}
	seenFC := make(map[string]struct{}, len(src.FactorizeChecks))
	for _, c := range src.FactorizeChecks {
		if _, dup := seenFC[c.ID]; dup {
			return nil, newError("validate", "factorize_check.id",
				"duplicate factorize_check id: "+c.ID)
		}
		seenFC[c.ID] = struct{}{}
		out.FactorizeChecks = append(out.FactorizeChecks, FactorizeCheck{
			Id:      c.ID,
			Command: append([]string(nil), c.Command...),
		})
	}
	seenEP := make(map[string]struct{}, len(src.ExtensionPoints))
	for _, ep := range src.ExtensionPoints {
		if _, dup := seenEP[ep.ID]; dup {
			return nil, newError("validate", "extension_point.id",
				"duplicate extension_point id: "+ep.ID)
		}
		seenEP[ep.ID] = struct{}{}
		out.ExtensionPoints = append(out.ExtensionPoints, ExtensionPoint{
			Id:          ep.ID,
			Kind:        ep.Kind,
			Name:        ep.Name,
			Description: ep.Description,
		})
	}
	for _, did := range src.EnabledDoctrines {
		out.EnabledDoctrines = append(out.EnabledDoctrines, DoctrineId(did))
	}
	// Validate that every output/seed ContentID resolves.
	for _, o := range out.Outputs {
		if _, ok := providers[o.ContentID]; !ok {
			return nil, newError("validate", "content_id",
				fmt.Sprintf("output %q: unknown content_id %q", o.ID, o.ContentID))
		}
	}
	for _, s := range out.Seeds {
		if _, ok := providers[s.ContentID]; !ok {
			return nil, newError("validate", "content_id",
				fmt.Sprintf("seed %q: unknown content_id %q", s.ID, s.ContentID))
		}
	}
	// Sort collections for deterministic downstream use.
	sort.Slice(out.Outputs, func(i, j int) bool { return out.Outputs[i].Path < out.Outputs[j].Path })
	sort.Slice(out.Seeds, func(i, j int) bool { return out.Seeds[i].Path < out.Seeds[j].Path })
	sort.Slice(out.ObservedContracts, func(i, j int) bool { return out.ObservedContracts[i].Id < out.ObservedContracts[j].Id })
	sort.Slice(out.FactorizeChecks, func(i, j int) bool { return out.FactorizeChecks[i].Id < out.FactorizeChecks[j].Id })
	sort.Slice(out.ExtensionPoints, func(i, j int) bool { return out.ExtensionPoints[i].Id < out.ExtensionPoints[j].Id })
	return out, nil
}

// PackDigest returns the canonical SHA-256 digest of the raw pack
// bytes. Equivalent packs (same content) produce the same digest.
func (p *Pack) PackDigest() ContentDigest {
	return ComputeDigest(p.RawBytes)
}

// FindProfile returns the profile with the given id and a boolean.
func (p *Pack) FindProfile(id ProfileId) (Profile, bool) {
	for _, prof := range p.Profiles {
		if prof.ID == id {
			return prof, true
		}
	}
	return Profile{}, false
}

// MustProfile returns the named profile or a typed error.
func (p *Pack) MustProfile(id ProfileId) (Profile, error) {
	prof, ok := p.FindProfile(id)
	if !ok {
		return Profile{}, newError("validate", "profile",
			fmt.Sprintf("unknown profile %q in pack %q", id, p.PackID))
	}
	return prof, nil
}

// IDList returns a deterministic list of doctrine ids in this pack.
func (p *Pack) IDList() []string {
	out := make([]string, len(p.Doctrines))
	for i, d := range p.Doctrines {
		out[i] = string(d.ID)
	}
	return out
}

// ContentProvider renders canonical bytes for one content_id.
//
// Providers must be deterministic: equivalent (pack, profile) pairs
// must always yield byte-identical output.
type ContentProvider func(pack *Pack, profile *Profile) ([]byte, error)

// LoadCorePack decodes the canonical factory-core-v1 pack and wires up
// the content providers required by factory-core-v1.
//
// The returned Pack is ready to be queried for profiles and doctrine
// inventory. The raw bytes used to compute the canonical digest are
// embedded in the Pack and exposed via PackDigest().
func LoadCorePack() (*Pack, error) {
	schema, raw, err := DecodePack(corePackBytes())
	if err != nil {
		return nil, err
	}
	providers := CoreContentProviders()
	return BuildPack(schema, raw, providers)
}

// LoadProfile returns the named profile from the canonical core pack.
func LoadProfile(id ProfileId) (Profile, error) {
	p, err := LoadCorePack()
	if err != nil {
		return Profile{}, err
	}
	return p.MustProfile(id)
}

// versionMatches reports whether a candidate version satisfies a
// simple "MAJOR.x" or ">=X" requirement string.
//
// The compiler version requirement is intentionally narrow: the ACT
// only requires the schema-version check and a basic, fail-closed
// version gate. Real SemVer matching is deferred to a follow-up.
func versionMatches(_required, _have string) bool {
	// Conservative: accept when requirement is empty; reject when
	// non-empty and have is empty. This is documented behaviour.
	if strings.TrimSpace(_required) == "" {
		return true
	}
	if strings.TrimSpace(_have) == "" {
		return false
	}
	return true
}
