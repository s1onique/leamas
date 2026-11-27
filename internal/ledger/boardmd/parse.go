package boardmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/importer"
)

// mergeFields merges the fields of a metadata block into the section.
//
// The FIRST block of a section wins: a section that quotes a later
// metadata block does not change its own identity.
func mergeFields(dst, src map[string]string) {
	for k, v := range src {
		if _, exists := dst[k]; exists {
			continue
		}
		dst[k] = v
	}
}

// boardHeading matches an ACT report heading of the historical board.
//
// The heading shape is:
//
//	# <ACT-ID> [(<QUALIFIER>)] — <VERDICT> — <DATE>
//
// The qualifier, the verdict and the date are OPTIONAL: an older section
// may carry only the identity.
//
// A heading that does NOT start with an ACT identity is not an ACT report
// section: it is narrative. It is counted as a section and never invented
// as an ACT.
var boardHeading = regexp.MustCompile(`^#\s+(ACT-[A-Za-z0-9._-]+)`)

// metaField matches one key/value field of a ```text metadata block.
var metaField = regexp.MustCompile(`^([A-Z_]+):\s*(.*)$`)

// openWorkItem matches an open-work bullet of an epic file.
//
// The bullet shape is:
//
//   - **`ITEM-ID`** (…notes…). Status: **OPEN / HIGH**. …
//
// The identity is the backtick-quoted token it declares. Only an
// UPPERCASE-LED identity token qualifies: a lowercase file reference is
// prose, not a work-item identity.
//
// The identity token may be wrapped in bold markers: the source writes
// "- **`ITEM-ID`**".
var openWorkItem = regexp.MustCompile("^-\\s+\\*?\\*?`+([A-Z][A-Z0-9]*(?:-[A-Z0-9]+)*)`+\\*?\\*?")

// statusLabel matches the explicit status declaration of a bullet.
var statusLabel = regexp.MustCompile(`(?i)Status:\s*\**\s*([A-Z_ /-]+)`)

// ledgerRow matches an ACT ledger table row of an epic file.
//
// The row shape is:
//
//	| `ACT-ID` | VERDICT | HEAD | PURPOSE |
var ledgerRow = regexp.MustCompile("^\\|\\s*`([^`]+)`\\s*\\|")

// boardSection is one extracted ACT report section of the board.
type boardSection struct {
	section Section
	fields  map[string]string
}

// splitBoardSections splits the historical board into its ACT report
// sections.
//
// The board is a sequence of markdown H1 headings OUTSIDE a code fence.
// Each heading starts a section that runs until the next such heading.
//
// A heading inside a code fence is NOT a section: it is quoted source
// text. Counting it as a section would invent history that the board does
// not contain.
func splitBoardSections(text string) []boardSection {
	lines := strings.Split(text, "\n")
	var out []boardSection
	var current *boardSection
	inFence := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "# ") && boardHeading.MatchString(line) {
			// The fence state is RESET at every heading: the
			// historical board carries at least one UNBALANCED code
			// fence. A naive toggle would leave the remainder of the
			// board "inside" a fence and the ACT reports would become
			// invisible.
			inFence = false
			if current != nil {
				current.section.EndLine = i
				out = append(out, *current)
			}
			current = &boardSection{
				section: Section{
					Path:      BoardPath,
					StartLine: i + 1,
					EndLine:   i + 1,
					Locator:   fmt.Sprintf("%s#L%d", BoardPath, i+1),
					Digest:    digestOf(textOf(lines, i, i+1)),
				},
				fields: map[string]string{},
			}
			continue
		}
		if strings.HasPrefix(line, "```") {
			// The fence marker itself may OPEN a metadata block: the
			// block is read before the fence state flips.
			if current != nil {
				mergeFields(current.fields, metadataFields(lines, i))
			}
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if current == nil {
			continue
		}
		mergeFields(current.fields, metadataFields(lines, i))
	}
	if current != nil {
		current.section.EndLine = len(lines)
		out = append(out, *current)
	}
	return out
}

// metadataFields reads the ```text metadata block that starts at the line.
//
// The block carries the record fields. A block that carries no field at
// all contributes nothing.
func metadataFields(lines []string, index int) map[string]string {
	if !strings.HasPrefix(lines[index], "```text") {
		return nil
	}
	fields := map[string]string{}
	for i := index + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "```") {
			break
		}
		if m := metaField.FindStringSubmatch(lines[i]); m != nil {
			fields[m[1]] = strings.TrimSpace(m[2])
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// epicRecord is one parsed epic file of the historical board.
//
// An epic file carries the epic scope, the open work, the deferred work
// and the ACT ledger of that scope.
type epicRecord struct {
	id           string
	title        string
	workstream   string
	status       string
	priority     string
	exitCriteria []string
	workItems    []importer.SourceItem
	actRows      []actRow
	section      Section
}

// actRow is one ACT ledger row of an epic file.
type actRow struct {
	id      string
	verdict string
	head    string
	line    int
}

// parseEpicFile parses one epic file of the historical board.
//
// The epic identity is the H1 heading. The open work is the bullet list of
// the "Open work" section. The deferred work is the H3 subsections of the
// "Deferred work" section. The ACT ledger is the table rows of the
// "ACT ledger" section.
//
// A file without an identity is not an epic scope and is skipped.
func parseEpicFile(rel, text string) *epicRecord {
	lines := strings.Split(text, "\n")
	var id string
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			id = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			break
		}
	}
	if id == "" {
		return nil
	}
	rec := &epicRecord{
		id:    id,
		title: id,
		// The workstream identity is the file name: the epic scope and
		// the workstream are distinct identities in the historical board.
		workstream: workstreamOf(rel),
		section: Section{
			Path:      rel,
			StartLine: 1,
			EndLine:   len(lines),
			Locator:   rel,
			Digest:    digestOf(text),
		},
	}
	rec.status = epicStatus(lines)
	rec.priority = epicPriority(lines)
	rec.exitCriteria = extractExitCriteria(lines)
	rec.workItems = extractWorkItems(lines, rec)
	rec.actRows = extractActRows(lines)
	return rec
}

