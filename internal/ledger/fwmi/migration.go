package fwmi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// ErrConflictingReconciliation is returned when two reconciliation records
// claim the same reconciliation identity with different content.
//
// The conflict is rejected BEFORE publication: the second record would
// otherwise overwrite the first one, and a lost reconciliation is a lost
// obligation.
var ErrConflictingReconciliation = fmt.Errorf("conflicting reconciliation identity")

// migrationRecordOf converts one reconciliation record into its durable
// migration record.
//
// The record conserves the IDENTITY of the reconciliation (its kind, its
// resolution and the identities it concerns), its REASON, its source
// provenance and the AFFECTED records. It never resolves an unresolved
// reconciliation: the outcome is conserved verbatim.
func migrationRecordOf(rec ReconciliationRecord, manifest SourceManifest) domain.MigrationRecord {
	identities := append([]string{}, rec.Identities...)
	sort.Strings(identities)
	record := domain.MigrationRecord{
		SchemaVersion:    domain.LedgerSchemaVersion,
		MigrationID:      migrationIDOf(rec),
		SourceRepository: manifest.Repository,
		SourceHead:       manifest.SourceHead,
		ScopedID:         strings.Join(identities, ","),
		Outcome:          rec.Kind,
		Detail:           rec.Resolution + ": " + rec.Reason,
		Revision:         1,
	}
	if len(rec.SourceRefs) > 0 {
		ref := rec.SourceRefs[0]
		record.SourcePath = ref.Path
		record.SourceLine = ref.StartLine
		record.SourceDigest = ref.SHA256
	}
	// ALL the source references of the reconciliation are conserved: a
	// reconciliation that cites several locations loses none of them.
	record.SourceRefs = convertRefsWithHead(rec.SourceRefs, manifest.SourceHead)
	return record
}

// migrationIDOf returns the deterministic identity of a reconciliation
// record.
//
// The identity is DERIVED FROM THE CONTENT of the reconciliation: its kind,
// its resolution and the identities it concerns. Two DISTINCT
// reconciliation records therefore never share an identity, and an exact
// replay of the same record always produces the same identity: the replay
// does not create a second durable record.
func migrationIDOf(rec ReconciliationRecord) string {
	identities := append([]string{}, rec.Identities...)
	sort.Strings(identities)
	parts := []string{
		"kind=" + rec.Kind,
		"resolution=" + rec.Resolution,
		"identities=" + strings.Join(identities, ","),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1e")))
	return "MIGRATION-FWMI-" + hex.EncodeToString(sum[:])[:16]
}

// sameReconciliationContent reports whether two reconciliation records carry
// the same content.
//
// The comparison covers the reason and the byte-exact source provenance: two
// records that claim the same identity with a different reason are a
// CONFLICT, not a repetition.
func sameReconciliationContent(a, b ReconciliationRecord) bool {
	if a.Kind != b.Kind || a.Resolution != b.Resolution || a.Reason != b.Reason {
		return false
	}
	if len(a.Identities) != len(b.Identities) {
		return false
	}
	left := append([]string{}, a.Identities...)
	right := append([]string{}, b.Identities...)
	sort.Strings(left)
	sort.Strings(right)
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	if len(a.SourceRefs) != len(b.SourceRefs) {
		return false
	}
	for i := range a.SourceRefs {
		if a.SourceRefs[i] != b.SourceRefs[i] {
			return false
		}
	}
	return true
}
