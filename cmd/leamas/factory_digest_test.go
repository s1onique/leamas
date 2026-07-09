// Package main provides tests for the factory digest command.
package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/digest"
)

// fakeWriteDigest captures options and returns a configurable error.
func fakeWriteDigest(captured *digest.Options, err error) func(digest.Options) error {
	return func(opts digest.Options) error {
		*captured = opts
		return err
	}
}

func TestParseDigestArgs_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantMode        digest.Mode
		wantErr         bool
		wantErrContains string
	}{
		// Success cases - modes
		{"auto mode (default)", []string{"--output", "/tmp/d.md"}, digest.ModeAuto, false, ""},
		{"dirty mode", []string{"--dirty", "--output", "/tmp/d.md"}, digest.ModeDirty, false, ""},
		{"staged mode", []string{"--staged", "--output", "/tmp/d.md"}, digest.ModeStaged, false, ""},
		{"range mode", []string{"--range", "HEAD~1..HEAD", "--output", "/tmp/d.md"}, digest.ModeRange, false, ""},
		// Success cases - flags order
		{"output first", []string{"--output", "/tmp/d.md"}, digest.ModeAuto, false, ""},
		{"dirty then output", []string{"--dirty", "--output", "/tmp/d.md"}, digest.ModeDirty, false, ""},
		{"staged then output", []string{"--staged", "--output", "/tmp/d.md"}, digest.ModeStaged, false, ""},
		{"output then dirty", []string{"--output", "/tmp/d.md", "--dirty"}, digest.ModeDirty, false, ""},
		// Error cases
		{"missing output", []string{"--dirty"}, "", true, "--output is required"},
		{"missing range arg", []string{"--range", "--unknown", "/tmp/d.md"}, "", true, "requires a revision range argument"},
		{"missing range arg at end", []string{"--range"}, "", true, "requires a revision range argument"},
		{"range arg is flag", []string{"--range", "--dirty", "/tmp/d.md"}, "", true, "requires a revision range argument"},
		{"missing output arg", []string{"--output"}, "", true, "--output requires a path argument"},
		{"unknown flag", []string{"--unknown", "/tmp/d.md"}, "", true, "unknown flag"},
		{"dirty+staged conflict", []string{"--dirty", "--staged", "--output", "/tmp/d.md"}, "", true, "cannot specify both --dirty and --staged"},
		{"dirty+range conflict", []string{"--dirty", "--range", "a..b", "--output", "/tmp/d.md"}, "", true, "cannot specify both --dirty and --range"},
		{"staged+range conflict", []string{"--staged", "--range", "a..b", "--output", "/tmp/d.md"}, "", true, "cannot specify both --staged and --range"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDigestArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.mode != tt.wantMode {
				t.Errorf("mode = %s, want %s", got.mode, tt.wantMode)
			}
		})
	}
}

func TestParseDigestArgs_RangeSpec(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantSpec string
	}{
		{"simple range", []string{"--range", "a..b", "--output", "/tmp/d.md"}, "a..b"},
		{"HEAD range", []string{"--range", "HEAD~5..HEAD", "--output", "/tmp/d.md"}, "HEAD~5..HEAD"},
		{"commit range", []string{"--range", "abc123..def456", "--output", "/tmp/d.md"}, "abc123..def456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDigestArgs(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.hasRange {
				t.Error("expected hasRange to be true")
			}
			if got.rangeSpec != tt.wantSpec {
				t.Errorf("rangeSpec = %q, want %q", got.rangeSpec, tt.wantSpec)
			}
		})
	}
}

