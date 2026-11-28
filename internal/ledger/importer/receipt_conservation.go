package importer

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// This file holds the COMPLETE expected-versus-persisted conservation
// comparison of an applied import.
//
// The comparison is a comparison of the CANONICAL RECORDS themselves: the
// expected state is the pre-import canonical state plus the explicitly
// authorized plan, and every record of every family is compared against the
// state that was actually reloaded from the store.
//
//   - the epic scopes,
//   - the work items,
//   - the indexed historical ACT executions,
//   - the durable migration reconciliation records.
//
// No semantic field is EXCLUDED: the lifecycle state, the acceptance state,
// the archival state, the provenance and the revision of a record are part of
// the canonical record, and an unauthorized change to any of them is a
// conservation failure.
//
// A legitimate canonical evolution is not excluded either: it happened
// BEFORE the import, so the pre-import state already carries it and the
// expected state carries it too.

// BuildPersistedReceipt computes the conservation receipt of an applied
// import AGAINST THE ACTUAL PERSISTED STATE.
//
// The planned records are NOT the proof of conservation: the state that was
// actually written to disk is. The receipt compares the EXPECTED state —
// the state before the import plus the applied plan — with the PERSISTED
// state, record by record and field by field.
//
// A persisted state that lost a record, that carries an unexpected record or
// that changed any part of a record reports Conserved=false.
func BuildPersistedReceipt(before domain.Ledger, p Plan, persisted domain.Ledger) Receipt {
	r := BuildReceipt(before, p)
	r.Conserved = true
	r.Discrepancies = []string{}
	r.Discrepancies = append(r.Discrepancies, comparePersistedEpics(before, p, persisted)...)
	r.Discrepancies = append(r.Discrepancies, comparePersistedItems(before, p, persisted)...)
	r.Discrepancies = append(r.Discrepancies, comparePersistedActs(before, p, persisted)...)
	r.Discrepancies = append(r.Discrepancies, comparePersistedMigrations(before, p, persisted)...)
	r.Discrepancies = append(r.Discrepancies, comparePersistedEvidence(before, persisted)...)
	r.Discrepancies = append(r.Discrepancies, comparePersistedDecisions(before, persisted)...)
	r.Discrepancies = append(r.Discrepancies, comparePersistedKnowledge(before, persisted)...)
	if len(r.Discrepancies) > 0 {
		r.Conserved = false
	}
	return r
}

// compareCanonicalRecords compares an expected canonical record with the
// persisted one.
//
// The comparison uses the CANONICAL SERIALIZATION of the record: the records
// are equal when their canonical form is equal. The comparison is therefore
// COMPLETE — every semantic field participates — and it never enumerates a
// selection of fields that a future field could escape from.
//
// The differing fields are reported for the operator: the difference is
// named, not merely counted.
func compareCanonicalRecords(kind, id string, expected, persisted any) []string {
	expectedJSON, expectedErr := json.Marshal(expected)
	persistedJSON, persistedErr := json.Marshal(persisted)
	if expectedErr == nil && persistedErr == nil &&
		string(expectedJSON) == string(persistedJSON) {
		return nil
	}
	out := []string{
		fmt.Sprintf("the persisted %s %s differs from the imported record", kind, id),
	}
	for _, field := range changedFields(expected, persisted) {
		out = append(out, fmt.Sprintf("the persisted %s %s changed its %s", kind, id, field))
	}
	return out
}

// changedFields returns the canonical field paths that differ between two
// canonical records.
//
// The paths use the WIRE names of the records, so the report names the field
// the operator sees in the persisted JSON.
func changedFields(expected, persisted any) []string {
	var out []string
	diffValues(reflect.ValueOf(expected), reflect.ValueOf(persisted), "", &out)
	sort.Strings(out)
	return out
}

// diffValues walks two canonical values and collects the differing paths.
func diffValues(expected, persisted reflect.Value, path string, out *[]string) {
	if !expected.IsValid() || !persisted.IsValid() {
		return
	}
	if expected.Type() != persisted.Type() {
		*out = append(*out, path)
		return
	}
	switch expected.Kind() {
	case reflect.Struct:
		recordType := expected.Type()
		for i := 0; i < recordType.NumField(); i++ {
			field := recordType.Field(i)
			if !field.IsExported() {
				continue
			}
			name := wireName(field)
			if name == "-" {
				continue
			}
			diffValues(expected.Field(i), persisted.Field(i), joinPath(path, name), out)
		}
	case reflect.Slice, reflect.Array:
		if expected.Len() != persisted.Len() {
			*out = append(*out, path)
			return
		}
		for i := 0; i < expected.Len(); i++ {
			diffValues(expected.Index(i), persisted.Index(i),
				fmt.Sprintf("%s[%d]", path, i), out)
		}
	case reflect.Map:
		if expected.Len() != persisted.Len() {
			*out = append(*out, path)
			return
		}
		for _, key := range expected.MapKeys() {
			diffValues(expected.MapIndex(key), persisted.MapIndex(key),
				joinPath(path, fmt.Sprint(key.Interface())), out)
		}
	case reflect.Pointer:
		if expected.IsNil() != persisted.IsNil() {
			*out = append(*out, path)
			return
		}
		if expected.IsNil() {
			return
		}
		diffValues(expected.Elem(), persisted.Elem(), path, out)
	default:
		if !reflect.DeepEqual(expected.Interface(), persisted.Interface()) {
			*out = append(*out, path)
		}
	}
}

