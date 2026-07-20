package gatesummary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkNormalizationV1Minimal benchmarks v1 minimal normalization.
func BenchmarkNormalizationV1Minimal(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "valid", "v1-minimal.json"))
	if err != nil {
		b.Fatalf("failed to read fixture: %v", err)
	}

	decodeResult := Decode(strings.NewReader(string(data)))
	if !decodeResult.Success() {
		b.Fatalf("decode failed: %v", decodeResult.Diagnostics)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		norm := Normalize(decodeResult.Document)
		if !norm.Success() {
			b.Fatalf("normalization failed: %v", norm.Diagnostics)
		}
	}
}

// BenchmarkNormalizationV2Minimal benchmarks v2 minimal normalization.
func BenchmarkNormalizationV2Minimal(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "valid", "v2-minimal.json"))
	if err != nil {
		b.Fatalf("failed to read fixture: %v", err)
	}

	decodeResult := Decode(strings.NewReader(string(data)))
	if !decodeResult.Success() {
		b.Fatalf("decode failed: %v", decodeResult.Diagnostics)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		norm := Normalize(decodeResult.Document)
		if !norm.Success() {
			b.Fatalf("normalization failed: %v", norm.Diagnostics)
		}
	}
}

// BenchmarkNormalizationV2Full benchmarks v2 full normalization.
func BenchmarkNormalizationV2Full(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "valid", "v2-full.json"))
	if err != nil {
		b.Fatalf("failed to read fixture: %v", err)
	}

	decodeResult := Decode(strings.NewReader(string(data)))
	if !decodeResult.Success() {
		b.Fatalf("decode failed: %v", decodeResult.Diagnostics)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		norm := Normalize(decodeResult.Document)
		if !norm.Success() {
			b.Fatalf("normalization failed: %v", norm.Diagnostics)
		}
	}
}

// BenchmarkNormalizationClineMM benchmarks ClineMM µC-3 normalization.
func BenchmarkNormalizationClineMM(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "valid", "v2-clinemm-microc3.json"))
	if err != nil {
		b.Fatalf("failed to read fixture: %v", err)
	}

	decodeResult := Decode(strings.NewReader(string(data)))
	if !decodeResult.Success() {
		b.Fatalf("decode failed: %v", decodeResult.Diagnostics)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		norm := Normalize(decodeResult.Document)
		if !norm.Success() {
			b.Fatalf("normalization failed: %v", norm.Diagnostics)
		}
	}
}
