// SPDX-License-Identifier: Apache-2.0

// Package digest: range_scope_classifier_test.go provides unit tests
// for the range-scope classifier pure function.
package digest

import "testing"

func TestClassifyRangeScope_Boundaries(t *testing.T) {
	tests := []struct {
		name         string
		filesChanged int
		commitCount  int
		mergeCount   int
		want         RangeTargetedness
	}{
		// NORMAL: below all thresholds
		{"zero_files", 0, 0, 0, RangeTargetednessNormal},
		{"at_broad_file_threshold", 500, 0, 0, RangeTargetednessNormal},
		{"at_broad_commit_threshold", 1, 100, 0, RangeTargetednessNormal},
		{"at_broad_merge_file_threshold", 250, 0, 1, RangeTargetednessNormal},
		{"small_files_small_commits", 10, 5, 0, RangeTargetednessNormal},

		// BROAD: exceeds broad thresholds
		{"exceeds_broad_file_threshold", 501, 0, 0, RangeTargetednessBroad},
		{"exceeds_broad_commit_threshold", 1, 101, 0, RangeTargetednessBroad},
		{"exceeds_broad_merge_file_threshold", 251, 0, 1, RangeTargetednessBroad},
		{"at_extreme_threshold", 1000, 0, 0, RangeTargetednessBroad}, // 1000 > 500 = BROAD

		// EXTREME: exceeds extreme threshold (>1000)
		{"exceeds_extreme_threshold", 1001, 0, 0, RangeTargetednessExtreme},

		// EXTREME takes precedence over BROAD
		{"extreme_with_high_commits", 1001, 200, 5, RangeTargetednessExtreme},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyRangeScope(tt.filesChanged, tt.commitCount, tt.mergeCount)
			if got != tt.want {
				t.Errorf("classifyRangeScope(%d, %d, %d) = %s, want %s",
					tt.filesChanged, tt.commitCount, tt.mergeCount, got, tt.want)
			}
		})
	}
}

func TestParseRangeSpec(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantL   string
		wantR   string
		wantErr bool
	}{
		{"valid_oid_range", "abc123..def456", "abc123", "def456", false},
		{"valid_head_range", "HEAD~1..HEAD", "HEAD~1", "HEAD", false},
		{"valid_with_spaces", " abc123 .. def456 ", "abc123", "def456", false},
		{"symmetric_difference_rejected", "A...B", "", "", true},
		{"missing_dotdot", "abc123", "", "", true},
		{"empty_left", "..abc123", "", "", true},
		{"empty_right", "abc123..", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, r, err := parseRangeSpec(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseRangeSpec(%q) = (%q, %q, nil), want error", tt.input, l, r)
				}
			} else {
				if err != nil {
					t.Errorf("parseRangeSpec(%q) returned error: %v", tt.input, err)
				}
				if l != tt.wantL || r != tt.wantR {
					t.Errorf("parseRangeSpec(%q) = (%q, %q), want (%q, %q)",
						tt.input, l, r, tt.wantL, tt.wantR)
				}
			}
		})
	}
}
