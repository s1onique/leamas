// Package dupcode defines the bounded, deterministic fuzz wire format.
//
// The format preserves explicit path IDs, region ordinals and ranges,
// variable window ranges, ownership validity, and raw record order. The
// decoder is total: every byte slice maps to at most eight regions and
// normally at most 32 windows. The 0xff extended-count marker permits
// deterministic N32/N128 seeds up to 256 windows without making large
// random inputs common. No malformed input calls t.Skip.
package dupcode

import (
	"encoding/binary"
	"fmt"
)

const (
	v4FuzzMaxRegions    = 8
	v4FuzzMaxWindows    = 256
	v4FuzzCommonWindows = 32
	v4FuzzPositionMod   = 1024
)

var v4FuzzPaths = [...]string{
	"alpha.go",
	"beta.go",
	"gamma.go",
	"shared.go",
	"missing.go",
}

type v4FuzzCursor struct {
	data []byte
	pos  int
}

func (cursor *v4FuzzCursor) nextByte() byte {
	if len(cursor.data) == 0 {
		return 0
	}
	value := cursor.data[cursor.pos%len(cursor.data)]
	cursor.pos++
	return value
}

func (cursor *v4FuzzCursor) nextUint16() uint16 {
	bytes := [2]byte{cursor.nextByte(), cursor.nextByte()}
	return binary.LittleEndian.Uint16(bytes[:])
}

func (cursor *v4FuzzCursor) nextPosition() int {
	return int(cursor.nextUint16() % v4FuzzPositionMod)
}

func (cursor *v4FuzzCursor) nextPath() string {
	return v4FuzzPaths[int(cursor.nextByte())%len(v4FuzzPaths)]
}

func v4DecodeFuzzFixture(wire []byte) v4CorpusFixture {
	cursor := &v4FuzzCursor{data: wire}
	regionCount := int(cursor.nextByte())%v4FuzzMaxRegions + 1
	regions := make([]v4FixtureRegion, 0, regionCount)
	lengths := make(map[string]int)
	for i := 0; i < regionCount; i++ {
		path := cursor.nextPath()
		ordinal := int(cursor.nextByte() % v4FuzzMaxRegions)
		start, end := cursor.nextPosition(), cursor.nextPosition()
		if end < start {
			start, end = end, start
		}
		regions = append(regions, v4FixtureRegion{
			Path: path, Ordinal: ordinal, StartPos: start, EndPos: end,
			StartLine: start + 1, EndLine: end + 1,
		})
		if end+1 > lengths[path] {
			lengths[path] = end + 1
		}
	}

	windowCount := v4DecodeFuzzWindowCount(cursor)
	windows := make([]v4RawWindow, 0, windowCount)
	for i := 0; i < windowCount; i++ {
		path := cursor.nextPath()
		start, end := cursor.nextPosition(), cursor.nextPosition()
		if end < start {
			start, end = end, start
		}
		windows = append(windows, v4RawWindow{
			Path: path, StartPos: start, EndPos: end,
			StartLine: start + 1, EndLine: end + 1,
		})
	}

	left, right := v4SyntaxRegionID{}, v4SyntaxRegionID{}
	if len(regions) > 0 {
		left = v4SyntaxRegionID{Path: regions[0].Path, Ordinal: regions[0].Ordinal}
		right = left
	}
	if len(regions) > 1 {
		right = v4SyntaxRegionID{Path: regions[1].Path, Ordinal: regions[1].Ordinal}
	}
	return v4CorpusFixture{
		Name: "fuzz-wire", Dimension: "FuzzWire",
		Regions: regions, FileLength: lengths, RawWindows: windows,
		LeftRegion: left, RightRegion: right,
	}
}

func v4DecodeFuzzWindowCount(cursor *v4FuzzCursor) int {
	marker := cursor.nextByte()
	if marker != 0xff {
		return int(marker)%v4FuzzCommonWindows + 1
	}
	return int(cursor.nextUint16())%v4FuzzMaxWindows + 1
}

func v4EncodeFuzzFixture(fx v4CorpusFixture) []byte {
	if len(fx.Regions) < 1 || len(fx.Regions) > v4FuzzMaxRegions {
		panic(fmt.Sprintf("fuzz fixture %s region count=%d, want 1..%d",
			fx.Name, len(fx.Regions), v4FuzzMaxRegions))
	}
	if len(fx.RawWindows) < 1 || len(fx.RawWindows) > v4FuzzMaxWindows {
		panic(fmt.Sprintf("fuzz fixture %s window count=%d, want 1..%d",
			fx.Name, len(fx.RawWindows), v4FuzzMaxWindows))
	}
	wire := []byte{byte(len(fx.Regions) - 1)}
	for _, region := range fx.Regions {
		wire = append(wire, v4FuzzPathID(region.Path), byte(region.Ordinal))
		wire = v4AppendFuzzUint16(wire, region.StartPos)
		wire = v4AppendFuzzUint16(wire, region.EndPos)
	}
	if len(fx.RawWindows) <= v4FuzzCommonWindows {
		wire = append(wire, byte(len(fx.RawWindows)-1))
	} else {
		wire = append(wire, 0xff)
		wire = v4AppendFuzzUint16(wire, len(fx.RawWindows)-1)
	}
	for _, window := range fx.RawWindows {
		wire = append(wire, v4FuzzPathID(window.Path))
		wire = v4AppendFuzzUint16(wire, window.StartPos)
		wire = v4AppendFuzzUint16(wire, window.EndPos)
	}
	return wire
}

func v4AppendFuzzUint16(wire []byte, value int) []byte {
	if value < 0 || value >= v4FuzzPositionMod {
		panic(fmt.Sprintf("fuzz wire value=%d outside 0..%d", value, v4FuzzPositionMod-1))
	}
	return binary.LittleEndian.AppendUint16(wire, uint16(value))
}

func v4FuzzPathID(path string) byte {
	for i, candidate := range v4FuzzPaths {
		if candidate == path {
			return byte(i)
		}
	}
	panic("fuzz wire path is not representable: " + path)
}
