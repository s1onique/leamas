package gatesummary

import (
	"bytes"
	"encoding/json"
	"sort"
)

// duplicateHit describes one observed duplicate object member name.
type duplicateHit struct {
	path string
	key  string
}

// detectDuplicateKeys walks the full JSON object structure and reports
// every duplicate object member name. Each duplicate is recorded with
// its JSON Pointer path. Objects inside arrays are tracked under the
// appropriate array index path.
func detectDuplicateKeys(data []byte) []duplicateHit {
	dec := json.NewDecoder(bytes.NewReader(data))
	var hits []duplicateHit
	walkForDuplicates(dec, "", &hits)
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].path != hits[j].path {
			return hits[i].path < hits[j].path
		}
		return hits[i].key < hits[j].key
	})
	return hits
}

// walkForDuplicates recursively scans JSON values, collecting
// duplicate object member names. The path argument is the JSON
// Pointer prefix of the current container.
func walkForDuplicates(dec *json.Decoder, path string, hits *[]duplicateHit) {
	tok, err := dec.Token()
	if err != nil {
		return
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return
	}
	switch d {
	case '{':
		seen := make(map[string]bool)
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return
			}
			key, ok := keyTok.(string)
			if !ok {
				return
			}
			childPath := path + "/" + escapePointer(key)
			if seen[key] {
				*hits = append(*hits, duplicateHit{path: childPath, key: key})
			}
			seen[key] = true
			walkForDuplicates(dec, childPath, hits)
		}
		if _, err := dec.Token(); err != nil {
			return
		}
	case '[':
		i := 0
		for dec.More() {
			childPath := path + "/" + itoa(i)
			walkForDuplicates(dec, childPath, hits)
			i++
		}
		if _, err := dec.Token(); err != nil {
			return
		}
	}
}

// itoa formats a non-negative integer without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}