// wireName returns the wire name of a struct field.
func wireName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		return field.Name
	}
	return name
}

// joinPath joins a field path with a field name.
func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

// expectedEpics returns the epic scopes the applied import must have left
// in the persisted state.
func expectedEpics(before domain.Ledger, p Plan) map[domain.EpicID]domain.Epic {
	out := map[domain.EpicID]domain.Epic{}
	for _, epic := range before.Epics {
		out[epic.EpicID] = epic
	}
	for _, epic := range p.Epics {
		out[epic.EpicID] = epic
	}
	return out
}

// expectedItems returns the work items the applied import must have left in
// the persisted state.
func expectedItems(before domain.Ledger, p Plan) map[domain.ScopedItemID]domain.WorkItem {
	out := map[domain.ScopedItemID]domain.WorkItem{}
	for _, item := range before.Items {
		out[item.Scoped()] = item
	}
	for _, item := range p.Items {
		out[item.Scoped()] = item
	}
	return out
}

// expectedActs returns the historical ACT executions the applied import must
// have left in the persisted state.
func expectedActs(before domain.Ledger, p Plan) map[domain.ActID]domain.ActExecution {
	out := map[domain.ActID]domain.ActExecution{}
	for _, act := range before.Acts {
		out[act.ActID] = act
	}
	for _, act := range p.Acts {
		out[act.ActID] = act
	}
	return out
}

// expectedMigrations returns the durable reconciliation records the applied
// import must have left in the persisted state.
func expectedMigrations(before domain.Ledger, p Plan) map[string]domain.MigrationRecord {
	out := map[string]domain.MigrationRecord{}
	for _, rec := range before.Migrations {
		out[rec.MigrationID] = rec
	}
	for _, rec := range p.MigrationRecords() {
		out[rec.MigrationID] = rec
	}
	for _, rec := range p.AdditionalMigrations {
		out[rec.MigrationID] = rec
	}
	return out
}

// comparePersistedEpics compares the expected epic scopes with the persisted
// ones.
func comparePersistedEpics(before domain.Ledger, p Plan, persisted domain.Ledger) []string {
	var out []string
	expected := expectedEpics(before, p)
	persistedByID := map[domain.EpicID]domain.Epic{}
	for _, epic := range persisted.Epics {
		persistedByID[epic.EpicID] = epic
	}
	for id, want := range expected {
		stored, ok := persistedByID[id]
		if !ok {
			out = append(out, "the persisted state lost the epic "+string(id))
			continue
		}
		out = append(out, compareCanonicalRecords("epic", string(id), want, stored)...)
	}
	for _, epic := range persisted.Epics {
		if _, ok := expected[epic.EpicID]; !ok {
			out = append(out, "the persisted state contains an unexpected epic "+string(epic.EpicID))
		}
	}
	return out
}

// comparePersistedItems compares the expected work items with the persisted
// ones.
func comparePersistedItems(before domain.Ledger, p Plan, persisted domain.Ledger) []string {
	var out []string
	expected := expectedItems(before, p)
	for id, want := range expected {
		stored, ok := persisted.ItemByScoped(id)
		if !ok {
			out = append(out, "the persisted state lost the work item "+id.String())
			continue
		}
		out = append(out, compareCanonicalRecords("work item", id.String(), want, stored)...)
	}
	known := map[string]bool{}
	for id := range expected {
		known[id.String()] = true
	}
	for _, item := range persisted.Items {
		if !known[item.Scoped().String()] {
			out = append(out, "the persisted state contains an unexpected work item "+item.Scoped().String())
		}
	}
	return out
}

// comparePersistedActs compares the expected historical ACT executions with
// the persisted ones.
func comparePersistedActs(before domain.Ledger, p Plan, persisted domain.Ledger) []string {
	var out []string
	expected := expectedActs(before, p)
	for id, want := range expected {
		stored, ok := persisted.ActByID(id)
		if !ok {
			out = append(out, "the persisted state lost the historical ACT "+string(id))
			continue
		}
		out = append(out, compareCanonicalRecords("historical ACT", string(id), want, stored)...)
	}
	for _, act := range persisted.Acts {
		if _, ok := expected[act.ActID]; !ok {
			out = append(out, "the persisted state contains an unexpected historical ACT "+string(act.ActID))
		}
	}
	return out
}

// comparePersistedMigrations compares the expected durable reconciliation
// records with the persisted ones.
func comparePersistedMigrations(before domain.Ledger, p Plan, persisted domain.Ledger) []string {
	var out []string
	expected := expectedMigrations(before, p)
	persistedByID := map[string]domain.MigrationRecord{}
	for _, rec := range persisted.Migrations {
		persistedByID[rec.MigrationID] = rec
	}
	for id, want := range expected {
		stored, ok := persistedByID[id]
		if !ok {
			out = append(out, "the persisted state lost the migration record "+id)
			continue
		}
		out = append(out, compareCanonicalRecords("migration record", id, want, stored)...)
	}
	for _, rec := range persisted.Migrations {
		if _, ok := expected[rec.MigrationID]; !ok {
			out = append(out, "the persisted state contains an unexpected migration record "+rec.MigrationID)
		}
	}
	return out
}
