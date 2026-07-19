// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"path/filepath"
	"strings"
)

// ComputeStats computes file statistics from a manifest.
//
// Each manifest entry contributes exactly one bucket to
// `FilesChanged`. Buckets match the Git status letters. The
// classification helpers (`isGeneratedFileAtPath`,
// `isBinaryFileAtPath`, `classifyFile`) operate on the raw path
// stored in `ReviewChangedFile.Path`; semantic identity is
// preserved verbatim and `PathEscape` is never applied here.
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
		case StatusTypeChanged:
			stats.TypeChangedFiles++
		case StatusRenamed:
			stats.RenamedFiles++
		case StatusCopied:
			stats.CopiedFiles++
		case StatusUntracked:
			stats.UntrackedFiles++
		case StatusUnmerged:
			stats.UnmergedFiles++
		case StatusUnknown:
			stats.UnknownFiles++
		case StatusBrokenPair:
			stats.BrokenPairFiles++
		}

		if isGeneratedFileAtPath(filepath.Join(repoRoot, f.Path)) {
			stats.GeneratedFiles++
			continue
		}
		if isBinaryFileAtPath(filepath.Join(repoRoot, f.Path)) {
			stats.BinaryFiles++
			continue
		}
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

	return stats
}

// RenderStats renders the CHANGESET_STATS section.
//
// Key order is the canonical v3 layout documented in
// `docs/factory/digest-contract.md`. v3 inserts the three new
// status-tracked fields after the corresponding existing keys and
// pulls `untracked_files` ahead of the per-file classification
// fields. The order is load-bearing for downstream consumers; do
// not reorder without bumping the contract version.
func RenderStats(stats FileStats) string {
	var sb strings.Builder
	sb.WriteString("## CHANGESET_STATS\n")

	// Tracked-status keys, alphabetical by Git letter.
	sb.WriteString("files_changed=")
	sb.WriteString(intToString(stats.FilesChanged))
	sb.WriteString("\nadded_files=")
	sb.WriteString(intToString(stats.AddedFiles))
	sb.WriteString("\nmodified_files=")
	sb.WriteString(intToString(stats.ModifiedFiles))
	sb.WriteString("\ndeleted_files=")
	sb.WriteString(intToString(stats.DeletedFiles))
	sb.WriteString("\ntype_changed_files=")
	sb.WriteString(intToString(stats.TypeChangedFiles))
	sb.WriteString("\nrenamed_files=")
	sb.WriteString(intToString(stats.RenamedFiles))
	sb.WriteString("\ncopied_files=")
	sb.WriteString(intToString(stats.CopiedFiles))
	sb.WriteString("\nunmerged_files=")
	sb.WriteString(intToString(stats.UnmergedFiles))
	sb.WriteString("\nunknown_files=")
	sb.WriteString(intToString(stats.UnknownFiles))
	sb.WriteString("\nbroken_pair_files=")
	sb.WriteString(intToString(stats.BrokenPairFiles))
	sb.WriteString("\nuntracked_files=")
	sb.WriteString(intToString(stats.UntrackedFiles))

	// Per-file classification buckets.
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
