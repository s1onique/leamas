package boardmd

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// exitCriteriaHeading matches the heading of an exit-criteria section.
var exitCriteriaHeading = regexp.MustCompile(`(?i)^#{2,3}\s+.*(exit criteria|definition of done).*$`)

// extractActRows extracts the ACT ledger rows of an epic file.
//
// The ACT ledger table rows carry the ACT identity, the verdict and the
// source head. They are the ACT EXECUTION records of the historical
// board, not work items: a closed ACT is not a second current obligation.
func extractActRows(lines []string) []actRow {
	var out []actRow
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
		if !strings.EqualFold(section, "ACT ledger") {
			continue
		}
		m := ledgerRow.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		cells := splitTableRow(line)
		row := actRow{id: m[1], line: i + 1}
		if len(cells) > 1 {
			row.verdict = strings.Trim(strings.TrimSpace(cells[1]), "*` ")
		}
		if len(cells) > 2 {
			row.head = strings.Trim(strings.TrimSpace(cells[2]), "*` ")
		}
		out = append(out, row)
	}
	return out
}

// frontierHeading matches the heading of the current-frontier section.
var frontierHeading = regexp.MustCompile(`(?i)^#{2,3}\s+current frontier\s*$`)

// frontierRow matches one current-frontier lane row.
//
// The row shape is:
//
//	| Lane | Pri | State | NEXT | Detail |
//
// The lane is the workstream scope, the state is the explicit source
// status and the NEXT column names the frontier ACT of the lane.
var frontierRow = regexp.MustCompile(`^\|\s*([^|]+?)\s*\|\s*([^|]*?)\s*\|\s*([^|]*?)\s*\|\s*([^|]*?)\s*\|(?:\s*([^|]*?)\s*\|)?`)

// frontierLane is one extracted current-frontier lane.
//
// A lane is the CURRENT STATE of one workstream scope: its priority, its
// state and the ACT that is its frontier.
type frontierLane struct {
	lane   string
	pri    string
	state  string
	next   string
	detail string
	epic   string
	line   int
}

// extractFrontierLanes extracts the current-frontier lane table.
//
// The current frontier is the AUTHORITATIVE current state of the board: it
// names the workstream scopes, their state and their frontier ACT.
//
// A lane without a state stays unresolved: it is never coerced into a
// known lifecycle state.
func extractFrontierLanes(lines []string) []frontierLane {
	var out []frontierLane
	inSection := false
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			// The fence state is RESET at every heading: the
			// historical board carries at least one UNBALANCED code
			// fence. A naive toggle would leave the whole remainder
			// of the board "inside" a fence and the current frontier
			// would become invisible.
			inFence = false
			inSection = frontierHeading.MatchString(line)
			continue
		}
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence || !inSection {
			continue
		}
		m := frontierRow.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lane := strings.TrimSpace(m[1])
		// The header row and the separator row are not lanes.
		if lane == "" || lane == "Lane" || strings.HasPrefix(lane, "---") {
			continue
		}
		next := strings.Trim(strings.TrimSpace(m[4]), "*` ")
		detail := ""
		if len(m) > 5 {
			detail = strings.TrimSpace(m[5])
		}
		out = append(out, frontierLane{
			lane:   lane,
			pri:    strings.Trim(strings.TrimSpace(m[2]), "*` "),
			state:  strings.Trim(strings.TrimSpace(m[3]), "*` "),
			next:   next,
			detail: detail,
			epic:   epicOfDetail(detail),
			line:   i + 1,
		})
	}
	return out
}

// epicOfDetail returns the epic file a detail link names.
//
// The detail link names the epic scope the lane belongs to: the link
// target is the epic file.
func epicOfDetail(detail string) string {
	start := strings.Index(detail, "](./epics/")
	if start < 0 {
		return ""
	}
	rest := detail[start+len("](./epics/"):]
	end := strings.Index(rest, ")")
	if end < 0 {
		return ""
	}
	return strings.TrimSuffix(rest[:end], ".md")
}

// laneTitle returns the title of a frontier lane.
//
// The NEXT cell of the historical board may carry a very long narrative.
// The canonical ledger carries STRUCTURED STATE AND REFERENCES: the title
// is the frontier ACT identity, and the full narrative stays in the source
// board, reachable through the source line locator.
func laneTitle(lane frontierLane) string {
	if id := actIdentityOf(lane.next); id != "" {
		return id
	}
	return lane.lane
}

// epicScopeOf resolves the epic scope of a lane.
//
// The lane names its epic by the epic FILE stem. The epic IDENTITY is
// that file's heading. A lane whose epic file does not exist falls back
// to the board scope: the lane is then reported as belonging to the
// board scope rather than to an invented epic.
func epicScopeOf(fileStem string, scopes map[string]string) string {
	if id, ok := scopes[fileStem]; ok {
		return id
	}
	return "EPIC-CLINEMM-BOARD"
}

// laneIdentity returns the canonical identity of a frontier lane.
//
// The lane LABEL is free text ("Approval / editor-tool"). The identity
// must be a stable, canonical token: the frontier ACT of the lane when it
// names one, and the lane label otherwise, slugged into the canonical
// identity shape.
func laneIdentity(lane frontierLane) string {
	if id := actIdentityOf(lane.next); id != "" {
		return id
	}
	return slugIdentity(lane.lane)
}

// actIdentityOf returns the ACT identity a frontier cell names.
//
// The cell may name the ACT alone or name it among other text. The ACT
// identity is the first token that has the canonical ACT shape.
func actIdentityOf(cell string) string {
	for _, field := range strings.Fields(cell) {
		trimmed := strings.Trim(field, "*`(),")
		if strings.HasPrefix(trimmed, "ACT-") {
			return trimmed
		}
	}
	return ""
}

// slugIdentity converts free text into the canonical identity shape.
//
// The identity is derived from the text deterministically: the same lane
// label always produces the same identity, and two different labels never
// collide.
func slugIdentity(text string) string {
	upper := strings.ToUpper(text)
	var b strings.Builder
	for _, r := range upper {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	slug := strings.Trim(b.String(), "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	if slug == "" {
		return "LANE-UNNAMED"
	}
	if slug[0] < 'A' || slug[0] > 'Z' {
		slug = "LANE-" + slug
	}
	return slug
}

// splitTableRow splits a markdown table row into its cells.
func splitTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	return strings.Split(trimmed, "|")
}

// extractExitCriteria extracts the exit criteria of an epic file.
//
// The criteria are the bullet list items of the exit-criteria section. A
// criterion is conserved VERBATIM: its exact text is never reworded.
func extractExitCriteria(lines []string) []string {
	var out []string
	inSection := false
	inFence := false
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(line, "#") {
			inSection = exitCriteriaHeading.MatchString(line)
			continue
		}
		if !inSection {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") && !strings.HasPrefix(trimmed, "* ") {
			continue
		}
		criterion := strings.TrimSpace(trimmed[2:])
		if criterion == "" {
			continue
		}
		out = append(out, criterion)
	}
	return out
}

// textOf returns the source text of a line range.
func textOf(lines []string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start >= end {
		return ""
	}
	return strings.Join(lines[start:end], "\n") + "\n"
}

// digestOf returns the SHA-256 digest of a source text.
func digestOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
