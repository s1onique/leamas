package doctrinecompiler

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
)

// corePackBytes holds the canonical factory-core-v1 pack JSON.
//
//go:embed packs/factory-core-v1/pack.json
var corePackJSON []byte

// corePackBytes returns the embedded canonical pack bytes.
//
// The returned slice is owned by the caller; mutations are safe.
func corePackBytes() []byte {
	out := make([]byte, len(corePackJSON))
	copy(out, corePackJSON)
	return out
}

// ContentID namespaces used by factory-core-v1.
const (
	ContentIDFactoryMk         = "factory-core:factory-mk"
	ContentIDDoctrineInventory = "factory-core:doctrine-inventory-md"
	ContentIDFactoryReadme     = "factory-core:factory-readme-md"
	ContentIDRootMakefile      = "factory-core:root-makefile"
	ContentIDProjectSelector   = "factory-core:project-selector"
	ContentIDLockFile          = "factory-core:lock-file"
)

// CoreContentProviders returns the content providers for all content
// ids declared by factory-core-v1.
//
// The returned map is a fresh copy; callers may freely mutate it.
func CoreContentProviders() map[string]ContentProvider {
	return map[string]ContentProvider{
		ContentIDFactoryMk:         renderFactoryMk,
		ContentIDDoctrineInventory: renderDoctrineInventoryMd,
		ContentIDFactoryReadme:     renderFactoryReadmeMd,
		ContentIDRootMakefile:      renderRootMakefile,
		ContentIDProjectSelector:   renderProjectSelector,
		// ContentIDLockFile is intentionally absent. The lock file is
		// produced from the projection itself, not from a content
		// provider, because its contents depend on pack/profile state
		// and the computed digests of every managed output.
	}
}

// renderFactoryMk generates the .factory/generated/factory.mk fragment.
//
// The fragment defines a read-only factorize target. It never invokes
// compile, never rewrites the lock, never formats sources, propagates
// non-zero exit codes, and preserves failure output.
func renderFactoryMk(pack *Pack, profile *Profile) ([]byte, error) {
	if pack == nil || profile == nil {
		return nil, newError("compile", "factory-mk", "nil pack or profile")
	}
	var b strings.Builder
	b.WriteString(generatedNotice(pack))
	b.WriteString(Newline)
	b.WriteString("# This Make fragment is read-only. It must never invoke")
	b.WriteString(Newline)
	b.WriteString("# `leamas factory doctrine compile`, never rewrite the")
	b.WriteString(Newline)
	b.WriteString("# doctrine lock, and never format or modify source files.")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString(".PHONY: factorize")
	b.WriteString(Newline)
	b.WriteString("LEAMAS ?= leamas")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString("factorize:")
	b.WriteString(Newline)
	checks := append([]FactorizeCheck(nil), profile.FactorizeChecks...)
	sort.Slice(checks, func(i, j int) bool { return checks[i].Id < checks[j].Id })
	if len(checks) == 0 {
		b.WriteString("\t@echo \"factorize: no checks declared\"")
		b.WriteString(Newline)
		return []byte(b.String()), nil
	}
	for _, c := range checks {
		b.WriteString("\t@echo \"factorize: ")
		b.WriteString(c.Id)
		b.WriteString("\"")
		b.WriteString(Newline)
		b.WriteString("\t@$(LEAMAS)")
		for _, arg := range c.Command[1:] {
			b.WriteString(" ")
			b.WriteString(arg)
		}
		b.WriteString(Newline)
	}
	return []byte(b.String()), nil
}

// renderDoctrineInventoryMd generates the doctrine-inventory markdown.
//
// The output is a deterministic markdown list of doctrines enabled by
// the pack, suitable for human review and for automated diffing.
func renderDoctrineInventoryMd(pack *Pack, profile *Profile) ([]byte, error) {
	if pack == nil {
		return nil, newError("compile", "doctrine-inventory", "nil pack")
	}
	var b strings.Builder
	b.WriteString(generatedNotice(pack))
	b.WriteString(Newline)
	b.WriteString("# Factory Doctrine Inventory")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString("Pack: ")
	b.WriteString(string(pack.PackID))
	b.WriteString(" (")
	b.WriteString(string(pack.PackVersion))
	b.WriteString(")")
	b.WriteString(Newline)
	if profile != nil {
		b.WriteString("Profile: ")
		b.WriteString(string(profile.ID))
		b.WriteString(Newline)
	}
	b.WriteString(Newline)
	b.WriteString("This document is generated. Do not edit it directly.")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString("## Doctrines")
	b.WriteString(Newline)
	b.WriteString(Newline)
	doctrines := append([]Doctrine(nil), pack.Doctrines...)
	sort.Slice(doctrines, func(i, j int) bool { return doctrines[i].ID < doctrines[j].ID })
	for _, d := range doctrines {
		b.WriteString("- **`")
		b.WriteString(string(d.ID))
		b.WriteString("`** — ")
		b.WriteString(d.Summary)
		b.WriteString(Newline)
	}
	return []byte(b.String()), nil
}

