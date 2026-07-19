package gatesummary

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
)

// duplicateHit describes one observed duplicate object member name.
type duplicateHit struct {
	path string
	key  string
}

// duplicateFrame is one heap-backed scanner frame. The explicit frame slice
// prevents untrusted JSON nesting from consuming the goroutine call stack.
type duplicateFrame struct {
	kind          json.Delim
	path          *duplicatePath
	keys          map[string]struct{}
	arrayIndex    int
	pendingKey    string
	hasPendingKey bool
}

// duplicatePath links raw JSON Pointer tokens without copying the complete
// ancestor path at every nesting level. Paths are rendered only for hits.
type duplicatePath struct {
	parent *duplicatePath
	token  string
}

// detectDuplicateKeys walks the complete token stream iteratively and reports
// every duplicate object member name at its exact JSON Pointer path.
func detectDuplicateKeys(data []byte) []duplicateHit {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	first, err := dec.Token()
	if err != nil {
		return nil
	}
	opening, ok := first.(json.Delim)
	if !ok || (opening != '{' && opening != '[') {
		return nil
	}

	frames := []duplicateFrame{newDuplicateFrame(opening, nil)}
	var hits []duplicateHit
	for len(frames) > 0 {
		token, tokenErr := dec.Token()
		if tokenErr != nil {
			break
		}
		frame := &frames[len(frames)-1]
		if closedDuplicateFrame(frame, token) {
			frames = frames[:len(frames)-1]
			continue
		}

		switch frame.kind {
		case '{':
			if !frame.hasPendingKey {
				key, isKey := token.(string)
				if !isKey {
					return sortedDuplicateHits(hits)
				}
				if _, exists := frame.keys[key]; exists {
					hits = append(hits, duplicateHit{
						path: renderDuplicatePath(frame.path, key),
						key:  key,
					})
				}
				frame.keys[key] = struct{}{}
				frame.pendingKey = key
				frame.hasPendingKey = true
				continue
			}

			key := frame.pendingKey
			frame.pendingKey = ""
			frame.hasPendingKey = false
			pushDuplicateFrame(&frames, token, frame.path, key)
		case '[':
			index := frame.arrayIndex
			frame.arrayIndex++
			pushDuplicateFrame(&frames, token, frame.path, itoa(index))
		}
	}
	return sortedDuplicateHits(hits)
}

func newDuplicateFrame(kind json.Delim, path *duplicatePath) duplicateFrame {
	frame := duplicateFrame{kind: kind, path: path}
	if kind == '{' {
		frame.keys = make(map[string]struct{})
	}
	return frame
}

func closedDuplicateFrame(frame *duplicateFrame, token json.Token) bool {
	delim, ok := token.(json.Delim)
	if !ok || frame.hasPendingKey {
		return false
	}
	return (frame.kind == '{' && delim == '}') || (frame.kind == '[' && delim == ']')
}

func pushDuplicateFrame(
	frames *[]duplicateFrame,
	token json.Token,
	parent *duplicatePath,
	pathToken string,
) {
	delim, ok := token.(json.Delim)
	if !ok || (delim != '{' && delim != '[') {
		return
	}
	path := &duplicatePath{parent: parent, token: pathToken}
	*frames = append(*frames, newDuplicateFrame(delim, path))
}

func renderDuplicatePath(parent *duplicatePath, leaf string) string {
	tokens := []string{leaf}
	for node := parent; node != nil; node = node.parent {
		tokens = append(tokens, node.token)
	}

	var path strings.Builder
	for i := len(tokens) - 1; i >= 0; i-- {
		path.WriteByte('/')
		path.WriteString(escapePointer(tokens[i]))
	}
	return path.String()
}

func sortedDuplicateHits(hits []duplicateHit) []duplicateHit {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].path != hits[j].path {
			return hits[i].path < hits[j].path
		}
		return hits[i].key < hits[j].key
	})
	return hits
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
