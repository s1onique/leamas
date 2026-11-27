// Package boardmd implements the deterministic extraction adapter for the
// ClineMM historical epic board topology.
//
// The adapter reads the ACTUAL repository-native sources:
//
//	.factory/epic-board.md        the historical board: one ACT report
//	                              section per markdown H1 heading
//	.factory/epics/<epic>.md      the epic files: one epic scope per file
//
// It converts them into the importer's SourceBoard, preserving the
// provenance every record needs: the source repository, the source HEAD,
// the source path, the source line range, the source content digest, the
// original identity, the original status and verdict, the original
// workstream and the original evidence locators.
//
// The extraction is CONSERVATIVE. It never infers a dependency, a
// successful qualification or a causality from chronology. A status it
// does not recognise stays unresolved and non-executable.
package boardmd

import (
	"github.com/s1onique/leamas/internal/ledger/importer"
)

// AdapterVersion is the version of this extraction adapter.
//
// The adapter is versioned: a source topology the adapter does not
// understand is refused rather than mis-parsed.
const AdapterVersion = 1

// BoardPath is the repository-relative path of the historical board.
const BoardPath = ".factory/epic-board.md"

// EpicsDir is the repository-relative path of the epic files.
const EpicsDir = ".factory/epics"

// Source is the identity of the source repository the extraction read.
type Source struct {
	// Repository is the repository identity (a name or a path).
	Repository string `json:"repository"`
	// Head is the source HEAD the extraction was taken at.
	Head string `json:"head"`
	// BoardPath is the repository-relative path of the board.
	BoardPath string `json:"board_path"`
	// EpicsDir is the repository-relative path of the epic files.
	EpicsDir string `json:"epics_dir"`
}

// Section is one extracted source section.
//
// It is the frozen provenance of the extracted record: the section locator
// and the content digest of the exact source text the record came from.
type Section struct {
	// Path is the repository-relative path of the source file.
	Path string `json:"path"`
	// StartLine is the first line of the section (one-based).
	StartLine int `json:"start_line"`
	// EndLine is the last line of the section (one-based).
	EndLine int `json:"end_line"`
	// Digest is the SHA-256 digest of the exact section text.
	Digest string `json:"digest"`
	// Locator is the stable section locator.
	Locator string `json:"locator"`
}

// Inventory is the observed inventory of the extracted source.
//
// It is the EVIDENCE of the conservation: it counts every section, every
// epic scope, every work-item identity and every status class that the
// extraction observed.
type Inventory struct {
	// Sections is the number of extracted source sections.
	Sections int `json:"sections"`
	// Epics is the number of distinct epic scopes.
	Epics int `json:"epics"`
	// WorkItems is the number of distinct work-item identities.
	WorkItems int `json:"work_items"`
	// ACTs is the number of distinct ACT execution identities.
	ACTs int `json:"acts"`
	// ExitCriteria is the number of extracted epic exit criteria.
	ExitCriteria int `json:"exit_criteria"`
	// Unfinished is the number of records whose status is not a completion.
	Unfinished int `json:"unfinished"`
	// Unresolved is the number of records whose status is not understood.
	Unresolved int `json:"unresolved"`
	// DuplicateSections is the number of repeated historical
	// representations of one identity.
	DuplicateSections int `json:"duplicate_sections"`
}

// Extraction is the result of the deterministic extraction.
type Extraction struct {
	// Source is the identity of the source repository.
	Source Source `json:"source"`
	// Board is the normalized source board.
	Board importer.SourceBoard `json:"-"`
	// Inventory is the observed inventory of the source.
	Inventory Inventory `json:"inventory"`
	// Sections are the frozen provenance of every extracted section.
	Sections []Section `json:"sections"`
	// ACTs are the extracted ACT execution records: the historical ACT
	// reports of the source.
	ACTs []ACTExecution `json:"acts"`
	// Unavailable records the historical information the source does NOT
	// contain. It is reported explicitly.
	Unavailable []string `json:"unavailable"`
}
