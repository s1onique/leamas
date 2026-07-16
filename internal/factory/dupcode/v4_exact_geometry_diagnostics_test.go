// Package dupcode provides exact geometry contract tests for the V4 algorithm.
//
// This file groups the multiplicity diagnostic helpers used by both the
// public and the internal exact-geometry assertions. The helpers
// explicitly report expected N and actual M (with the projection) for
// every key whose multiplicity differs. Diagnostic key iteration is
// sorted for deterministic output.
//
// Sibling files in this contract group:
//
//   - v4_exact_geometry_support_test.go (projection types, assertions,
//     path projector, oracle)
//   - v4_exact_geometry_bodies_test.go
//   - v4_exact_geometry_internal_test.go
//   - v4_exact_geometry_internal_helpers_test.go (v4PipelineInternal)
//   - v4_exact_geometry_determinism_test.go
//   - v4_exact_geometry_ordering_test.go
//   - v4_exact_geometry_path_test.go
package dupcode

import (
	"fmt"
	"sort"
	"testing"
)

// reportMultiplicityDiffs is the truthful multiplicity diagnostic helper
// for the public exact-geometry projection. For every key whose
// multiplicity differs, it reports the explicit expected N and actual
// M (with the projection). Missing and unexpected diagnostics are
// emitted in addition to the multiplicity reports. Diagnostic key
// iteration is sorted for deterministic output.
func reportMultiplicityDiffs(
	t *testing.T,
	label string,
	expected []exactFindingGeometry,
	actual []exactFindingGeometry,
) {
	t.Helper()
	expectedCount := make(map[string]int)
	for _, e := range expected {
		expectedCount[canonicalFindingKey(e)]++
	}
	actualCount := make(map[string]int)
	for _, a := range actual {
		actualCount[canonicalFindingKey(a)]++
	}
	reportMultiplicityDiffsWithKeys(t, label, expectedCount, actualCount,
		func(f exactFindingGeometry) string {
			return fmt.Sprintf("token_count=%d occurrences=%v", f.TokenCount, f.Occurrences)
		},
		expected, actual)
}

// reportInternalMultiplicityDiffs is the truthful multiplicity
// diagnostic helper for the internal exact-geometry projection. Same
// contract as reportMultiplicityDiffs but for internal positions.
func reportInternalMultiplicityDiffs(
	t *testing.T,
	label string,
	expected []exactInternalFindingGeometry,
	actual []exactInternalFindingGeometry,
) {
	t.Helper()
	expectedCount := make(map[string]int)
	for _, e := range expected {
		expectedCount[canonicalInternalFindingKey(e)]++
	}
	actualCount := make(map[string]int)
	for _, a := range actual {
		actualCount[canonicalInternalFindingKey(a)]++
	}
	reportMultiplicityDiffsWithKeys(t, label, expectedCount, actualCount,
		func(f exactInternalFindingGeometry) string {
			return fmt.Sprintf("token_count=%d occurrences=%+v", f.TokenCount, f.Occurrences)
		},
		expected, actual)
}

// reportMultiplicityDiffsWithKeys is the shared diagnostic emitter. It
// sorts keys so output order is deterministic, then reports:
//
//   - explicit multiplicity mismatch (expected N, actual M, projection);
//   - missing diagnostics (every expected projection not covered by
//     actual);
//   - unexpected diagnostics (every actual projection not covered by
//     expected).
//
// The explicit multiplicity report is required: it cannot be replaced by
// missing/unexpected diagnostics.
func reportMultiplicityDiffsWithKeys[T any](
	t *testing.T,
	label string,
	expectedCount, actualCount map[string]int,
	projection func(T) string,
	expected, actual []T,
) {
	t.Helper()

	// Deterministic iteration: collect the union of keys, sort, then emit.
	keys := make(map[string]struct{})
	for k := range expectedCount {
		keys[k] = struct{}{}
	}
	for k := range actualCount {
		keys[k] = struct{}{}
	}
	sortedKeys := make([]string, 0, len(keys))
	for k := range keys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	// Cache projections for each key.
	expectedProj := make(map[string]string)
	for _, e := range expected {
		key := canonicalKeyFromProjection(e, projection)
		expectedProj[key] = projection(e)
	}
	actualProj := make(map[string]string)
	for _, a := range actual {
		key := canonicalKeyFromProjection(a, projection)
		actualProj[key] = projection(a)
	}

	for _, k := range sortedKeys {
		want := expectedCount[k]
		got := actualCount[k]
		if want == got {
			continue
		}
		t.Errorf("%s multiplicity mismatch:\n  expected: %d\n  actual:   %d\n  projection: %s",
			label, want, got, chooseProjection(k, expectedProj, actualProj))
		// Missing and unexpected diagnostics supplement the multiplicity
		// report; they do not replace it.
		if want > got {
			t.Errorf("%s missing (need %d, have %d): %s",
				label, want, got, expectedProj[k])
		}
		if got > want {
			t.Errorf("%s unexpected (have %d, need %d): %s",
				label, got, want, actualProj[k])
		}
	}
}

// canonicalKeyFromProjection re-derives the canonical key from the
// projection value. It is parameterized so that
// reportMultiplicityDiffsWithKeys can be shared between the public and
// the internal exact-geometry projections. The canonical key is the
// canonicalFindingKey (token_count + sorted occurrence geometry) for
// exactFindingGeometry, or the canonicalInternalFindingKey for
// exactInternalFindingGeometry.
func canonicalKeyFromProjection[T any](v T, projection func(T) string) string {
	switch any(v).(type) {
	case exactFindingGeometry:
		return canonicalFindingKey(any(v).(exactFindingGeometry))
	case exactInternalFindingGeometry:
		return canonicalInternalFindingKey(any(v).(exactInternalFindingGeometry))
	default:
		return projection(v)
	}
}

func chooseProjection(key string, expectedProj, actualProj map[string]string) string {
	if p, ok := expectedProj[key]; ok {
		return p
	}
	if p, ok := actualProj[key]; ok {
		return p
	}
	return key
}
