// Package digest provides targeted digest generation for Git repositories.
package digest

import "regexp"

// Status constants for changed files.
const (
	StatusAdded     = "A"
	StatusModified  = "M"
	StatusDeleted   = "D"
	StatusRenamed   = "R"
	StatusCopied    = "C"
	StatusUnmerged  = "U"
	StatusUntracked = "?"
)

// ReviewChangedFile represents a file with its review-oriented status.
type ReviewChangedFile struct {
	Status  string // Single-letter status: A, M, D, R, C, U, ?
	Path    string
	OldPath string // For renames/copies: the original path
}

// FileStats holds counts of different file types in the changeset.
type FileStats struct {
	FilesChanged   int
	AddedFiles     int
	ModifiedFiles  int
	DeletedFiles   int
	RenamedFiles   int
	CopiedFiles    int
	UntrackedFiles int
	UnmergedFiles  int
	BinaryFiles    int
	GeneratedFiles int
	TestFiles      int
	DocFiles       int
	SourceFiles    int
	ConfigFiles    int
}

// ReviewMap groups files by reviewer role.
type ReviewMap struct {
	Production []string
	Tests      []string
	Docs       []string
	Config     []string
	Generated  []string
	Binary     []string
}

// RiskSignals contains deterministic facts that help reviewers focus.
type RiskSignals struct {
	ProductionWithoutTests  bool
	TestsWithoutProduction  bool
	DocsWithoutCode         bool
	GeneratedFilesChanged   bool
	ConfigFilesChanged      bool
	DeletedFilesChanged     bool
	UnmergedFilesPresent    bool
	LargeFileChanged        bool
	LargeFileThresholdBytes int64
}

// LargeFileThreshold is the default threshold for detecting large files (1 MiB).
const LargeFileThreshold int64 = 1024 * 1024

// generatedMarker matches the canonical Go generated-file marker format.
var generatedMarker = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// PatchHygieneStatus values for git diff --check results.
const (
	PatchHygienePass        = "pass"
	PatchHygieneFail        = "fail"
	PatchHygieneUnavailable = "unavailable"
)

// MaxPatchHygieneDiagnostics is the maximum number of diagnostic lines to include.
const MaxPatchHygieneDiagnostics = 20

// MaxDiagnosticLineLength is the maximum length for each diagnostic line.
const MaxDiagnosticLineLength = 240

// PatchHygiene contains patch hygiene check results.
type PatchHygiene struct {
	GitDiffCheck     string
	WhitespaceErrors int
	ConflictMarkers  int
	DiagnosticLines  int
	Diagnostics      []string
}
