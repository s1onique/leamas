package gatesummary

// Version is the gate-summary schema discriminator.
type Version uint8

const (
	// Version1 is the frozen v1 wire contract.
	Version1 Version = 1
	// Version2 is the frozen v2 wire contract.
	Version2 Version = 2
	// Version3 is the InDeep Factory Gate Summary v3 wire contract.
	//
	// V3 is a strict superset of v2: every v2 field is present in v3,
	// and v3 additionally carries the InDeep canonical `counts`
	// aggregate block, per-check evidence identifiers (`id`/`order`/
	// `execution_class`/`argv`/`implementation`), the runner's
	// `deadline_exceeded` flag, both runner-classified and raw process
	// exit codes (`canonical_exit_code`/`raw_process_exit_code`), the
	// captured `termination_signal`, byte-exact stdout/stderr metrics,
	// truncation flags, and three top-level SHA-256 evidence bindings
	// (`registry_sha256`, `events_sha256`, `transcript_sha256`).
	Version3 Version = 3
)

// String returns the canonical "v1"/"v2"/"v3" textual form.
func (v Version) String() string {
	switch v {
	case Version1:
		return "v1"
	case Version2:
		return "v2"
	case Version3:
		return "v3"
	default:
		return "v?"
	}
}

// Document is the sealed, version-specific wire document produced by a
// successful Decode call. Exactly one of v1, v2, or v3 is populated.
type Document struct {
	v1 *V1Summary
	v2 *V2Summary
	v3 *V3Summary
}

// Version returns the schema version of the document.
func (d Document) Version() Version {
	if d.v1 != nil {
		return Version1
	}
	if d.v2 != nil {
		return Version2
	}
	if d.v3 != nil {
		return Version3
	}
	return 0
}

// V1 returns the v1 wire summary and true when this document is v1.
func (d Document) V1() (V1Summary, bool) {
	if d.v1 == nil {
		return V1Summary{}, false
	}
	return *d.v1, true
}

// V2 returns the v2 wire summary and true when this document is v2.
func (d Document) V2() (V2Summary, bool) {
	if d.v2 == nil {
		return V2Summary{}, false
	}
	return *d.v2, true
}

// V3 returns the v3 wire summary and true when this document is v3.
func (d Document) V3() (V3Summary, bool) {
	if d.v3 == nil {
		return V3Summary{}, false
	}
	return *d.v3, true
}

// newDocumentV1 constructs a sealed v1 Document.
func newDocumentV1(s V1Summary) Document {
	return Document{v1: &s}
}

// newDocumentV2 constructs a sealed v2 Document.
func newDocumentV2(s V2Summary) Document {
	return Document{v2: &s}
}

// newDocumentV3 constructs a sealed v3 Document.
func newDocumentV3(s V3Summary) Document {
	return Document{v3: &s}
}
