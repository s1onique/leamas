// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"path/filepath"
	"strings"
)

// ComputeStats computes file statistics from a manifest.
func ComputeStats(manifest []ReviewChangedFile, repoRoot string) FileStats {
	var stats FileStats

	for _, f := range manifest {
		stats.FilesChanged++

		switch f.Status {
		case StatusAdded:
			stats.AddedFiles++
		case StatusModified:
			stats.ModifiedFiles++
		case StatusDeleted:
			stats.DeletedFiles++
		case StatusRenamed:
			stats.RenamedFiles++
		case StatusCopied:
			stats.CopiedFiles++
		case StatusUntracked:
			stats.UntrackedFiles++
		case StatusUnmerged:
			stats.UnmergedFiles++
		}

		if isGeneratedFileAtPath(filepath.Join(repoRoot, f.Path)) {
			stats.GeneratedFiles++
		} else if isBinaryFileAtPath(filepath.Join(repoRoot, f.Path)) {
			stats.BinaryFiles++
		} else {
			switch classifyFile(f.Path) {
			case "test":
				stats.TestFiles++
			case "doc":
				stats.DocFiles++
			case "config":
				stats.ConfigFiles++
			case "source":
				stats.SourceFiles++
			}
		}
	}

	return stats
}

// RenderStats renders the CHANGESET_STATS section.
func RenderStats(stats FileStats) string {
	var sb strings.Builder
	sb.WriteString("## CHANGESET_STATS\n")
	sb.WriteString("files_changed=")
	sb.WriteString(intToString(stats.FilesChanged))
	sb.WriteString("\nadded_files=")
	sb.WriteString(intToString(stats.AddedFiles))
	sb.WriteString("\nmodified_files=")
	sb.WriteString(intToString(stats.ModifiedFiles))
	sb.WriteString("\ndeleted_files=")
	sb.WriteString(intToString(stats.DeletedFiles))
	sb.WriteString("\nrenamed_files=")
	sb.WriteString(intToString(stats.RenamedFiles))
	sb.WriteString("\ncopied_files=")
	sb.WriteString(intToString(stats.CopiedFiles))
	sb.WriteString("\nuntracked_files=")
	sb.WriteString(intToString(stats.UntrackedFiles))
	sb.WriteString("\nunmerged_files=")
	sb.WriteString(intToString(stats.UnmergedFiles))
	sb.WriteString("\nbinary_files=")
	sb.WriteString(intToString(stats.BinaryFiles))
	sb.WriteString("\ngenerated_files=")
	sb.WriteString(intToString(stats.GeneratedFiles))
	sb.WriteString("\ntest_files=")
	sb.WriteString(intToString(stats.TestFiles))
	sb.WriteString("\ndoc_files=")
	sb.WriteString(intToString(stats.DocFiles))
	sb.WriteString("\nsource_files=")
	sb.WriteString(intToString(stats.SourceFiles))
	sb.WriteString("\nconfig_files=")
	sb.WriteString(intToString(stats.ConfigFiles))
	sb.WriteString("\n")
	return sb.String()
}
