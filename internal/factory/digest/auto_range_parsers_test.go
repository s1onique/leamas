// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// auto_range_parsers_test.go covers the markdown table parser used by
// the close-report strategy and the actIDFromPath extractor.
package digest

import (
	"testing"
)

// TestParseImplementationRangeTable covers the markdown parser.
func TestParseImplementationRangeTable(t *testing.T) {
	cases := []struct {
		name     string
		markdown string
		wantBase string
		wantSubj string
		wantOK   bool
	}{
		{
			name: "happy path",
			markdown: `## Implementation Range

| Identity | Commit |
|----------|--------|
| BASE | c9944bf14defdc494fa029c423edbcda8186ac4a |
| C07 | 254c05f1a69caea7f06f66eb01c4d775beefa45b |
| Subject (HEAD) | 9276bce (whitespace fix) |
`,
			wantBase: "c9944bf14defdc494fa029c423edbcda8186ac4a",
			wantSubj: "9276bce",
			wantOK:   true,
		},
		{
			name:     "short SHA in BASE only",
			markdown: "## Implementation Range\n\n| Identity | Commit |\n| BASE | 1234567 |\n",
			wantBase: "1234567",
			wantOK:   true,
		},
		{
			name:     "missing header",
			markdown: "## Summary\n\nNo table here.\n",
			wantOK:   false,
		},
		{
			name:     "table without BASE",
			markdown: "## Implementation Range\n\n| Identity | Commit |\n| SUBJECT | abc1234 |\n",
			wantOK:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base, subj, ok := parseImplementationRangeTable(tc.markdown)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && base != tc.wantBase {
				t.Fatalf("base = %s, want %s", base, tc.wantBase)
			}
			if ok && subj != tc.wantSubj {
				t.Fatalf("subj = %s, want %s", subj, tc.wantSubj)
			}
		})
	}
}

// TestActIDFromPath covers the path-based extractor.
func TestActIDFromPath(t *testing.T) {
	cases := []struct {
		path string
		want string
		ok   bool
	}{
		{"docs/close-reports/ACT-LEAMAS-FACTORY-FOO.md", "ACT-LEAMAS-FACTORY-FOO", true},
		{"docs/closure-manifests/ACT-LEAMAS-FACTORY-FOO.json", "ACT-LEAMAS-FACTORY-FOO", true},
		{"docs/closure-manifests/ACT-LEAMAS-FACTORY-FOO.attestation.json", "ACT-LEAMAS-FACTORY-FOO", true},
		{"cmd/foo.go", "", false},
		{"docs/closure-manifests/random.txt", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got, ok := actIDFromPath(tc.path)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("got = %q, want %q", got, tc.want)
			}
		})
	}
}
