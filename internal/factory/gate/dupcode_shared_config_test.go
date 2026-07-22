// Package gate provides tests for the dupcode shared analysis context.
package gate

import (
	"slices"
	"testing"

	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// TestDupcodeInputEqual tests the DupcodeInput.equal method with complete config.
func TestDupcodeInputEqual(t *testing.T) {
	tests := []struct {
		name   string
		a      DupcodeInput
		b      DupcodeInput
		expect bool
	}{
		{
			name:   "identical inputs",
			a:      testInput(".", 40, 400, nil, nil, false),
			b:      testInput(".", 40, 400, nil, nil, false),
			expect: true,
		},
		{
			name:   "different root",
			a:      testInput(".", 40, 400, nil, nil, false),
			b:      testInput("/other", 40, 400, nil, nil, false),
			expect: false,
		},
		{
			name:   "different minLines",
			a:      testInput(".", 40, 400, nil, nil, false),
			b:      testInput(".", 50, 400, nil, nil, false),
			expect: false,
		},
		{
			name:   "different minTokens",
			a:      testInput(".", 40, 400, nil, nil, false),
			b:      testInput(".", 40, 500, nil, nil, false),
			expect: false,
		},
		{
			name:   "different ExcludeDirs",
			a:      testInput(".", 40, 400, []string{"a", "b"}, nil, false),
			b:      testInput(".", 40, 400, []string{"a", "c"}, nil, false),
			expect: false,
		},
		{
			name:   "different ExcludeFileSuffixes",
			a:      testInput(".", 40, 400, nil, []string{".pb.go", ".pb2.go"}, false),
			b:      testInput(".", 40, 400, nil, []string{".gen.go"}, false),
			expect: false,
		},
		{
			name:   "different IgnoreGenerated",
			a:      testInput(".", 40, 400, nil, nil, false),
			b:      testInput(".", 40, 400, nil, nil, true),
			expect: false,
		},
		{
			name:   "nil ExcludeDirs equals explicit default (both canonicalize to defaults)",
			a:      testInput(".", 40, 400, nil, nil, false),
			b:      testInput(".", 40, 400, dupcode.DefaultConfig().ExcludeDirs, nil, false),
			expect: true,
		},
		{
			name:   "nil ExcludeDirs differs from empty ExcludeDirs",
			a:      testInput(".", 40, 400, nil, nil, false),
			b:      testInput(".", 40, 400, []string{}, nil, false),
			expect: false, // nil → defaults, empty → exclude nothing
		},
		{
			name:   "nil ExcludeFileSuffixes differs from empty ExcludeFileSuffixes",
			a:      testInput(".", 40, 400, nil, nil, false),
			b:      testInput(".", 40, 400, nil, []string{}, false),
			expect: false, // nil → defaults, empty → exclude nothing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.equal(tt.b)
			if got != tt.expect {
				t.Errorf("DupcodeInput.equal() = %v, want %v", got, tt.expect)
			}
		})
	}
}

// TestCloneConfig tests that cloneConfig creates independent copies.
func TestCloneConfig(t *testing.T) {
	original := dupcode.Config{
		Root:                ".",
		MinLines:            40,
		MinTokens:           400,
		ExcludeDirs:         []string{"a", "b"},
		ExcludeFileSuffixes: []string{".gen.go"},
		IgnoreGenerated:     true,
	}

	cloned := cloneConfig(original)

	// Verify values are copied
	if cloned.Root != original.Root {
		t.Errorf("cloneConfig: Root not copied")
	}
	if cloned.MinLines != original.MinLines {
		t.Errorf("cloneConfig: MinLines not copied")
	}
	if cloned.MinTokens != original.MinTokens {
		t.Errorf("cloneConfig: MinTokens not copied")
	}
	if cloned.IgnoreGenerated != original.IgnoreGenerated {
		t.Errorf("cloneConfig: IgnoreGenerated not copied")
	}

	// Verify slices are independent copies
	if !slices.Equal(cloned.ExcludeDirs, original.ExcludeDirs) {
		t.Errorf("cloneConfig: ExcludeDirs not copied correctly")
	}
	if !slices.Equal(cloned.ExcludeFileSuffixes, original.ExcludeFileSuffixes) {
		t.Errorf("cloneConfig: ExcludeFileSuffixes not copied correctly")
	}

	// Mutate original and verify clone is unaffected
	original.ExcludeDirs[0] = "modified"
	original.ExcludeFileSuffixes[0] = "modified"

	if slices.Equal(cloned.ExcludeDirs, original.ExcludeDirs) {
		t.Error("cloneConfig: ExcludeDirs is not independent (mutation leaked)")
	}
	if slices.Equal(cloned.ExcludeFileSuffixes, original.ExcludeFileSuffixes) {
		t.Error("cloneConfig: ExcludeFileSuffixes is not independent (mutation leaked)")
	}
}
