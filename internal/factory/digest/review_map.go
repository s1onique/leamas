// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"path/filepath"
	"sort"
	"strings"
)

// BuildReviewMap groups files by reviewer role.
func BuildReviewMap(manifest []ReviewChangedFile, repoRoot string) ReviewMap {
	var rm ReviewMap
	rm.Production = []string{}
	rm.Tests = []string{}
	rm.Docs = []string{}
	rm.Config = []string{}
	rm.Generated = []string{}
	rm.Binary = []string{}

	seen := make(map[string]bool)

	for _, f := range manifest {
		if seen[f.Path] {
			continue
		}
		seen[f.Path] = true

		fullPath := filepath.Join(repoRoot, f.Path)

		if isGeneratedFileAtPath(fullPath) {
			rm.Generated = append(rm.Generated, f.Path)
			continue
		}
		if isBinaryFileAtPath(fullPath) {
			rm.Binary = append(rm.Binary, f.Path)
			continue
		}

		switch classifyFile(f.Path) {
		case "test":
			rm.Tests = append(rm.Tests, f.Path)
		case "doc":
			rm.Docs = append(rm.Docs, f.Path)
		case "config":
			rm.Config = append(rm.Config, f.Path)
		default:
			rm.Production = append(rm.Production, f.Path)
		}
	}

	sort.Strings(rm.Production)
	sort.Strings(rm.Tests)
	sort.Strings(rm.Docs)
	sort.Strings(rm.Config)
	sort.Strings(rm.Generated)
	sort.Strings(rm.Binary)

	return rm
}

// RenderReviewMap renders the REVIEW_MAP section.
func RenderReviewMap(rm ReviewMap) string {
	var sb strings.Builder
	sb.WriteString("## REVIEW_MAP\n")

	renderGroup(&sb, "production", rm.Production)
	renderGroup(&sb, "tests", rm.Tests)
	renderGroup(&sb, "docs", rm.Docs)
	renderGroup(&sb, "config", rm.Config)
	renderGroup(&sb, "generated", rm.Generated)
	renderGroup(&sb, "binary", rm.Binary)

	return sb.String()
}

func renderGroup(sb *strings.Builder, name string, paths []string) {
	sb.WriteString(name)
	sb.WriteString(":\n")
	if len(paths) == 0 {
		sb.WriteString("  - none\n")
	} else {
		for _, p := range paths {
			sb.WriteString("  - ")
			sb.WriteString(p)
			sb.WriteString("\n")
		}
	}
}
