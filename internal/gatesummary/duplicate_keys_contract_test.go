package gatesummary

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeDeepestAcceptedNestingWithoutPanic(t *testing.T) {
	cases := []struct {
		name  string
		open  string
		close string
	}{
		{name: "arrays", open: "[", close: "]"},
		{name: "objects", open: `{"x":`, close: "}"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, depth := deepestAcceptedVersionNesting(t, tc.open, tc.close)
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("Decode panicked at accepted depth %d: %v", depth, recovered)
				}
			}()

			trace := decodeTrace{}
			res := decodeWithTrace(bytes.NewReader(data), &trace)
			assertOnlyCode(t, res, CodeInvalidVersionType)
			if trace.Stage != stageVersionProbe || trace.SchemaInvoked || trace.WireDecoded {
				t.Fatalf("deepest accepted nesting crossed version boundary: %+v", trace)
			}
		})
	}
}

func deepestAcceptedVersionNesting(t *testing.T, open, close string) ([]byte, int) {
	t.Helper()
	build := func(depth int) []byte {
		return []byte(`{"schema_version":` + strings.Repeat(open, depth) + "2" +
			strings.Repeat(close, depth) + "}")
	}

	low, high := 0, 1
	for json.Valid(build(high)) {
		low = high
		high *= 2
		if len(build(high)) > MaxDocumentBytes {
			t.Fatal("document bound reached before JSON nesting bound")
		}
	}
	for low+1 < high {
		mid := low + (high-low)/2
		if json.Valid(build(mid)) {
			low = mid
		} else {
			high = mid
		}
	}

	data := build(low)
	if !json.Valid(data) || json.Valid(build(low+1)) {
		t.Fatalf("failed to identify deepest accepted JSON nesting at depth %d", low)
	}
	return data, low
}
