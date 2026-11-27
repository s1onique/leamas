package fwmi

import (
	"strings"
	"testing"
)

// This file qualifies the ADVERSARIAL DISCRIMINATORS that the interchange
// CONTRACT must enforce, against the real loader and adapter.
//
// D1 — a missing source repository must not establish a verified source
//      binding.
// D2 — an unresolved reconciliation record must survive persistence.

// TestD1MissingSourceRepositoryDoesNotEstablishASourceBinding witnesses D1.
//
// The contract: a package whose source manifest names NO repository is
// rejected. Without the repository identity the records would be bound to an
// empty source identity, and that empty binding would be indistinguishable
// from a verified one.
func TestD1MissingSourceRepositoryDoesNotEstablishASourceBinding(t *testing.T) {
	files := validPackage()
	files[ManifestFile] = strings.Replace(validManifest,
		`"repository": "clinemm",`, `"repository": "",`, 1)

	_, err := Load(writePackage(t, files))
	if err == nil {
		t.Fatal("a package with a missing source repository was accepted")
	}
	if !strings.Contains(err.Error(), "source repository") {
		t.Fatalf("the missing source repository was not reported: %v", err)
	}
}

// TestD1MissingSourceRepositoryNeverReachesTheCanonicalBinding witnesses D1
// at the adaptation seam.
//
// The contract: the adapter never substitutes a placeholder repository for a
// missing one. An empty source identity is not a source binding, so it is
// never written into the canonical provenance.
func TestD1MissingSourceRepositoryNeverReachesTheCanonicalBinding(t *testing.T) {
	pkg := &Package{
		Manifest: SourceManifest{SchemaVersion: SchemaVersion, SourceHead: "abc"},
	}
	res, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	for _, item := range res.Board.Items {
		if item.SourceConfidence == "" {
			t.Fatal("the adapted record carries no source confidence")
		}
	}
	// The board source must never be the placeholder of a missing repository.
	if res.Board.Source == "fwmi-interchange" {
		t.Fatal("the adapter substituted a placeholder for the missing source repository")
	}
}

// TestD2UnresolvedReconciliationSurvivesPersistence witnesses D2.
//
// The contract: an UNRESOLVED reconciliation record of the package survives
// as durable migration state. It is never dropped, never resolved by the
// adapter and never downgraded to a console warning.
func TestD2UnresolvedReconciliationSurvivesPersistence(t *testing.T) {
	pkg, err := Load(writePackage(t, validPackage()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	res, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	if len(res.Migrations) != 1 {
		t.Fatalf("the reconciliation records = %d, want 1", len(res.Migrations))
	}
	rec := res.Migrations[0]
	if rec.Outcome != "POSSIBLE_CROSS_LINK" {
		t.Fatalf("the reconciliation kind was not conserved: %q", rec.Outcome)
	}
	if !strings.Contains(rec.Detail, "UNRESOLVED") {
		t.Fatalf("the unresolved resolution was not conserved: %q", rec.Detail)
	}
	if rec.SourceRepository != "clinemm" {
		t.Fatalf("the source repository binding was lost: %q", rec.SourceRepository)
	}
	if rec.SourceHead == "" {
		t.Fatal("the source HEAD binding was lost")
	}
}
