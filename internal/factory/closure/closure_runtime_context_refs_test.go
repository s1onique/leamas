// SPDX-License-Identifier: Apache-2.0

package closure

// closure_runtime_context_refs_test.go provides the
// refs authority umbrellas required by
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02-A.
//
// The wire format is the explicit NUL+LF pair that production
// `git for-each-ref --format=%(objectname)%00%(refname)%00`
// emits: each record is `<oid>\x00<refname>\x00\n`. The
// parser enforces a terminal NUL+LF so a half-record
// cannot slip through.

import (
	"context"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
)

func TestClosureRefsRowsMatrix(t *testing.T) {
	t.Parallel()
	const validOID = "0123456789abcdef0123456789abcdef01234567"
	const validName = "refs/heads/main"
	const otherName = "refs/heads/feature"
	rows := []struct {
		name    string
		raw     string
		isValid bool
	}{
		{"valid single record", validOID + "\x00" + validName + "\x00\n", true},
		{"valid empty repo", "", true},
		{"valid two records", validOID + "\x00" + validName + "\x00\n" + validOID + "\x00" + otherName + "\x00\n", true},
		{"missing final NUL+LF", validOID + "\x00" + validName, false},
		{"missing final LF", validOID + "\x00" + validName + "\x00", false},
		{"truncated after NUL", validOID + "\x00", false},
		{"only NUL", "\x00", false},
		{"only LF", "\n", false},
		{"odd field count", validOID + "\x00" + validName + "\x00\n" + validOID + "\x00\n", false},
		{"duplicate ref", validOID + "\x00" + validName + "\x00\n" + validOID + "\x00" + validName + "\x00\n", false},
		{"empty object", "\x00" + validName + "\x00\n", false},
		{"empty refname", validOID + "\x00\x00\n", false},
		{"invalid OID prefix", "not-hex-" + validOID + "\x00" + validName + "\x00\n", false},
		{"short OID", validOID[:20] + "\x00" + validName + "\x00\n", false},
		{"long OID", validOID + "extra" + "\x00" + validName + "\x00\n", false},
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			records, diags := parseNulFramedRefs([]byte(row.raw))
			if row.isValid {
				if len(diags) > 0 {
					t.Fatalf("valid row should not produce diagnostics: %+v", diags)
				}
				return
			}
			if len(diags) == 0 {
				t.Fatalf("malformed row %q must reject; got %d records", row.name, len(records))
			}
			if !diags.HasCode(V2CodeCallerStateUnavailable) {
				t.Fatalf("malformed row must fail closed with caller_state_unavailable, got %v", diags.Codes())
			}
		})
	}
}

func TestClosureRefsEmptyToNonEmptyProvesCallerRefsChanged(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	git := realGitClient{}
	beforeBytes, _, _, ok := snapshotCallerRefs(context.Background(), git, dir)
	if !ok {
		t.Fatalf("before refs snapshot must be available")
	}
	mustRunGit(t, dir, "checkout", "-b", "drift-add")
	afterBytes, _, _, ok := snapshotCallerRefs(context.Background(), git, dir)
	if !ok {
		t.Fatalf("after refs snapshot must be available")
	}
	if beforeBytes == afterBytes {
		t.Fatalf("before/after drift must produce different bytes; both %q", beforeBytes)
	}
	if !strings.HasSuffix(afterBytes, "\x00\n") {
		t.Fatalf("after bytes must end with terminal NUL+LF, got %q", afterBytes)
	}
	out, err := execution.RunGit(context.Background(), dir, "for-each-ref",
		"--format=%(objectname)%00%(refname)%00", "--sort=refname")
	if err != nil {
		t.Fatalf("direct for-each-ref: %v", err)
	}
	if string(out.Stdout) != afterBytes {
		t.Fatalf("real for-each-ref must match production parser input: %q vs %q",
			string(out.Stdout), afterBytes)
	}
	if beforeBytes == afterBytes {
		t.Fatalf("Diff() must observe byte-level drift")
	}
}
