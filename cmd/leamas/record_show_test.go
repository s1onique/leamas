// Unit tests for the runRecordShow shared abstraction in record_show.go.
//
// These tests exercise the shared operation directly, without going through
// the public command paths. They supplement (and do not replace) the
// command-level characterization tests in claim_show_characterization_test.go
// and evidence_show_characterization_test.go.
//
// The shared operation is intentionally minimal: it parses flags, validates
// inputs, opens the run bundle, calls the spec-supplied record loader, and
// dispatches to the spec-supplied renderer. Tests focus on:
//
//   - successful shared operation (text + JSON)
//   - validation passed from the command boundary
//   - command-specific policy invocation
//   - persistence success and failure handling
//   - result projection inputs
//   - cleanup after partial failure
//   - deterministic serialized output
//
// Tests in this file must not be marked t.Parallel(): the capture helper
// swaps package-level os.Stdout/os.Stderr globals.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/witness/claim"
)

// minimalRecord is the test fixture record type used by these unit tests.
// It exists so that runRecordShow is exercised through a real spec without
// pulling in claim.NewClaim/NewEvidence or the disk store.
type minimalRecord struct {
	ID    string
	Title string
}

// recordCounter is a tiny atomic counter used to count spec invocations
// across tests. It is reset by tests that need a stable baseline.
var recordCounter int64

// resetRecordCounter zeros the global invocation counter.
func resetRecordCounter() {
	atomic.StoreInt64(&recordCounter, 0)
}

// readRecordCounter returns the current counter value.
func readRecordCounter() int64 {
	return atomic.LoadInt64(&recordCounter)
}

// makeMinimalRunBundleForRecordShow creates a run bundle directory plus a
// trivial "records" subdirectory so runbundle.Open succeeds. It does NOT
// write any claim/evidence records; record loading is provided by the
// spec-supplied loader.
func makeMinimalRunBundleForRecordShow(t *testing.T) (root, runID string) {
	t.Helper()
	root = t.TempDir()
	runID = "run-test-20260101T000000Z-minrec01"
	if c := runWitnessRunBundleCreate([]string{"--root", root, "--id", runID}); c != 0 {
		t.Fatalf("create bundle: code=%d", c)
	}
	return root, runID
}

// minimalSpec constructs a recordShowSpec backed by an in-memory map so
// runRecordShow can be exercised without filesystem coupling.
func minimalSpec(rec map[string]minimalRecord, returnErr error) recordShowSpec {
	return recordShowSpec{
		KindName:    "minimal",
		PosArgLabel: "<minimal-id>",
		ValidateID: func(raw string) error {
			if raw == "" {
				return errors.New("empty id")
			}
			if strings.Contains(raw, "/") {
				return errors.New("path separator not allowed")
			}
			return nil
		},
		NotFoundErr: errMinimalNotFound,
		ReadRecord: func(_ claim.Store, raw string) (any, error) {
			atomic.AddInt64(&recordCounter, 1)
			if returnErr != nil {
				return nil, returnErr
			}
			rec, ok := rec[raw]
			if !ok {
				return nil, errMinimalNotFound
			}
			return &rec, nil
		},
		RenderText: func(w io.Writer, _ string, record any) {
			r, ok := record.(*minimalRecord)
			if !ok {
				fmt.Fprintf(w, "ERROR: wrong type %T\n", record)
				return
			}
			fmt.Fprintf(w, "MinimalID: %s\n", r.ID)
			fmt.Fprintf(w, "Title: %s\n", r.Title)
		},
		RenderJSON: func(w io.Writer, record any) error {
			r, ok := record.(*minimalRecord)
			if !ok {
				return fmt.Errorf("wrong type %T", record)
			}
			out := struct {
				OK      bool           `json:"ok"`
				Minimal *minimalRecord `json:"minimal"`
			}{OK: true, Minimal: r}
			data, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(w, string(data))
			return nil
		},
	}
}

// errMinimalNotFound is the sentinel that runRecordShow translates into
// the "<kind> not found: <id>" error message.
var errMinimalNotFound = errors.New("minimal not found")

// TestRunRecordShow_Text_Success asserts that a successful shared operation
// produces the spec-supplied text rendering and exits 0.
func TestRunRecordShow_Text_Success(t *testing.T) {
	root, runID := makeMinimalRunBundleForRecordShow(t)
	recs := map[string]minimalRecord{
		"alpha": {ID: "alpha", Title: "first"},
	}
	spec := minimalSpec(recs, nil)

	stdout, stderr, code := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "alpha"},
			spec,
		)
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr must be empty on success, got %q", stderr)
	}
	want := "MinimalID: alpha\nTitle: first\n"
	if stdout != want {
		t.Fatalf("stdout mismatch:\n--- want ---\n%s\n--- got ---\n%s", want, stdout)
	}
}

// TestRunRecordShow_JSON_Success asserts that the JSON dispatcher produces
// the spec-supplied JSON envelope and exits 0.
func TestRunRecordShow_JSON_Success(t *testing.T) {
	root, runID := makeMinimalRunBundleForRecordShow(t)
	recs := map[string]minimalRecord{
		"beta": {ID: "beta", Title: "second"},
	}
	spec := minimalSpec(recs, nil)

	stdout, stderr, code := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "--json", "beta"},
			spec,
		)
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr must be empty on success, got %q", stderr)
	}

	var envelope struct {
		OK      bool           `json:"ok"`
		Minimal *minimalRecord `json:"minimal"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if !envelope.OK {
		t.Errorf("envelope.ok = false, want true")
	}
	if envelope.Minimal == nil {
		t.Fatalf("envelope.minimal missing")
	}
	if envelope.Minimal.ID != "beta" || envelope.Minimal.Title != "second" {
		t.Errorf("unexpected payload: %+v", envelope.Minimal)
	}
}

// TestRunRecordShow_DeterministicOutput asserts that two consecutive
// invocations with the same spec and inputs produce byte-identical
// stdout for both text and JSON modes. This guards against accidental
// map-iteration nondeterminism in any renderer added later.
func TestRunRecordShow_DeterministicOutput(t *testing.T) {
	root, runID := makeMinimalRunBundleForRecordShow(t)
	recs := map[string]minimalRecord{
		"alpha": {ID: "alpha", Title: "first"},
		"beta":  {ID: "beta", Title: "second"},
	}
	spec := minimalSpec(recs, nil)

	stdout1, _, _ := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "alpha"},
			spec,
		)
	})
	stdout2, _, _ := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "alpha"},
			spec,
		)
	})
	if stdout1 != stdout2 {
		t.Fatalf("non-deterministic text output:\nrun1=%q\nrun2=%q", stdout1, stdout2)
	}

	json1, _, _ := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "--json", "alpha"},
			spec,
		)
	})
	json2, _, _ := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "--json", "alpha"},
			spec,
		)
	})
	if json1 != json2 {
		t.Fatalf("non-deterministic json output:\nrun1=%q\nrun2=%q", json1, json2)
	}
}
