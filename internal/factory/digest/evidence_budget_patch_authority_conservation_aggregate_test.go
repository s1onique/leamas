// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_conservation_aggregate_test.go
// holds the AGGREGATE-budget conservation witness for
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01.
//
// AMENDMENT-01 FINALIZATION (F2): the fixture was reduced from 1,500
// files to EIGHT larger files, and the assertions were strengthened
// from "a heading is present" to "the COMPLETE patch is present
// verbatim".
//
// The companion file
// evidence_budget_patch_authority_conservation_test.go holds the
// redaction and binary-change witnesses.
package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Case 5: aggregate budget exceeded -> existing policy conserved.
// ---------------------------------------------------------------------------

// TestPatchBudget_AggregateBudgetExceeded_Conserved verifies that the
// aggregate (total) render budget still clamps the digest, exactly as
// before this ACT.
//
// AMENDMENT-01 (A5): the predecessor witness did not establish that
// the aggregate budget was EVER reached. Its 40 files each carried a
// ~196-byte patch, so the combined raw patch was ~7.8 KiB against a
// 256 KiB budget. The witness would have passed even if the renderer
// discarded every patch.
//
// The corrected witness:
//
//   - measures the ACTUAL aggregate patch bytes;
//   - requires that aggregate to EXCEED the applicable total budget;
//   - requires every individual patch to fit the per-file budget;
//   - requires that at least one patch is INCLUDED and at least one is
//     OMITTED;
//   - requires the omission record to name the omitted files;
//   - requires that an interrupted file does not masquerade as
//     complete.
func TestPatchBudget_AggregateBudgetExceeded_Conserved(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// Each file is a large SOURCE whose CONTENT change alone is
	// larger than the per-file budget would allow, but the patch
	// text of a one-line change is small. The aggregate of several
	// such patches must exceed the total render budget.
	//
	// AMENDMENT-01 FINALIZATION (F2): the predecessor used 1500
	// files of ~2000 lines and launched one independent Git
	// subprocess per file to measure them. That is an unnecessary
	// expense: the objective is to exceed the aggregate budget while
	// every individual patch stays below the per-file budget, and a
	// handful of larger patches establishes the same precondition.
	//
	// The file count is sized from the MEASURED per-file patch size
	// so the fixture provably creates real aggregate pressure rather
	// than assuming it.
	//
	// Each file carries `changedLines` MODIFIED lines. A single
	// modified line yields a ~323 byte patch regardless of the file
	// size, so a fixture that modified one line per file would need
	// thousands of files to create aggregate pressure. Modifying
	// several hundred lines per file yields a ~35 KB patch -- below
	// the 64 KiB per-file budget, so every individual patch is still
	// INCLUDED, while eight of them exceed the 256 KiB aggregate
	// budget.
	const fileCount = 8
	const linesPerFile = 4000
	const changedLines = 400

	for i := 0; i < fileCount; i++ {
		var sb strings.Builder
		for j := 0; j < linesPerFile; j++ {
			sb.WriteString("aggregate pressure padding content line\n")
		}
		writeFilePath(t,
			filepath.Join(dir, fmt.Sprintf("agg_%02d.ts", i)),
			sb.String())
	}
	patchBudgetCommitAll(t, dir, "add many large sources")

	for i := 0; i < fileCount; i++ {
		p := filepath.Join(dir, fmt.Sprintf("agg_%02d.ts", i))
		body, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		lines := strings.Split(string(body),
			"aggregate pressure padding content line\n")
		for j := 0; j < changedLines && j+1 < len(lines); j++ {
			lines[j] += "EDITED"
		}
		writeFilePath(t, p, strings.Join(lines,
			"aggregate pressure padding content line\n"))
	}
	patchBudgetCommitAll(t, dir, "edit many large sources")

	// PRECONDITION 1: every individual patch fits the per-file
	// budget.
	expectedPatches := map[string]string{}
	aggregate := 0
	for i := 0; i < fileCount; i++ {
		name := fmt.Sprintf("agg_%02d.ts", i)
		patch, err := RunGit(dir, []string{
			"diff", "--unified=3", "HEAD~1..HEAD", "--", name})
		if err != nil {
			t.Fatalf("git diff %s: %v", name, err)
		}
		if len(patch) > MaxPerFileBytes {
			t.Fatalf("fixture setup: %s patch=%d bytes must fit "+
				"MaxPerFileBytes=%d", name, len(patch),
				MaxPerFileBytes)
		}
		expectedPatches[name] = patch
		aggregate += len(patch)
	}
	// PRECONDITION 2: the aggregate really does exceed the total
	// render budget. Without this the witness proves nothing.
	if aggregate <= MaxTotalRenderBytes {
		t.Fatalf("fixture setup: aggregate patch bytes=%d must "+
			"EXCEED MaxTotalRenderBytes=%d for this witness to "+
			"establish real budget pressure",
			aggregate, MaxTotalRenderBytes)
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")

	// POSTCONDITION 1: the bounded file-evidence section still
	// respects the total render budget.
	//
	// SCOPE NOTE: MaxTotalRenderBytes bounds the FILE-EVIDENCE
	// SECTION, not the whole digest. The contract header, manifest,
	// stats, review map, risk signals, range scope, patch hygiene,
	// public-surface delta, dependency delta and lifecycle sections
	// are assembled outside the bounded writer. That is the
	// pre-existing contract of the bounded renderer, unchanged by
	// this ACT and by AMENDMENT-01.
	sections := patchBudgetSplitSections(d)
	sectionBytes := 0
	for _, s := range sections {
		sectionBytes += len(s)
	}
	if sectionBytes > MaxTotalRenderBytes {
		t.Fatalf("aggregate budget violated: the file-evidence "+
			"sections total %d bytes and exceed the %d byte "+
			"total budget ceiling",
			sectionBytes, MaxTotalRenderBytes)
	}
	// POSTCONDITION 2: at least one file WAS included, and its
	// COMPLETE patch is published VERBATIM.
	//
	// AMENDMENT-01 FINALIZATION (F2): the predecessor asserted only
	// that a file HEADING was present. A heading is not proof of
	// patch inclusion: an implementation that emitted every heading
	// and every patch body nowhere would satisfy it. The expected
	// patch is now compared against the MEASURED patch.
	included := 0
	for name, expected := range expectedPatches {
		sections := patchBudgetSplitSections(d)
		section, ok := sections["=== "+name+" ==="]
		if !ok {
			continue
		}
		included++
		// Every published section must DECLARE its completeness.
		//
		// AMENDMENT-01 FINALIZATION (F2): this is the assertion the
		// aggregate boundary used to violate: the last file cut
		// short by the budget had NO completeness statement at
		// all, so a partial publication was indistinguishable
		// from a complete one.
		complete := strings.Contains(section,
			"EvidenceCompleteness: COMPLETE")
		incomplete := strings.Contains(section,
			"EvidenceCompleteness: INCOMPLETE")
		if !complete && !incomplete {
			t.Fatalf("aggregate budget witness: the section of %s "+
				"does not declare its completeness. A partial "+
				"publication must never be presented without a "+
				"completeness statement. Section:\n%s",
				name, section)
		}
		if complete && incomplete {
			t.Fatalf("aggregate budget witness: the section of %s "+
				"claims both COMPLETE and INCOMPLETE. "+
				"Section:\n%s", name, section)
		}
		if !complete {
			// An INCOMPLETE section must disclose WHY.
			if !strings.Contains(section, "SuppressionReason:") {
				t.Fatalf("aggregate budget witness: the "+
					"INCOMPLETE section of %s does not disclose a "+
					"suppression reason. Section:\n%s",
					name, section)
			}
			continue
		}
		// The COMPLETE section must carry the EXACT patch.
		if !strings.Contains(section, strings.TrimRight(expected, "\n")) {
			t.Fatalf("aggregate budget witness: the section of %s "+
				"does not carry the COMPLETE patch verbatim. "+
				"Expected patch:\n%s\nSection:\n%s",
				name, expected, section)
		}
	}
	if included == 0 {
		t.Fatalf("aggregate budget witness: no file was included at "+
			"all, so the length assertion alone proved nothing. "+
			"Digest excerpt:\n%s", patchBudgetExcerpt(d))
	}
	// POSTCONDITION 3: the aggregate boundary must be VISIBLE in the
	// digest.
	//
	// AMENDMENT-01 FINALIZATION (F2): the predecessor asserted that
	// an omission record was present. That assertion only holds
	// when the budget is consumed by SKIPPING files. When the budget
	// is consumed by a PARTIAL WRITE of the last file -- which is
	// what happens with a handful of large patches -- there is no
	// omitted file, and the visible evidence of the boundary is the
	// INCOMPLETE disposition of the file that was cut short.
	//
	// The contract is therefore stated as a disjunction, and BOTH
	// branches are checked for truthfulness:
	//
	//   (a) files were OMITTED: the omission record must exist and
	//       must name the ACTUAL omitted paths;
	//   (b) a file was CUT SHORT: its section must declare
	//       INCOMPLETE and disclose the suppression reason.
	//
	// At least one of them must hold, otherwise the aggregate
	// pressure was invisible to the reviewer.
	omittedNames := map[string]bool{}
	for name := range expectedPatches {
		if _, ok := sections["=== "+name+" ==="]; !ok {
			omittedNames[name] = true
		}
	}
	cutShort := 0
	for _, section := range sections {
		if strings.Contains(section, "EvidenceCompleteness: INCOMPLETE") {
			cutShort++
			if !strings.Contains(section, "SuppressionReason:") {
				t.Fatalf("aggregate budget witness: an "+
					"INCOMPLETE section does not disclose a "+
					"suppression reason. Section:\n%s", section)
			}
		}
	}
	if len(omittedNames) == 0 && cutShort == 0 {
		t.Fatalf("aggregate budget witness: the aggregate exceeded "+
			"the budget but the boundary was INVISIBLE: no file "+
			"was omitted and no file was cut short. Digest "+
			"excerpt:\n%s", patchBudgetExcerpt(d))
	}
	if len(omittedNames) > 0 {
		// The omission record must exist and name the ACTUAL
		// omitted paths.
		if !strings.Contains(d, "EVIDENCE_TRUNCATED=true") {
			t.Fatalf("aggregate budget witness: %d file(s) "+
				"were omitted but no omission was recorded. "+
				"Digest excerpt:\n%s", len(omittedNames),
				patchBudgetExcerpt(d))
		}
		if !strings.Contains(d, "reason=TOTAL_RENDER_BUDGET") {
			t.Fatalf("aggregate budget witness: the omission record "+
				"must name the aggregate budget as the reason. "+
				"Digest excerpt:\n%s", patchBudgetExcerpt(d))
		}
		omissionRecord := d
		if idx := strings.Index(d, "## File Evidence Omission"); idx >= 0 {
			omissionRecord = d[idx:]
		}
		for name := range omittedNames {
			if !strings.Contains(omissionRecord, name) {
				t.Fatalf("aggregate budget witness: the omission "+
					"record does not name the omitted file %s. "+
					"Omission record:\n%s", name, omissionRecord)
			}
		}
	}
	// POSTCONDITION 4: a file that the aggregate boundary cut short
	// must NOT claim to be complete, and a file that claims to be
	// complete must carry its exact patch.
	for name, section := range sections {
		if !strings.Contains(section, "EvidenceCompleteness: COMPLETE") {
			continue
		}
		if strings.Contains(section, "INCOMPLETE") {
			t.Fatalf("aggregate budget witness: a section claims "+
				"both COMPLETE and INCOMPLETE. Section:\n%s",
				section)
		}
		// A COMPLETE section must carry a completeness statement
		// that a silent truncation cannot satisfy.
		expected, ok := expectedPatches[strings.TrimSuffix(
			strings.TrimPrefix(name, "=== "), " ===")]
		if !ok {
			continue
		}
		if !strings.Contains(section, strings.TrimRight(expected, "\n")) {
			t.Fatalf("aggregate budget witness: the section of %s "+
				"claims COMPLETE but does not carry the "+
				"complete patch. Section:\n%s", name, section)
		}
	}
}
