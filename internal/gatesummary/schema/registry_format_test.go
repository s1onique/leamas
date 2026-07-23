package schema

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

// TestRegistryFilesEndWithOneLF asserts that every embedded schema
// ends with exactly one LF byte. The byte format is part of the
// wire-format contract.
func TestRegistryFilesEndWithOneLF(t *testing.T) {
	for _, v := range []Version{VersionV1, VersionV2} {
		data := MustBytes(v)
		if !lineEndingGuard(data) {
			t.Fatalf("%s does not end with exactly one LF; trailing bytes: %q", v, tailForTest(data))
		}
	}
}

// TestRegistryFilesHaveNoBOM asserts that no embedded schema starts
// with the UTF-8 BOM (U+FEFF).
func TestRegistryFilesHaveNoBOM(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	for _, v := range []Version{VersionV1, VersionV2} {
		data := MustBytes(v)
		if bytes.HasPrefix(data, bom) {
			t.Fatalf("%s starts with a UTF-8 BOM", v)
		}
	}
}

// TestRegistryFilesHaveLFOnly asserts that embedded schemas use only
// LF line endings (no CR).
func TestRegistryFilesHaveLFOnly(t *testing.T) {
	for _, v := range []Version{VersionV1, VersionV2} {
		data := MustBytes(v)
		if bytes.Contains(data, []byte{'\r'}) {
			t.Fatalf("%s contains CR byte; line endings must be LF only", v)
		}
	}
}

// TestRegistryFilesAreValidJSON asserts that the embedded bytes are
// valid JSON documents.
func TestRegistryFilesAreValidJSON(t *testing.T) {
	for _, v := range []Version{VersionV1, VersionV2} {
		data := MustBytes(v)
		var anyV any
		if err := jsonUnmarshal(data, &anyV); err != nil {
			t.Fatalf("%s: not valid JSON: %v", v, err)
		}
	}
}

// TestRegistrySchemaIDsAreStable asserts the literal $id of each
// schema. The URN is the public stable identifier; the contract binds
// the literal string here.
func TestRegistrySchemaIDsAreStable(t *testing.T) {
	for _, v := range []Version{VersionV1, VersionV2} {
		data := MustBytes(v)
		var doc map[string]any
		if err := jsonUnmarshal(data, &doc); err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		gotID, _ := doc["$id"].(string)
		wantID := ""
		switch v {
		case VersionV1:
			wantID = SchemaIDV1
		case VersionV2:
			wantID = SchemaIDV2
		}
		if gotID != wantID {
			t.Errorf("%s $id = %q, want %q", v, gotID, wantID)
		}
		gotSchema, _ := doc["$schema"].(string)
		if gotSchema != "https://json-schema.org/draft/2020-12/schema" {
			t.Errorf("%s $schema = %q, want Draft 2020-12 URI", v, gotSchema)
		}
	}
}

// TestRegistryFilesNoTimestampsOrPaths asserts that the canonical
// schema files do not embed timestamps, host paths, or commit
// identifiers.
func TestRegistryFilesNoTimestampsOrPaths(t *testing.T) {
	for _, v := range []Version{VersionV1, VersionV2} {
		data := MustBytes(v)
		for _, banned := range []string{"/home/", "/Users/", "C:\\", "2024-", "2025-", "2026-", "2027-"} {
			if bytes.Contains(data, []byte(banned)) {
				t.Fatalf("%s contains banned substring %q", v, banned)
			}
		}
	}
}

// TestRegistryWriteRejectsShortWrite asserts that WriteExact (and
// therefore Write) detects a destination that accepts a prefix of
// the bytes and returns a non-nil error. This is the defensive
// short-write path required by the byte-exact contract.
func TestRegistryWriteRejectsShortWrite(t *testing.T) {
	w := &shortWriter{}
	if err := WriteExact(w, []byte("hello world")); err == nil {
		t.Fatalf("WriteExact must reject short writes")
	}
}

