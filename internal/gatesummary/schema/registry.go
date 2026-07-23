package schema

import (
	"fmt"
	"io"
	"sort"
)

// Descriptor describes one supported schema version for the CLI
// introspection surface. The fields are descriptive only; they are not
// embedded inside the JSON Schema itself.
type Descriptor struct {
	// Version is the canonical, case-sensitive version name.
	Version Version
	// Status is the CLI metadata label (supported or current).
	Status Status
	// SchemaID is the stable URN identifier of the JSON Schema.
	SchemaID string
}

// Descriptor returns the descriptor for a single version. Unknown
// versions produce an error so the CLI can fail closed.
func DescriptorFor(v Version) (Descriptor, error) {
	switch v {
	case VersionV1:
		return Descriptor{
			Version:  VersionV1,
			Status:   StatusSupported,
			SchemaID: SchemaIDV1,
		}, nil
	case VersionV2:
		return Descriptor{
			Version:  VersionV2,
			Status:   StatusCurrent,
			SchemaID: SchemaIDV2,
		}, nil
	}
	return Descriptor{}, &UnknownVersionError{Version: v}
}

// List returns a fresh descriptor slice for every supported schema,
// sorted by Version lexicographically. The slice is owned by the caller
// and may be mutated freely; the underlying schema bytes are not
// exposed through this slice.
func List() []Descriptor {
	versions := []Version{VersionV1, VersionV2}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	out := make([]Descriptor, 0, len(versions))
	for _, v := range versions {
		d, _ := DescriptorFor(v)
		out = append(out, d)
	}
	return out
}

// UnknownVersionError is returned when a request names a version that
// is not part of the closed version set. It is a typed error so callers
// can distinguish unknown-version failures from other operational
// failures.
type UnknownVersionError struct {
	// Version is the textual version the user requested.
	Version Version
}

// Error implements the error interface.
func (e *UnknownVersionError) Error() string {
	return fmt.Sprintf("gate-summary schema: unknown version %q", string(e.Version))
}

// Bytes returns a clone of the embedded schema bytes for the given
// version. The returned slice is owned by the caller; mutating it does
// not affect the embedded authority. Unknown versions return a typed
// *UnknownVersionError.
func Bytes(v Version) ([]byte, error) {
	name := schemaFileName(v)
	if name == "" {
		return nil, &UnknownVersionError{Version: v}
	}
	raw, err := files.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("gate-summary schema: read embedded %s: %w", name, err)
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out, nil
}

// WriteExact writes the complete byte slice to dst and returns
// io.ErrShortWrite if the destination reports a partial success.
// The check is required because the io.Writer contract permits Write
// to accept a prefix of the slice and still return a non-nil error.
//
// A failing destination may therefore observe a prefix of the bytes
// before reporting failure. Callers that need atomic, never-observed
// semantics must pipe the output through a buffer first.
func WriteExact(dst io.Writer, data []byte) error {
	n, err := dst.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

// Write writes the exact embedded schema bytes for the given version
// to dst through the checked WriteExact helper. A short write or
// writer error is propagated. Unknown versions return a typed
// *UnknownVersionError.
//
// A failing destination may observe a prefix of the schema bytes
// before reporting failure; the contract does not guarantee atomic
// output to hostile destinations.
func Write(v Version, dst io.Writer) error {
	data, err := Bytes(v)
	if err != nil {
		return err
	}
	if err := WriteExact(dst, data); err != nil {
		// Preserve the requested version in the wrapped error so
		// callers can distinguish the underlying writer failure
		// from the unknown-version failure.
		return fmt.Errorf("gate-summary schema: write %s: %w", string(v), err)
	}
	return nil
}

// MustBytes returns the embedded schema bytes for the given version
// and panics on error. It is intended for tests and for callers that
// have already validated the version.
func MustBytes(v Version) []byte {
	b, err := Bytes(v)
	if err != nil {
		panic(err)
	}
	return b
}

// IsUnknownVersion reports whether err is an *UnknownVersionError so
// callers can map the failure to a CLI exit code without exposing the
// concrete error type.
func IsUnknownVersion(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*UnknownVersionError)
	return ok
}