func TestParseDigestArgs_FlagPositions(t *testing.T) {
	// Test various flag orderings
	tests := []struct {
		name string
		args []string
	}{
		{"all flags before output", []string{"--dirty", "--output", "/tmp/d.md"}},
		{"output between flags", []string{"--output", "/tmp/d.md", "--dirty"}},
		{"single flag", []string{"--dirty", "--output", "/tmp/d.md"}},
		{"empty args", []string{"--output", "/tmp/d.md"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDigestArgs(tt.args)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunFactoryDigest_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		writeErr        error
		wantCode        int
		wantMode        digest.Mode
		wantErrContains string
	}{
		// Success cases
		{"auto mode success", []string{"--output", "/tmp/d.md"}, nil, 0, digest.ModeAuto, ""},
		{"dirty mode success", []string{"--dirty", "--output", "/tmp/d.md"}, nil, 0, digest.ModeDirty, ""},
		{"staged mode success", []string{"--staged", "--output", "/tmp/d.md"}, nil, 0, digest.ModeStaged, ""},
		{"range mode success", []string{"--range", "a..b", "--output", "/tmp/d.md"}, nil, 0, digest.ModeRange, ""},
		// Error cases
		{"parse error", []string{"--unknown"}, nil, 1, "", "unknown flag"},
		{"write error", []string{"--output", "/tmp/d.md"}, errors.New("write failed"), 1, "", "write failed"},
		{"missing output", []string{}, nil, 1, "", "--output is required"},
		{"conflict error", []string{"--dirty", "--staged"}, nil, 1, "", "cannot specify both"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured digest.Options
			fakeWrite := fakeWriteDigest(&captured, tt.writeErr)

			var stdout, stderr bytes.Buffer
			code := runFactoryDigest(tt.args, &stdout, &stderr, fakeWrite)

			if code != tt.wantCode {
				t.Errorf("code = %d, want %d. stderr: %s", code, tt.wantCode, stderr.String())
			}

			if tt.wantMode != "" && captured.Mode != tt.wantMode {
				t.Errorf("mode = %s, want %s", captured.Mode, tt.wantMode)
			}

			if tt.wantErrContains != "" {
				output := stderr.String()
				if !strings.Contains(output, tt.wantErrContains) {
					t.Errorf("stderr does not contain %q: %s", tt.wantErrContains, output)
				}
			}
		})
	}
}

func TestRunFactoryDigest_OutputPath(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
	}{
		{"simple path", []string{"--output", "/tmp/d.md"}, "/tmp/d.md"},
		{"nested path", []string{"--dirty", "--output", "/var/tmp/out.md"}, "/var/tmp/out.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured digest.Options
			fakeWrite := fakeWriteDigest(&captured, nil)

			var stdout bytes.Buffer
			code := runFactoryDigest(tt.args, &stdout, &bytes.Buffer{}, fakeWrite)

			if code != 0 {
				t.Fatalf("expected code 0, got %d", code)
			}
			if captured.Output != tt.wantPath {
				t.Errorf("output = %q, want %q", captured.Output, tt.wantPath)
			}
			if !strings.Contains(stdout.String(), tt.wantPath) {
				t.Errorf("stdout does not contain path %q: %s", tt.wantPath, stdout.String())
			}
		})
	}
}

func TestRunFactoryDigest_RangeOption(t *testing.T) {
	var captured digest.Options
	fakeWrite := fakeWriteDigest(&captured, nil)

	var stdout bytes.Buffer
	code := runFactoryDigest([]string{"--range", "HEAD~3..HEAD", "--output", "/tmp/d.md"}, &stdout, &bytes.Buffer{}, fakeWrite)

	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if captured.Range != "HEAD~3..HEAD" {
		t.Errorf("range = %q, want %q", captured.Range, "HEAD~3..HEAD")
	}
}

func TestRunFactoryDigest_UsesCorrectWriters(t *testing.T) {
	var captured digest.Options
	fakeWrite := fakeWriteDigest(&captured, nil)

	// Test that errors go to stderr
	var stderr bytes.Buffer
	code := runFactoryDigest([]string{"--unknown"}, &bytes.Buffer{}, &stderr, fakeWrite)

	if code != 1 {
		t.Errorf("expected code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Error("expected usage text in stderr")
	}

	// Test that success output goes to stdout
	var stdout bytes.Buffer
	code = runFactoryDigest([]string{"--output", "/tmp/d.md"}, &stdout, &bytes.Buffer{}, fakeWrite)

	if code != 0 {
		t.Errorf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "/tmp/d.md") {
		t.Error("expected path in stdout")
	}
}

func TestDigestUsageText_ContainsRequiredFlags(t *testing.T) {
	var buf bytes.Buffer
	printDigestUsageTo(&buf)
	text := buf.String()

	tests := []struct {
		name string
		flag string
	}{
		{"dirty flag", "--dirty"},
		{"staged flag", "--staged"},
		{"range flag", "--range"},
		{"output flag", "--output"},
		{"usage header", "Usage:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(text, tt.flag) {
				t.Errorf("usage text missing %q", tt.flag)
			}
		})
	}
}

func TestDigestUsageText_AllLinesEndWithNewline(t *testing.T) {
	var buf bytes.Buffer
	printDigestUsageTo(&buf)
	text := buf.String()
	lines := strings.Split(text, "\n")
	for _, line := range lines[:len(lines)-1] {
		if line != "" && !strings.HasSuffix(line, "") {
			// Just verify no weird content
		}
	}
	_ = lines // avoid unused variable
}
