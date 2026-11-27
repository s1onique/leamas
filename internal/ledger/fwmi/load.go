package fwmi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// File names of the interchange package.
const (
	ManifestFile           = "source-manifest.json"
	HistoricalFile         = "historical-index.json"
	ReconciliationFileName = "reconciliation.json"
	EpicsDir               = "epics"
)

// Package is a parsed interchange package.
type Package struct {
	// Manifest establishes the examined source.
	Manifest SourceManifest
	// Epics are the reconciled epic records, ordered by epic identity.
	Epics []EpicRecord
	// Historical is the historical ACT index.
	Historical HistoricalIndex
	// Reconciliation is the reconciliation manifest.
	Reconciliation ReconciliationFile
}

// Load reads and validates an interchange package from a directory.
//
// The load is STRICT: an unknown field, a second JSON document, an unknown
// vocabulary value or a duplicate identity is a rejection, not a warning.
func Load(dir string) (*Package, error) {
	manifestData, err := os.ReadFile(filepath.Join(dir, ManifestFile))
	if err != nil {
		return nil, fmt.Errorf("read the source manifest: %w", err)
	}
	var manifest SourceManifest
	if err := decodeStrict(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("decode the source manifest: %w", err)
	}
	if manifest.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported source manifest schema version %d; this adapter understands version %d",
			manifest.SchemaVersion, SchemaVersion)
	}

	epics, err := loadEpics(dir)
	if err != nil {
		return nil, err
	}

	var historical HistoricalIndex
	histData, err := os.ReadFile(filepath.Join(dir, HistoricalFile))
	if err != nil {
		return nil, fmt.Errorf("read the historical index: %w", err)
	}
	if err := decodeStrict(histData, &historical); err != nil {
		return nil, fmt.Errorf("decode the historical index: %w", err)
	}
	if historical.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported historical index schema version %d; this adapter understands version %d",
			historical.SchemaVersion, SchemaVersion)
	}

	var recon ReconciliationFile
	reconData, err := os.ReadFile(filepath.Join(dir, ReconciliationFileName))
	if err != nil {
		return nil, fmt.Errorf("read the reconciliation manifest: %w", err)
	}
	if err := decodeStrict(reconData, &recon); err != nil {
		return nil, fmt.Errorf("decode the reconciliation manifest: %w", err)
	}
	if recon.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported reconciliation schema version %d; this adapter understands version %d",
			recon.SchemaVersion, SchemaVersion)
	}

	pkg := &Package{
		Manifest:       manifest,
		Epics:          epics,
		Historical:     historical,
		Reconciliation: recon,
	}
	if err := pkg.Validate(); err != nil {
		return nil, err
	}
	return pkg, nil
}

// loadEpics reads every epic record of the package.
func loadEpics(dir string) ([]EpicRecord, error) {
	epicDir := filepath.Join(dir, EpicsDir)
	entries, err := os.ReadDir(epicDir)
	if err != nil {
		return nil, fmt.Errorf("read the epic directory: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	out := make([]EpicRecord, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(epicDir, name))
		if err != nil {
			return nil, fmt.Errorf("read the epic record %s: %w", name, err)
		}
		var epic EpicRecord
		if err := decodeStrict(data, &epic); err != nil {
			return nil, fmt.Errorf("decode the epic record %s: %w", name, err)
		}
		if epic.SchemaVersion != SchemaVersion {
			return nil, fmt.Errorf("the epic record %s declares schema version %d; this adapter understands version %d",
				name, epic.SchemaVersion, SchemaVersion)
		}
		out = append(out, epic)
	}
	return out, nil
}

// decodeStrict decodes exactly ONE JSON document.
//
// An unknown field is rejected. A second document, a second scalar or
// trailing garbage is rejected. The EOF check is the second Decode. An
// AMBIGUOUS document — one that names the same object member twice — is
// rejected as well: a duplicated member has no single meaning, and the
// interpretation of the record would depend on the decoder.
func decodeStrict(data []byte, target any) error {
	if err := rejectDuplicateMembers(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	if err := dec.Decode(new(json.RawMessage)); err != io.EOF {
		if err == nil {
			return fmt.Errorf("the document contains more than one JSON value")
		}
		return fmt.Errorf("the document contains trailing data: %w", err)
	}
	return nil
}

// rejectDuplicateMembers rejects a document that names the same member twice
// in ONE object.
//
// The same name in DIFFERENT objects is legitimate: two records may each
// carry a "title" member. Only a repetition inside a single object is
// ambiguous.
func rejectDuplicateMembers(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	return scanJSONValue(dec, tok)
}

// scanJSONValue scans the value that starts with the given token.
func scanJSONValue(dec *json.Decoder, tok json.Token) error {
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("the document contains a malformed object member")
			}
			if seen[key] {
				return fmt.Errorf("the document contains the duplicate object member %q", key)
			}
			seen[key] = true
			valueToken, err := dec.Token()
			if err != nil {
				return err
			}
			if err := scanJSONValue(dec, valueToken); err != nil {
				return err
			}
		}
	case '[':
		for dec.More() {
			valueToken, err := dec.Token()
			if err != nil {
				return err
			}
			if err := scanJSONValue(dec, valueToken); err != nil {
				return err
			}
		}
	}
	// Consume the closing delimiter of the object or the array.
	_, err := dec.Token()
	return err
}
