// SPDX-License-Identifier: Apache-2.0

// Package digest: generator_binding_adapter_test.go proves the
// adapter maps raw OID strings into the typed binding inputs
// correctly. The adapter is the only place that translates
// string-typed identities into GeneratorIdentity /
// RepositoryIdentity / DigestAuthoritySubject records.
package digest

import "testing"

// TestAdapterClassifiesValidOIDs proves that the adapter
// recognizes valid full 40-char hex OIDs and sets the validity
// flag to true.
func TestAdapterClassifiesValidOIDs(t *testing.T) {
	const x = "0123456789abcdef0123456789abcdef01234567"
	got := ResolveGeneratorBinding(x, x, x, false)
	if got.Status != GeneratorBindingAuthoritative {
		t.Errorf("valid OIDs: got %q, want %q", got.Status, GeneratorBindingAuthoritative)
	}
	if !got.AuthoritativeForDigest {
		t.Errorf("valid OIDs: must be authoritative")
	}
}

// TestAdapterRejectsUnknownValues proves that unknown / unset
// strings are classified as IDENTITY_UNBOUND, not silently
// promoted to MATCH or AUTHORITATIVE.
func TestAdapterRejectsUnknownValues(t *testing.T) {
	cases := []struct {
		name           string
		generator      string
		head           string
		subject        string
		dirty          bool
		wantStatus     GeneratorBindingStatus
		wantAuthoritve bool
	}{
		// "unknown" is a non-empty placeholder string. The
		// adapter classifies it as EVIDENCE_INVALID (the
		// most conservative verdict) because it is not a
		// valid hex OID. This is the "unknown => fail closed"
		// semantic from ACT §26.
		{
			name:           "all_unknown_placeholder",
			generator:      "unknown",
			head:           "unknown",
			subject:        "unknown",
			dirty:          false,
			wantStatus:     GeneratorBindingEvidenceInvalid,
			wantAuthoritve: false,
		},
		// Empty strings map to missing identity (UNBOUND).
		{
			name:           "all_empty",
			generator:      "",
			head:           "",
			subject:        "",
			dirty:          false,
			wantStatus:     GeneratorBindingIdentityUnbound,
			wantAuthoritve: false,
		},
		// Generator unset (empty) but head + subject set.
		{
			name:           "generator_unset_head_set_subject_set",
			generator:      "",
			head:           "0123456789abcdef0123456789abcdef01234567",
			subject:        "0123456789abcdef0123456789abcdef01234567",
			dirty:          false,
			wantStatus:     GeneratorBindingIdentityUnbound,
			wantAuthoritve: false,
		},
		// Garbage generator is EVIDENCE_INVALID.
		{
			name:           "garbage_generator",
			generator:      "not-an-oid",
			head:           "0123456789abcdef0123456789abcdef01234567",
			subject:        "0123456789abcdef0123456789abcdef01234567",
			dirty:          false,
			wantStatus:     GeneratorBindingEvidenceInvalid,
			wantAuthoritve: false,
		},
		// Short OID (not 40+ hex chars) is EVIDENCE_INVALID.
		{
			name:           "short_oid_generator",
			generator:      "0123456789abcdef",
			head:           "0123456789abcdef0123456789abcdef01234567",
			subject:        "0123456789abcdef0123456789abcdef01234567",
			dirty:          false,
			wantStatus:     GeneratorBindingEvidenceInvalid,
			wantAuthoritve: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveGeneratorBinding(tc.generator, tc.head, tc.subject, tc.dirty)
			if got.Status != tc.wantStatus {
				t.Errorf("Status: got %q, want %q", got.Status, tc.wantStatus)
			}
			if got.AuthoritativeForDigest != tc.wantAuthoritve {
				t.Errorf("AuthoritativeForDigest: got %t, want %t",
					got.AuthoritativeForDigest, tc.wantAuthoritve)
			}
		})
	}
}

// TestAdapterNormalizesCase proves the adapter lowercases valid
// hex OIDs before comparison, mirroring Git's documented
// short-SHA conventions and the existing fullOIDPattern
// normalization in auto_range.go.
func TestAdapterNormalizesCase(t *testing.T) {
	const xMixed = "0123456789AbCdEf0123456789abcdef01234567"
	const xUpper = "0123456789ABCDEF0123456789ABCDEF01234567"

	got := ResolveGeneratorBinding(xMixed, xUpper, xMixed, false)
	if got.Status != GeneratorBindingAuthoritative {
		t.Errorf("mixed-case OIDs should normalize: got %q, want %q",
			got.Status, GeneratorBindingAuthoritative)
	}
}

// TestAdapterDirtyFlagPropagation proves that the dirty flag
// reaches the typed DigestAuthoritySubject record. When the
// generator commit matches HEAD and the digest is dirty, the
// adapter must report DIRTY_SUBJECT_UNBOUND, not AUTHORITATIVE.
func TestAdapterDirtyFlagPropagation(t *testing.T) {
	const x = "0123456789abcdef0123456789abcdef01234567"

	// Same OIDs but dirty=true.
	got := ResolveGeneratorBinding(x, x, x, true)
	if got.Status != GeneratorBindingDirtySubjectUnbound {
		t.Errorf("dirty=true: got %q, want %q",
			got.Status, GeneratorBindingDirtySubjectUnbound)
	}
	if got.AuthoritativeForDigest {
		t.Errorf("dirty=true: MUST NOT be authoritative")
	}

	// Same OIDs but dirty=false.
	clean := ResolveGeneratorBinding(x, x, x, false)
	if clean.Status != GeneratorBindingAuthoritative {
		t.Errorf("dirty=false: got %q, want %q",
			clean.Status, GeneratorBindingAuthoritative)
	}
	if !clean.AuthoritativeForDigest {
		t.Errorf("dirty=false: must be authoritative")
	}
}