// renderFactoryReadmeMd generates docs/factory/README.md.
func renderFactoryReadmeMd(pack *Pack, profile *Profile) ([]byte, error) {
	if pack == nil {
		return nil, newError("compile", "factory-readme", "nil pack")
	}
	var b strings.Builder
	b.WriteString(generatedNotice(pack))
	b.WriteString(Newline)
	b.WriteString("# Factory Wiring")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString("This directory documents the Factory doctrine wiring that")
	b.WriteString(Newline)
	b.WriteString("governs this repository.")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString("## Pack")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString("- Pack: `")
	b.WriteString(string(pack.PackID))
	b.WriteString("` (version `")
	b.WriteString(string(pack.PackVersion))
	b.WriteString("`)")
	b.WriteString(Newline)
	if profile != nil {
		b.WriteString("- Profile: `")
		b.WriteString(string(profile.ID))
		b.WriteString("`")
		b.WriteString(Newline)
	}
	b.WriteString(Newline)
	b.WriteString("## Files in this projection")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString("| Path | Ownership |")
	b.WriteString(Newline)
	b.WriteString("| --- | --- |")
	b.WriteString(Newline)
	b.WriteString("| `.factory/project.json` | seeded (target-owned after creation) |")
	b.WriteString(Newline)
	b.WriteString("| `.factory/doctrine.lock.json` | managed (compiler-owned) |")
	b.WriteString(Newline)
	b.WriteString("| `.factory/generated/factory.mk` | managed (compiler-owned) |")
	b.WriteString(Newline)
	b.WriteString("| `.factory/generated/doctrine-inventory.md` | managed (compiler-owned) |")
	b.WriteString(Newline)
	b.WriteString("| `docs/factory/README.md` | managed (compiler-owned) |")
	b.WriteString(Newline)
	b.WriteString("| `Makefile` | seeded (target-owned after creation) |")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString("## Commands")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString("- `make factorize` — run read-only Factory verification")
	b.WriteString(Newline)
	b.WriteString("- `make gate` — run the repository's native gate")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString("See the doctrine inventory at")
	b.WriteString(Newline)
	b.WriteString("`.factory/generated/doctrine-inventory.md` for the full list")
	b.WriteString(Newline)
	b.WriteString("of enabled doctrines.")
	b.WriteString(Newline)
	return []byte(b.String()), nil
}

// renderRootMakefile generates the root Makefile seed.
//
// The seed includes the generated fragment and defines a minimal gate
// target that depends on factorize. Once created, the target repository
// owns this file and may extend the gate target freely.
func renderRootMakefile(pack *Pack, profile *Profile) ([]byte, error) {
	if pack == nil {
		return nil, newError("compile", "root-makefile", "nil pack")
	}
	var b strings.Builder
	b.WriteString(seededNotice(pack))
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString("include .factory/generated/factory.mk")
	b.WriteString(Newline)
	b.WriteString(Newline)
	b.WriteString(".PHONY: gate")
	b.WriteString(Newline)
	b.WriteString("gate: factorize")
	b.WriteString(Newline)
	return []byte(b.String()), nil
}

// renderProjectSelector generates the .factory/project.json seed.
//
// The selector identifies the pack and profile used by the target
// repository. Optional bounded settings may be added by future packs.
func renderProjectSelector(pack *Pack, profile *Profile) ([]byte, error) {
	if pack == nil || profile == nil {
		return nil, newError("compile", "project-selector", "nil pack or profile")
	}
	enc := NewCanonicalJSON()
	if err := enc.WriteObject([][2]any{
		{"pack", string(pack.PackID)},
		{"profile", string(profile.ID)},
		{"schema_version", int(LockSchemaVersion)},
	}); err != nil {
		return nil, err
	}
	out := append([]byte(nil), enc.Bytes()...)
	out = append(out, '\n')
	return out, nil
}

// generatedNotice returns the standard "Generated by Leamas" notice
// for managed files. The notice includes the pack identity so that
// humans can tell which pack produced the file.
func generatedNotice(pack *Pack) string {
	return fmt.Sprintf(
		"# Generated by Leamas from %s.\n# Do not edit this file directly.",
		string(pack.PackID))
}

// seededNotice returns the standard seeded-ownership notice for files
// that become repository-owned after creation. The notice communicates
// that the file was seeded by Leamas and that the repository owns it
// after creation.
func seededNotice(pack *Pack) string {
	return fmt.Sprintf(
		"# Seeded by Leamas from %s.\n# This file is repository-owned after creation.\n# Preserve the required Factory integration contracts.",
		string(pack.PackID))
}