// TestRegistryWriteExactEmitsCompletePayload asserts that
// WriteExact writes the entire slice when the destination accepts
// the whole payload.
func TestRegistryWriteExactEmitsCompletePayload(t *testing.T) {
	w := &countingWriter{}
	if err := WriteExact(w, []byte("hello world")); err != nil {
		t.Fatalf("WriteExact: %v", err)
	}
	if w.n != len("hello world") {
		t.Fatalf("WriteExact wrote %d bytes, want %d", w.n, len("hello world"))
	}
}

// TestRegistryWriteExactPropagatesWriterError asserts that a
// destination returning a non-nil error surfaces the error untouched.
func TestRegistryWriteExactPropagatesWriterError(t *testing.T) {
	fail := &failingWriter{err: errAlways}
	if err := WriteExact(fail, []byte("hello")); err == nil {
		t.Fatalf("WriteExact must propagate writer error")
	}
}

// TestRegistryWritePropagatesShortWrite asserts that the version
// of Write used by the CLI also detects short writes.
func TestRegistryWritePropagatesShortWrite(t *testing.T) {
	w := &shortWriter{}
	if err := Write(VersionV1, w); err == nil {
		t.Fatalf("Write must reject short writes")
	}
}

// TestRegistryWritePropagatesWriterError asserts that the
// production Write surfaces a writer failure untouched.
func TestRegistryWritePropagatesWriterError(t *testing.T) {
	fail := &failingWriter{err: errAlways}
	if err := Write(VersionV1, fail); err == nil {
		t.Fatalf("Write must propagate writer error")
	}
}

// TestRegistryWriteRejectsUnknownVersion asserts that Write on an
// unknown version returns a typed *UnknownVersionError without
// touching the destination writer.
func TestRegistryWriteRejectsUnknownVersion(t *testing.T) {
	w := &countingWriter{}
	if err := Write(Version("v3"), w); err == nil {
		t.Fatalf("Write must reject unknown version")
	} else if !IsUnknownVersion(err) {
		t.Fatalf("Write error = %v, want *UnknownVersionError", err)
	}
	if w.n != 0 {
		t.Fatalf("unknown-version Write touched the writer; n=%d", w.n)
	}
}

// --- helpers below ---

// lineEndingGuard returns true when the trimmed bytes end with exactly
// one LF byte.
func lineEndingGuard(data []byte) bool {
	trimmed := bytes.TrimRight(data, "\n")
	if len(trimmed) == len(data) {
		return false
	}
	return len(data)-len(trimmed) == 1
}

// tailForTest returns the last 16 bytes of data, with non-printable
// bytes escaped, for use in diagnostic messages.
func tailForTest(data []byte) string {
	if len(data) <= 16 {
		return string(data)
	}
	tail := data[len(data)-16:]
	var b bytes.Buffer
	for _, c := range tail {
		switch {
		case c == '\n':
			b.WriteString("\\n")
		case c == '\r':
			b.WriteString("\\r")
		case c == '\t':
			b.WriteString("\\t")
		case c >= 0x20 && c < 0x7f:
			b.WriteByte(c)
		default:
			b.WriteString(".")
		}
	}
	return b.String()
}

// failingWriter returns the configured error after recording the
// byte count it was offered.
type failingWriter struct {
	err error
	n   int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	w.n += len(p)
	return 0, w.err
}

// shortWriter accepts a prefix of each input and returns
// io.ErrShortWrite. It is used to verify that the checked write
// helper detects partial success.
type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return 1, nil
}

// countingWriter records every byte handed to it without producing
// any output.
type countingWriter struct {
	n int
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.n += len(p)
	return len(p), nil
}

// jsonUnmarshal is a thin convenience wrapper for json.Unmarshal that
// keeps the test file readable. It uses the standard library directly.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// errAlways is a sentinel error used by writer tests that require a
// non-nil failure to be returned on every Write call.
var errAlways = errors.New("writer failed")
