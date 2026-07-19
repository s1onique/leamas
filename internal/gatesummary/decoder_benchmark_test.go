package gatesummary

import (
	"bytes"
	"testing"
)

func BenchmarkDecodeV1Minimal(b *testing.B) {
	benchmarkDecodeFixture(b, "testdata/valid/v1-minimal.json")
}

func BenchmarkDecodeV2Minimal(b *testing.B) {
	benchmarkDecodeFixture(b, "testdata/valid/v2-minimal.json")
}

func BenchmarkDecodeV2Full(b *testing.B) {
	benchmarkDecodeFixture(b, "testdata/valid/v2-full.json")
}

func benchmarkDecodeFixture(b *testing.B, path string) {
	b.Helper()
	data := readFixture(b, path)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := Decode(bytes.NewReader(data))
		if !res.Success() {
			b.Fatalf("Decode(%s) failed: diagnostics=%+v err=%v", path, res.Diagnostics, res.Err)
		}
	}
}
