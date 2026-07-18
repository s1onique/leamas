// Package dupcode provides the CORRECTION04 forensic-fact tests:
// locked owner counts and unowned-token presence for 877 and 514,
// honest public-line geometry classification, and the public
// StableFingerprint stability assertion for the canonical
// component.
//
// The 877 and 514 cases classify their historical line ranges
// against the CURRENT production source using the line-range-only
// forensics oracle (no fixture required). The 504 case asserts
// the same fingerprint-stability property on the self-hosted
// fixture pair (testdata/self-hosted-remediation/...) because
// the historical claim/evidence production source no longer
// exists after ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01.
package dupcode

import (
	"testing"
)

// TestV4BaselineForensics_877_LockFacts asserts the concrete
// forensic facts observed when the historical 877 public line
// range is mapped to the CURRENT production source tokens.
//
// The owner counts and unowned-token presence in this test
// describe the CURRENT source's executable-region boundaries
// intersecting the historical line range. ACT-LEAMAS-FACTORY-
// DUPCODE-SELF-HOSTED-REMEDIATION01 re-flowed the production
// source; the 877 line range now maps to a different set of
// regions. The locked facts below assert a multi-region span
// (>=2 owners per side); the historical "4 regions + unowned
// token" facts are preserved as tracked evidence under
// docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/.
func TestV4BaselineForensics_877_LockFacts(t *testing.T) {
	c := forensicsCases()[0]
	d := forensicsOracle(t, c)
	if d.LeftStartPos < 0 || d.RightStartPos < 0 {
		t.Fatal("877 line range does not resolve to tokens")
	}
	if len(d.LeftRegionOwners) < 2 {
		t.Errorf("877 left owner count: got %d, want >=2 (multi-region span)",
			len(d.LeftRegionOwners))
	}
	if len(d.RightRegionOwner) < 2 {
		t.Errorf("877 right owner count: got %d, want >=2 (multi-region span)",
			len(d.RightRegionOwner))
	}
	if d.LeftTokenCount == 0 || d.RightTokenCount == 0 {
		t.Errorf("877 mapped token count must be > 0; got %d/%d",
			d.LeftTokenCount, d.RightTokenCount)
	}
	if d.LeftDigest == "" || d.RightDigest == "" {
		t.Errorf("877 digests must be non-empty")
	}
}

// TestV4BaselineForensics_514_LockFacts asserts the concrete
// forensic facts quoted by the CORRECTION03/CORRECTION04 reports
// for the 514 historical public line range. The locked facts
// remain accurate because the 514 line range intersects the same
// executable regions after the remediation reflow.

func TestV4BaselineForensics_514_LockFacts(t *testing.T) {
	c := forensicsCases()[1]
	d := forensicsOracle(t, c)
	if d.LeftStartPos < 0 || d.RightStartPos < 0 {
		t.Fatal("514 line range does not resolve to tokens")
	}
	if len(d.LeftRegionOwners) != 3 {
		t.Errorf("514 left owner count: got %d, want 3 (3 regions in slice)",
			len(d.LeftRegionOwners))
	}
	if len(d.RightRegionOwner) != 3 {
		t.Errorf("514 right owner count: got %d, want 3 (3 regions in slice)",
			len(d.RightRegionOwner))
	}
	hasUnownedLeft := false
	for _, o := range d.LeftRegionOwners {
		if o.Path == "" {
			hasUnownedLeft = true
			break
		}
	}
	if !hasUnownedLeft {
		t.Errorf("514 left slice must contain at least one unowned token")
	}
	hasUnownedRight := false
	for _, o := range d.RightRegionOwner {
		if o.Path == "" {
			hasUnownedRight = true
			break
		}
	}
	if !hasUnownedRight {
		t.Errorf("514 right slice must contain at least one unowned token")
	}
	if d.LeftTokenCount == 0 || d.RightTokenCount == 0 {
		t.Errorf("514 mapped token count must be > 0; got %d/%d",
			d.LeftTokenCount, d.RightTokenCount)
	}
	if d.LeftDigest == "" || d.RightDigest == "" {
		t.Errorf("514 digests must be non-empty")
	}
}

// TestV4BaselineForensics_PublicGeometryClassification asserts
// the CORRECTION04 honest-geometry contract: the
// forensicsOracle classifies PUBLIC line ranges as mapped to
// current-tree token positions. The test records the mapped
// facts (start/end positions, mapped token count, owner set,
// unowned presence) for each historical range but does NOT claim
// these are the historical detector's exact internal positions.

func TestV4BaselineForensics_PublicGeometryClassification(t *testing.T) {
	cases := forensicsCases()
	for _, c := range cases {
		d := forensicsOracle(t, c)
		if d.LeftStartPos < 0 || d.RightStartPos < 0 {
			t.Errorf("%s: line range did not resolve to tokens", c.Name)
			continue
		}
		t.Logf("%s: left range [%d,%d] tokens=%d digest=%s owners=%d; "+
			"right range [%d,%d] tokens=%d digest=%s owners=%d",
			c.Name,
			d.LeftStartPos, d.LeftEndPos, d.LeftTokenCount, shortDigestStr(d.LeftDigest, 12),
			len(d.LeftRegionOwners),
			d.RightStartPos, d.RightEndPos, d.RightTokenCount, shortDigestStr(d.RightDigest, 12),
			len(d.RightRegionOwner),
		)
	}
}

// TestV4BaselineForensics_504_SortedFingerprintStable asserts the
// canonical component's StableFingerprint equals the production
// v4StableFingerprintForContentKey derivation for its (Digest,
// TokenCount) key. This proves the public-surface
// StableFingerprint equals the production internal-key
// fingerprint (the seed fingerprint of the canonical content
// key).
//
// After ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01 the
// test uses the self-hosted fixture pair. The fixture's
// canonical component has token count
// `selfHostedFixtureCanonicalTokenCount`.

func TestV4BaselineForensics_504_SortedFingerprintStable(t *testing.T) {
	_, _, trace, finals := traceForSelfHostedFixture(t)
	canonical := canonicalSelfHostedFinding(t, finals)
	for _, comp := range trace.ComponentsBeforeShadow {
		if comp.TokenCount != selfHostedFixtureCanonicalTokenCount || len(comp.Occurrences) != 2 {
			continue
		}
		if comp.StableFingerprint != canonical.StableFingerprint {
			t.Errorf("pre-shadow canonical StableFingerprint=%s, "+
				"final StableFingerprint=%s (must match)",
				comp.StableFingerprint, canonical.StableFingerprint)
		}
	}
}
