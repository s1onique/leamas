// SPDX-License-Identifier: Apache-2.0

package dupcodeauthority

import (
	"testing"
)

// TestDupcodeAuthorityInvalidSHAVariants proves various SHA formats are rejected.
func TestDupcodeAuthorityInvalidSHAVariants(t *testing.T) {
	invalidSHAs := []string{
		"",
		"abc",
		"0123456789",
		"a71c034",
		"a71c0340dd08a821e66832488a83",
		"a71c0340dd08a821e66832488a83e665ba09f02c0",
		"ABCDEF0000000000000000000000000000000000000",
		"a71c0340dd08a821e66832488a83e665ba09f02g",
	}

	for _, sha := range invalidSHAs {
		if fullCommitOIDRegex.MatchString(sha) {
			t.Errorf("SHA %q should be invalid but matched regex", sha)
		}
	}
}

// TestDupcodeAuthorityValidSHAVariants proves valid SHA formats are accepted.
func TestDupcodeAuthorityValidSHAVariants(t *testing.T) {
	validSHAs := []string{
		"a71c0340dd08a821e66832488a83e665ba09f02c",
		"0000000000000000000000000000000000000000",
		"ffffffffffffffffffffffffffffffffffffffff",
		"0000000000000000000000000000000000000000000000000000000000000000",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}

	for _, sha := range validSHAs {
		if !fullCommitOIDRegex.MatchString(sha) {
			t.Errorf("SHA %q should be valid but did not match regex", sha)
		}
	}
}
