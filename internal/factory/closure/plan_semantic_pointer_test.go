package closure

import (
	"testing"
)

// TestJSONPointerHelpers tests the RFC 6901 JSON Pointer helpers.
func TestJSONPointerHelpers(t *testing.T) {
	t.Run("jsonPointer empty", func(t *testing.T) {
		got := jsonPointer()
		if got != "" {
			t.Errorf("jsonPointer() = %q, want %q", got, "")
		}
	})

	t.Run("jsonPointer single segment", func(t *testing.T) {
		got := jsonPointer("foo")
		if got != "/foo" {
			t.Errorf("jsonPointer(\"foo\") = %q, want %q", got, "/foo")
		}
	})

	t.Run("jsonPointer multiple segments", func(t *testing.T) {
		got := jsonPointer("foo", "bar", "baz")
		if got != "/foo/bar/baz" {
			t.Errorf("jsonPointer(\"foo\", \"bar\", \"baz\") = %q, want %q", got, "/foo/bar/baz")
		}
	})

	t.Run("jsonPointerIndex positive", func(t *testing.T) {
		got := jsonPointerIndex("checks", 0)
		if got != "/checks/0" {
			t.Errorf("jsonPointerIndex(\"checks\", 0) = %q, want %q", got, "/checks/0")
		}
	})

	t.Run("jsonPointerIndex large index", func(t *testing.T) {
		got := jsonPointerIndex("artifacts", 99)
		if got != "/artifacts/99" {
			t.Errorf("jsonPointerIndex(\"artifacts\", 99) = %q, want %q", got, "/artifacts/99")
		}
	})

	t.Run("jsonPointerIndex negative fails closed", func(t *testing.T) {
		got := jsonPointerIndex("checks", -1)
		if got != "" {
			t.Errorf("jsonPointerIndex(\"checks\", -1) = %q, want %q (fail closed)", got, "")
		}
	})

	t.Run("jsonPointerIndex empty segment fails closed", func(t *testing.T) {
		got := jsonPointerIndex("", 0)
		if got != "" {
			t.Errorf("jsonPointerIndex(\"\", 0) = %q, want %q (fail closed)", got, "")
		}
	})

	t.Run("jsonPointerArgvElement negative check index fails closed", func(t *testing.T) {
		got := jsonPointerArgvElement(-1, 0)
		if got != "" {
			t.Errorf("jsonPointerArgvElement(-1, 0) = %q, want %q (fail closed)", got, "")
		}
	})

	t.Run("jsonPointerArgvElement negative argv index fails closed", func(t *testing.T) {
		got := jsonPointerArgvElement(0, -1)
		if got != "" {
			t.Errorf("jsonPointerArgvElement(0, -1) = %q, want %q (fail closed)", got, "")
		}
	})

	t.Run("jsonPointerCheckID negative index fails closed", func(t *testing.T) {
		got := jsonPointerCheckID(-1, "id")
		if got != "" {
			t.Errorf("jsonPointerCheckID(-1, \"id\") = %q, want %q (fail closed)", got, "")
		}
	})

	t.Run("jsonPointerCheckID empty field fails closed", func(t *testing.T) {
		got := jsonPointerCheckID(0, "")
		if got != "" {
			t.Errorf("jsonPointerCheckID(0, \"\") = %q, want %q (fail closed)", got, "")
		}
	})

	t.Run("jsonPointerArtifactID negative index fails closed", func(t *testing.T) {
		got := jsonPointerArtifactID(-1, "path")
		if got != "" {
			t.Errorf("jsonPointerArtifactID(-1, \"path\") = %q, want %q (fail closed)", got, "")
		}
	})

	t.Run("jsonPointerArtifactID empty field fails closed", func(t *testing.T) {
		got := jsonPointerArtifactID(0, "")
		if got != "" {
			t.Errorf("jsonPointerArtifactID(0, \"\") = %q, want %q (fail closed)", got, "")
		}
	})

	t.Run("jsonPointerCheckID", func(t *testing.T) {
		got := jsonPointerCheckID(0, "id")
		if got != "/checks/0/id" {
			t.Errorf("jsonPointerCheckID(0, \"id\") = %q, want %q", got, "/checks/0/id")
		}
	})

	t.Run("jsonPointerArtifactID", func(t *testing.T) {
		got := jsonPointerArtifactID(1, "path")
		if got != "/artifacts/1/path" {
			t.Errorf("jsonPointerArtifactID(1, \"path\") = %q, want %q", got, "/artifacts/1/path")
		}
	})

	t.Run("jsonPointerArgvElement", func(t *testing.T) {
		got := jsonPointerArgvElement(0, 0)
		if got != "/checks/0/argv/0" {
			t.Errorf("jsonPointerArgvElement(0, 0) = %q, want %q", got, "/checks/0/argv/0")
		}
	})
}

// TestJSONPointerToken tests RFC 6901 token encoding.
func TestJSONPointerToken(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"foo", "foo"},
		{"foo/bar", "foo~1bar"},
		{"foo~bar", "foo~0bar"},
		{"foo/bar~baz", "foo~1bar~0baz"},
		{"a/b~c/d", "a~1b~0c~1d"},
		{"", ""},
		{"~", "~0"},
		{"/", "~1"},
		{"~1", "~01"},
		{"~0", "~00"},
		{"a~1b~0c", "a~01b~00c"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := jsonPointerToken(c.input)
			if got != c.expected {
				t.Errorf("jsonPointerToken(%q) = %q, want %q", c.input, got, c.expected)
			}
		})
	}
}
