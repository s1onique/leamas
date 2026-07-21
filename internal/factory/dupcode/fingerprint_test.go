// Package dupcode provides tests for duplicate code detection.
package dupcode

import (
	"testing"

	"go/token"
)

// TestNormalizeFingerprintPreservesTokenStringEncoding verifies that the
// normalized fingerprint uses the correct token String representation.
// Operators like ADD produce "+", not "ADD". Keywords like BREAK produce
// "break", not "BREAK". Only IDENT, STRING/CHAR, and numeric types are
// normalized to semantic categories.
func TestNormalizeFingerprintPreservesTokenStringEncoding(t *testing.T) {
	tests := []struct {
		name   string
		tokens []token.Token
		want   string
	}{
		{
			name:   "IDENT operators IDENT uses normalized categories",
			tokens: []token.Token{token.IDENT, token.ADD, token.IDENT},
			want:   "IDENT + IDENT",
		},
		{
			name:   "ASSIGN uses equals sign",
			tokens: []token.Token{token.ASSIGN},
			want:   "=",
		},
		{
			name:   "BREAK uses keyword form",
			tokens: []token.Token{token.BREAK},
			want:   "break",
		},
		{
			name:   "CHAN uses keyword form",
			tokens: []token.Token{token.CHAN},
			want:   "chan",
		},
		{
			name:   "ADD_ASSIGN uses += notation",
			tokens: []token.Token{token.ADD_ASSIGN},
			want:   "+=",
		},
		{
			name:   "CHAR is normalized to STRING",
			tokens: []token.Token{token.CHAR},
			want:   "STRING",
		},
		{
			name:   "numeric types normalized to NUMBER",
			tokens: []token.Token{token.INT, token.FLOAT, token.IMAG},
			want:   "NUMBER NUMBER NUMBER",
		},
		{
			name:   "STRING is normalized to STRING",
			tokens: []token.Token{token.STRING},
			want:   "STRING",
		},
		{
			name:   "IDENT is normalized to IDENT",
			tokens: []token.Token{token.IDENT},
			want:   "IDENT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeFingerprint(tt.tokens)
			if got != tt.want {
				t.Fatalf("normalizeFingerprint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeFingerprint(t *testing.T) {
	tokens := []token.Token{token.IDENT, token.STRING, token.IDENT}
	fp := normalizeFingerprint(tokens)
	if fp == "" {
		t.Error("expected non-empty fingerprint")
	}
}
