package doctrinecompiler

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
)

// ComputeDigest returns the hex-encoded SHA-256 digest of data.
func ComputeDigest(data []byte) ContentDigest {
	sum := sha256.Sum256(data)
	return ContentDigest(hex.EncodeToString(sum[:]))
}

// DigestFile returns the hex-encoded SHA-256 digest of file contents.
func DigestFile(path string) (ContentDigest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return ComputeDigest(data), nil
}

// DigestMap returns a deterministic digest of a sorted list of
// (key, value) pairs. The encoding is "key=value\n" lines joined with
// "\n" between pairs, sorted lexicographically by key.
//
// The encoding is used for derived digests where the canonical bytes
// themselves must remain free of map iteration order effects.
func DigestMap(pairs [][2]string) ContentDigest {
	sorted := make([][2]string, len(pairs))
	copy(sorted, pairs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i][0] != sorted[j][0] {
			return sorted[i][0] < sorted[j][0]
		}
		return sorted[i][1] < sorted[j][1]
	})
	h := sha256.New()
	for i, p := range sorted {
		if i > 0 {
			h.Write([]byte("\n"))
		}
		h.Write([]byte(p[0]))
		h.Write([]byte("="))
		h.Write([]byte(p[1]))
	}
	return ContentDigest(hex.EncodeToString(h.Sum(nil)))
}

// CanonicalJSONEncoder writes deterministically ordered JSON.
//
// All object keys are sorted alphabetically. Maps at the top level are
// encoded via stable struct shapes elsewhere; this encoder is reserved
// for hand-built payloads such as the lock file and the project
// selector, where determinism is a hard requirement.
type CanonicalJSONEncoder struct {
	buf []byte
}

// NewCanonicalJSON returns a fresh encoder.
func NewCanonicalJSON() *CanonicalJSONEncoder {
	return &CanonicalJSONEncoder{}
}

// Bytes returns the encoded payload.
func (e *CanonicalJSONEncoder) Bytes() []byte {
	return e.buf
}

// String returns the encoded payload as a string.
func (e *CanonicalJSONEncoder) String() string {
	return string(e.buf)
}

// WriteObject writes a JSON object with keys emitted in sorted order.
func (e *CanonicalJSONEncoder) WriteObject(pairs [][2]any) error {
	sorted := make([][2]any, len(pairs))
	copy(sorted, pairs)
	sort.Slice(sorted, func(i, j int) bool {
		return fmt.Sprintf("%v", sorted[i][0]) < fmt.Sprintf("%v", sorted[j][0])
	})
	e.buf = append(e.buf, '{')
	for i, p := range sorted {
		if i > 0 {
			e.buf = append(e.buf, ',')
		}
		if err := e.writeScalar(p[0]); err != nil {
			return err
		}
		e.buf = append(e.buf, ':')
		if err := e.writeScalar(p[1]); err != nil {
			return err
		}
	}
	e.buf = append(e.buf, '}')
	return nil
}

// WriteArray writes a JSON array from a slice of strings.
func (e *CanonicalJSONEncoder) WriteArray(values []string) {
	e.buf = append(e.buf, '[')
	for i, v := range values {
		if i > 0 {
			e.buf = append(e.buf, ',')
		}
		e.writeString(v)
	}
	e.buf = append(e.buf, ']')
}

// writeScalar writes a scalar JSON value.
func (e *CanonicalJSONEncoder) writeScalar(v any) error {
	switch x := v.(type) {
	case nil:
		e.buf = append(e.buf, "null"...)
	case bool:
		if x {
			e.buf = append(e.buf, "true"...)
		} else {
			e.buf = append(e.buf, "false"...)
		}
	case int:
		e.buf = append(e.buf, fmt.Sprintf("%d", x)...)
	case int64:
		e.buf = append(e.buf, fmt.Sprintf("%d", x)...)
	case string:
		e.writeString(x)
	default:
		return fmt.Errorf("canonical: unsupported scalar type %T", v)
	}
	return nil
}

// writeString writes a JSON string literal.
func (e *CanonicalJSONEncoder) writeString(s string) {
	e.buf = append(e.buf, '"')
	for _, r := range s {
		switch r {
		case '"':
			e.buf = append(e.buf, '\\', '"')
		case '\\':
			e.buf = append(e.buf, '\\', '\\')
		case '\n':
			e.buf = append(e.buf, '\\', 'n')
		case '\r':
			e.buf = append(e.buf, '\\', 'r')
		case '\t':
			e.buf = append(e.buf, '\\', 't')
		default:
			if r < 0x20 {
				e.buf = append(e.buf, fmt.Sprintf("\\u%04x", r)...)
			} else {
				e.buf = append(e.buf, string(r)...)
			}
		}
	}
	e.buf = append(e.buf, '"')
}

// digestError indicates a digest computation failure.
var digestError = errors.New("digest failure")
