package gatesummary

// Version is the gate-summary schema discriminator.
type Version uint8

const (
	// Version1 is the frozen v1 wire contract.
	Version1 Version = 1
	// Version2 is the frozen v2 wire contract.
	Version2 Version = 2
)

// String returns the canonical "v1"/"v2" textual form.
func (v Version) String() string {
	switch v {
	case Version1:
		return "v1"
	case Version2:
		return "v2"
	default:
		return "v?"
	}
}

// Document is the sealed, version-specific wire document produced by a
// successful Decode call. Exactly one of v1 or v2 is populated.
type Document struct {
	v1 *V1Summary
	v2 *V2Summary
}

// Version returns the schema version of the document.
func (d Document) Version() Version {
	if d.v1 != nil {
		return Version1
	}
	if d.v2 != nil {
		return Version2
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

// newDocumentV1 constructs a sealed v1 Document.
func newDocumentV1(s V1Summary) Document {
	return Document{v1: &s}
}

// newDocumentV2 constructs a sealed v2 Document.
func newDocumentV2(s V2Summary) Document {
	return Document{v2: &s}
}
