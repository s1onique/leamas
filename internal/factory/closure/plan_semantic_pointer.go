package closure

// plan_semantic_pointer.go provides RFC 6901 JSON Pointer helpers
// for constructing exact diagnostic paths.

// jsonPointer builds an RFC 6901 JSON Pointer from the given path segments.
// Empty segments are encoded as-is, and special characters are escaped per RFC 6901.
func jsonPointer(segments ...string) string {
	if len(segments) == 0 {
		return ""
	}
	result := ""
	for _, seg := range segments {
		result += "/" + jsonPointerToken(seg)
	}
	return result
}

// jsonPointerIndex builds /<segment>/<index> style paths for array elements.
// Negative indices and empty segment names are rejected (fail closed).
func jsonPointerIndex(segment string, index int) string {
	if index < 0 || segment == "" {
		return "" // Fail closed for negative indices or empty segment
	}
	return "/" + jsonPointerToken(segment) + "/" + itoa(index)
}

// jsonPointerCheckID constructs /checks/<index>/<field>.
// Fails closed for negative index or empty field.
func jsonPointerCheckID(index int, field string) string {
	if index < 0 || field == "" {
		return "" // Fail closed
	}
	return jsonPointerIndex("checks", index) + "/" + jsonPointerToken(field)
}

// jsonPointerArtifactID constructs /artifacts/<index>/<field>.
// Fails closed for negative index or empty field.
func jsonPointerArtifactID(index int, field string) string {
	if index < 0 || field == "" {
		return "" // Fail closed
	}
	return jsonPointerIndex("artifacts", index) + "/" + jsonPointerToken(field)
}

// jsonPointerArgvElement constructs /checks/<checkIndex>/argv/<argIndex>.
// Fails closed for negative indices.
func jsonPointerArgvElement(checkIndex, argIndex int) string {
	if checkIndex < 0 || argIndex < 0 {
		return "" // Fail closed for negative indices
	}
	return jsonPointerIndex("checks", checkIndex) + "/argv/" + itoa(argIndex)
}

// jsonPointerToken encodes a JSON Pointer token per RFC 6901.
// The tilde encoding is: ~ -> ~0, / -> ~1.
// Order matters: encode ~ first, then /.
func jsonPointerToken(s string) string {
	// First encode ~ (to avoid encoding ~0 as ~00)
	s = replaceAll(s, "~", "~0")
	// Then encode /
	s = replaceAll(s, "/", "~1")
	return s
}

// replaceAll is a simple string replace helper.
func replaceAll(s, old, new string) string {
	if s == "" || old == "" || old == new {
		return s
	}
	var b []byte
	for i := 0; i < len(s); {
		j := indexString(s, old, i)
		if j < 0 {
			b = append(b, s[i:]...)
			break
		}
		b = append(b, s[i:j]...)
		b = append(b, new...)
		i = j + len(old)
	}
	return string(b)
}

// indexString returns the index of the first occurrence of substr
// in s starting at position i, or -1 if not found.
func indexString(s, substr string, i int) int {
	if i >= len(s) || i < 0 || substr == "" {
		if i >= len(s) || i < 0 {
			return -1
		}
		return 0
	}
	for j := i; j <= len(s)-len(substr); j++ {
		if s[j:j+len(substr)] == substr {
			return j
		}
	}
	return -1
}