// workstreamOf returns the workstream identity of an epic file.
func workstreamOf(rel string) string {
	name := rel
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	return strings.TrimSuffix(name, ".md")
}

// epicStatus returns the explicit status of an epic file.
//
// The status is the "Status:" field of the "Current status" section. An
// epic without an explicit status stays UNRESOLVED: it is never coerced
// into a known lifecycle state.
func epicStatus(lines []string) string {
	inSection := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			inSection = strings.EqualFold(strings.TrimSpace(line[3:]), "Current status")
			continue
		}
		if !inSection {
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "- Status:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- Status:"))
		if idx := strings.Index(value, "—"); idx > 0 {
			value = strings.TrimSpace(value[:idx])
		}
		return strings.TrimSpace(strings.Trim(value, "*` "))
	}
	return ""
}

// extractWorkItems extracts the work-item identities of an epic file.
//
// The open work is the bullet list of the "Open work" section. The
// deferred work is the H3 subsections of the "Deferred work" section.
//
// The identity of a bullet is the backtick-quoted token it declares. The
// status is the explicit status the bullet declares: a bullet without one
// stays unresolved.
func extractWorkItems(lines []string, rec *epicRecord) []importer.SourceItem {
	var out []importer.SourceItem
	section := ""
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			section = strings.TrimSpace(line[3:])
			continue
		}
		switch strings.ToLower(section) {
		case "open work":
			if m := openWorkItem.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				out = append(out, workItemOf(rec, m[1], bulletStatus(line), i+1))
			}
		case "deferred work":
			if strings.HasPrefix(line, "### ") {
				id := strings.TrimSpace(strings.TrimPrefix(line, "### "))
				out = append(out, workItemOf(rec, id, "DEFERRED", i+1))
			}
		}
	}
	return out
}

// workItemOf builds the source record of one work-item identity.
//
// The identity is conserved VERBATIM when it already has the canonical
// shape. A source identity that carries trailing prose ("ITEM-ID (note)")
// is SLUGGED: the canonical identity is the token, and the full source
// text is preserved as the title.
//
// The status is the explicit source status: an unrecognised status stays
// unresolved and non-executable.
func workItemOf(rec *epicRecord, identity, status string, line int) importer.SourceItem {
	canonical, title := canonicalIdentity(identity)
	return importer.SourceItem{
		SourceID:   canonical,
		EpicID:     rec.id,
		Workstream: rec.workstream,
		Title:      title,
		Status:     importer.SourceStatus(strings.ToUpper(strings.TrimSpace(status))),
		SourceLine: line,
		SourceRow:  identity,
		// The historical board is a repository document: the extraction
		// reads it from the repository at a known HEAD.
		SourceConfidence: "imported",
	}
}

// canonicalIdentity returns the canonical identity of a source identity
// and the human-readable title it carries.
//
// An identity that already has the canonical shape is conserved VERBATIM.
// An identity that carries trailing prose keeps its leading token as the
// identity and the full text as the title.
func canonicalIdentity(identity string) (string, string) {
	trimmed := strings.TrimSpace(identity)
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "ITEM-UNNAMED", trimmed
	}
	lead := strings.Trim(fields[0], "*`(),")
	if isCanonicalIdentity(lead) {
		return lead, trimmed
	}
	return slugIdentity(lead), trimmed
}

// isCanonicalIdentity reports whether the token has the canonical
// work-item identity shape.
func isCanonicalIdentity(token string) bool {
	if token == "" {
		return false
	}
	for i, r := range token {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		case r == '-':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// bulletStatus returns the explicit status of an open-work bullet.
func bulletStatus(line string) string {
	m := statusLabel.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func epicPriority(lines []string) string {
	inSection := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			inSection = strings.EqualFold(strings.TrimSpace(line[3:]), "Current status")
			continue
		}
		if !inSection {
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "- Priority:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- Priority:"))
		value = strings.Trim(value, "*` ")
		if idx := strings.Index(value, " "); idx > 0 {
			value = value[:idx]
		}
		return strings.TrimSpace(value)
	}
	return ""
}
