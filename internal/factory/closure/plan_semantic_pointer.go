package closure

// plan_semantic_pointer.go provides RFC 6901 JSON Pointer helpers
// for constructing exact diagnostic paths.

// jsonPointerCheckID constructs /checks/<index>/<field>.
func jsonPointerCheckID(index int, field string) string {
	return "/checks/" + itoa(index) + "/" + field
}

// jsonPointerArtifactID constructs /artifacts/<index>/<field>.
func jsonPointerArtifactID(index int, field string) string {
	return "/artifacts/" + itoa(index) + "/" + field
}

// jsonPointerArgvElement constructs /checks/<checkIndex>/argv/<argIndex>.
// This is used for argv element validation errors.
func jsonPointerArgvElement(checkIndex, argIndex int) string {
	return "/checks/" + itoa(checkIndex) + "/argv/" + itoa(argIndex)
}

// jsonPointerToken encodes a JSON Pointer token per RFC 6901.
// The tilde encoding is: ~ -> ~0, / -> ~1.
// Order matters: encode ~ first, then /.
func jsonPointerToken(s string) string {
	// Encode ~ first (to avoid double-encoding existing ~0/~1)
	s = replaceAll(s, "~", "~0")
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
