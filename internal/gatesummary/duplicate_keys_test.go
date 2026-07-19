package gatesummary

import (
	"sort"
	"testing"
)

func TestDetectDuplicateKeysTopLevel(t *testing.T) {
	data := []byte(`{"a": 1, "a": 2}`)
	hits := detectDuplicateKeys(data)
	if len(hits) != 1 {
		t.Fatalf("expected 1 duplicate, got %d: %+v", len(hits), hits)
	}
	if hits[0].key != "a" {
		t.Fatalf("expected key 'a', got %q", hits[0].key)
	}
	if hits[0].path != "/a" {
		t.Fatalf("expected path '/a', got %q", hits[0].path)
	}
}

func TestDetectDuplicateKeysNested(t *testing.T) {
	data := []byte(`{"outer": {"x": 1, "x": 2}}`)
	hits := detectDuplicateKeys(data)
	if len(hits) != 1 {
		t.Fatalf("expected 1 duplicate, got %d: %+v", len(hits), hits)
	}
	if hits[0].path != "/outer/x" {
		t.Fatalf("expected path '/outer/x', got %q", hits[0].path)
	}
}

func TestDetectDuplicateKeysInsideArray(t *testing.T) {
	data := []byte(`{"items": [{"x": 1, "x": 2}, {"y": 1}]}`)
	hits := detectDuplicateKeys(data)
	if len(hits) != 1 {
		t.Fatalf("expected 1 duplicate, got %d", len(hits))
	}
	if hits[0].path != "/items/0/x" {
		t.Fatalf("expected /items/0/x, got %q", hits[0].path)
	}
}

func TestDetectDuplicateKeysIndependentKeysets(t *testing.T) {
	data := []byte(`{"a": {"k": 1}, "a": {"k": 2}}`)
	hits := detectDuplicateKeys(data)
	if len(hits) < 1 {
		t.Fatalf("expected at least 1 duplicate, got %d", len(hits))
	}
	if hits[0].key != "a" {
		t.Fatalf("expected key 'a', got %q", hits[0].key)
	}
}

func TestDetectDuplicateKeysNone(t *testing.T) {
	data := []byte(`{"a": 1, "b": 2, "c": 3}`)
	hits := detectDuplicateKeys(data)
	if len(hits) != 0 {
		t.Fatalf("expected no duplicates, got %d", len(hits))
	}
}

func TestDetectDuplicateKeysSorted(t *testing.T) {
	data := []byte(`{"z": 1, "z": 2, "y": 1, "y": 2}`)
	hits := detectDuplicateKeys(data)
	if len(hits) != 2 {
		t.Fatalf("expected 2 duplicates, got %d", len(hits))
	}
	keys := []string{hits[0].key, hits[1].key}
	if !sort.StringsAreSorted(keys) {
		t.Fatalf("expected sorted keys, got %v", keys)
	}
}

func TestDetectDuplicateKeysEscapesSpecial(t *testing.T) {
	data := []byte(`{"a/b": 1, "a/b": 2}`)
	hits := detectDuplicateKeys(data)
	if len(hits) != 1 {
		t.Fatalf("expected 1 duplicate, got %d", len(hits))
	}
	if hits[0].path != "/a~1b" {
		t.Fatalf("expected escaped path '/a~1b', got %q", hits[0].path)
	}
}
