package doctrinecompiler

import (
	"sort"
	"strings"
)

// ExplainReport is the structured output of the explain command.
type ExplainReport struct {
	Pack            ExplainPack
	Profile         ExplainProfile
	CompilerVersion string
	CompilerCommit  string
	PackDigest      string
	SourceRevision  string
	Managed         []string
	Seeded          []string
	Observed        []string
	Doctrines       []string
	ExtensionPoints []string
}

// ExplainPack carries the pack-level summary.
type ExplainPack struct {
	ID      string
	Version string
}

// ExplainProfile carries the profile-level summary.
type ExplainProfile struct {
	ID      string
	Summary string
}

// Explain inspects the target repository and produces an ExplainReport.
// The explain command performs no writes and is intended for human
// review or scripted consumption.
func Explain(pack *Pack, profile Profile, target, compilerVersion, compilerCommit, sourceRevision string) (ExplainReport, error) {
	resolver, err := NewResolver(target)
	if err != nil {
		return ExplainReport{}, err
	}
	rep := ExplainReport{
		Pack: ExplainPack{
			ID:      string(pack.PackID),
			Version: string(pack.PackVersion),
		},
		Profile: ExplainProfile{
			ID:      string(profile.ID),
			Summary: profile.Summary,
		},
		CompilerVersion: compilerVersion,
		CompilerCommit:  compilerCommit,
		PackDigest:      string(pack.PackDigest()),
		SourceRevision:  sourceRevision,
	}
	rep.Managed = collectExisting(resolver, profile.Outputs)
	rep.Seeded = collectExisting(resolver, toSeedPaths(profile.Seeds))
	for _, c := range profile.ObservedContracts {
		rep.Observed = append(rep.Observed, c.Id)
	}
	sort.Strings(rep.Observed)
	// Explain only the doctrines enabled by the profile when the
	// profile declares an explicit set; otherwise fall back to the
	// full pack inventory.
	if len(profile.EnabledDoctrines) > 0 {
		enabled := make(map[DoctrineId]struct{}, len(profile.EnabledDoctrines))
		for _, did := range profile.EnabledDoctrines {
			enabled[did] = struct{}{}
		}
		for _, d := range pack.Doctrines {
			if _, ok := enabled[d.ID]; ok {
				rep.Doctrines = append(rep.Doctrines, string(d.ID))
			}
		}
	} else {
		for _, d := range pack.Doctrines {
			rep.Doctrines = append(rep.Doctrines, string(d.ID))
		}
	}
	for _, ep := range profile.ExtensionPoints {
		rep.ExtensionPoints = append(rep.ExtensionPoints, ep.Id)
	}
	return rep, nil
}

// toSeedPaths converts []SeedDecl into []TargetPath.
func toSeedPaths(in []SeedDecl) []TargetPath {
	out := make([]TargetPath, 0, len(in))
	for _, s := range in {
		out = append(out, s.Path)
	}
	return out
}

// collectExisting returns a sorted list of TargetPath entries that
// exist on disk for the given set of declarations.
func collectExisting(resolver *Resolver, decls interface{}) []string {
	var paths []TargetPath
	switch d := decls.(type) {
	case []OutputDecl:
		for _, o := range d {
			paths = append(paths, o.Path)
		}
	case []TargetPath:
		paths = append(paths, d...)
	}
	sort.Slice(paths, func(i, j int) bool { return paths[i] < paths[j] })
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		kind, _, err := resolver.InspectPath(p)
		if err == nil && kind != PathMissing {
			out = append(out, string(p))
		}
	}
	return out
}

// FormatExplain renders an ExplainReport as deterministic human-readable
// text suitable for CLI output and golden-file tests.
func FormatExplain(rep ExplainReport) []byte {
	var b strings.Builder
	b.WriteString("explain:")
	b.WriteString(Newline)
	b.WriteString("  pack:")
	b.WriteString(Newline)
	b.WriteString("    id: ")
	b.WriteString(rep.Pack.ID)
	b.WriteString(Newline)
	b.WriteString("    version: ")
	b.WriteString(rep.Pack.Version)
	b.WriteString(Newline)
	b.WriteString("    digest: ")
	b.WriteString(rep.PackDigest)
	b.WriteString(Newline)
	b.WriteString("  profile:")
	b.WriteString(Newline)
	b.WriteString("    id: ")
	b.WriteString(rep.Profile.ID)
	b.WriteString(Newline)
	if rep.Profile.Summary != "" {
		b.WriteString("    summary: ")
		b.WriteString(rep.Profile.Summary)
		b.WriteString(Newline)
	}
	b.WriteString("  compiler:")
	b.WriteString(Newline)
	b.WriteString("    version: ")
	b.WriteString(rep.CompilerVersion)
	b.WriteString(Newline)
	if rep.CompilerCommit != "" {
		b.WriteString("    commit: ")
		b.WriteString(rep.CompilerCommit)
		b.WriteString(Newline)
	}
	if rep.SourceRevision != "" {
		b.WriteString("    source_revision: ")
		b.WriteString(rep.SourceRevision)
		b.WriteString(Newline)
	}
	b.WriteString("  managed_files:")
	b.WriteString(Newline)
	for _, p := range rep.Managed {
		b.WriteString("    - ")
		b.WriteString(p)
		b.WriteString(Newline)
	}
	b.WriteString("  seeded_files:")
	b.WriteString(Newline)
	for _, p := range rep.Seeded {
		b.WriteString("    - ")
		b.WriteString(p)
		b.WriteString(Newline)
	}
	b.WriteString("  observed_contracts:")
	b.WriteString(Newline)
	for _, id := range rep.Observed {
		b.WriteString("    - ")
		b.WriteString(id)
		b.WriteString(Newline)
	}
	b.WriteString("  doctrines:")
	b.WriteString(Newline)
	for _, id := range rep.Doctrines {
		b.WriteString("    - ")
		b.WriteString(id)
		b.WriteString(Newline)
	}
	b.WriteString("  extension_points:")
	b.WriteString(Newline)
	for _, id := range rep.ExtensionPoints {
		b.WriteString("    - ")
		b.WriteString(id)
		b.WriteString(Newline)
	}
	return []byte(b.String())
}
