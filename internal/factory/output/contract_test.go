package output

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestResultNewResult(t *testing.T) {
	r := NewResult("coverage")
	if r.Check != "coverage" {
		t.Errorf("expected check 'coverage', got '%s'", r.Check)
	}
	if r.OK {
		t.Error("expected OK to be false initially")
	}
	if len(r.Fields) != 0 {
		t.Errorf("expected empty fields, got %d", len(r.Fields))
	}
	if len(r.Failures) != 0 {
		t.Errorf("expected empty failures, got %d", len(r.Failures))
	}
}

func TestResultSetOK(t *testing.T) {
	r := NewResult("coverage")
	r.SetOK()
	if !r.OK {
		t.Error("expected OK to be true after SetOK")
	}
}

func TestResultAddField(t *testing.T) {
	r := NewResult("coverage")
	r.AddField("total", 63.3)
	r.AddField("min", 0.0)

	if len(r.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(r.Fields))
	}
}

func TestResultAddFailure(t *testing.T) {
	r := NewResult("coverage")
	r.AddFailure("threshold", "total below minimum")

	if len(r.Failures) != 1 {
		t.Errorf("expected 1 failure, got %d", len(r.Failures))
	}
	if r.OK {
		t.Error("expected OK to be false after AddFailure")
	}
}

func TestResultBoundedFailures(t *testing.T) {
	r := NewResult("coverage")
	for i := 0; i < 10; i++ {
		r.AddFailure("test", "test message")
	}

	if len(r.Failures) > 5 {
		t.Errorf("expected max 5 failures, got %d", len(r.Failures))
	}
}

func TestResultExitCode(t *testing.T) {
	// OK result
	r := NewResult("coverage")
	r.SetOK()
	if r.ExitCode() != 0 {
		t.Errorf("expected exit code 0 for OK, got %d", r.ExitCode())
	}

	// Fail result
	r2 := NewResult("coverage")
	r2.AddFailure("test", "test")
	if r2.ExitCode() != 1 {
		t.Errorf("expected exit code 1 for fail, got %d", r2.ExitCode())
	}
}

func TestRenderLineOK(t *testing.T) {
	r := NewResult("coverage")
	r.SetOK()
	r.AddField("total", 63.3)
	r.AddField("min", 0.0)

	output := RenderLine(*r)
	expected := "coverage: min=0 total=63.3 OK"

	if output != expected {
		t.Errorf("expected '%s', got '%s'", expected, output)
	}
}

func TestRenderLineFail(t *testing.T) {
	r := NewResult("coverage")
	r.AddField("total", 50.0)
	r.AddField("min", 63.0)
	r.AddFailure("threshold", "total below minimum")

	output := RenderLine(*r)
	if !strings.Contains(output, "FAIL") {
		t.Error("expected output to contain FAIL")
	}
	if !strings.Contains(output, "threshold") {
		t.Error("expected output to contain failure kind")
	}
}

func TestRenderJSON(t *testing.T) {
	r := NewResult("coverage")
	r.SetOK()
	r.AddField("total", 63.3)
	r.AddField("min", 0.0)

	data, err := RenderJSON(*r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if parsed["ok"] != true {
		t.Error("expected ok=true in JSON")
	}
	if parsed["check"] != "coverage" {
		t.Errorf("expected check='coverage', got '%v'", parsed["check"])
	}
}

func TestRenderJSONDeterminism(t *testing.T) {
	// Add fields in non-alphabetical order
	r1 := NewResult("test")
	r1.SetOK()
	r1.AddField("zebra", 1)
	r1.AddField("alpha", 2)
	r1.AddField("middle", 3)

	r2 := NewResult("test")
	r2.SetOK()
	r2.AddField("alpha", 2)
	r2.AddField("middle", 3)
	r2.AddField("zebra", 1)

	data1, _ := RenderJSON(*r1)
	data2, _ := RenderJSON(*r2)

	if string(data1) != string(data2) {
		t.Error("JSON output should be deterministic regardless of field insertion order")
	}
}

func TestNoANSICodes(t *testing.T) {
	r := NewResult("coverage")
	r.SetOK()
	r.AddField("total", 100.0)

	output := RenderLine(*r)
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	if ansiRegex.MatchString(output) {
		t.Error("output should not contain ANSI escape codes")
	}
}

func TestNoProseInOutput(t *testing.T) {
	r := NewResult("coverage")
	r.SetOK()
	r.AddField("total", 100.0)

	output := RenderLine(*r)

	// Check for common prose patterns
	prosePatterns := []string{
		"checking", "verified", "completed", "finished",
		"Running", "Checking", "Verifying",
	}

	for _, pattern := range prosePatterns {
		if strings.Contains(output, pattern) {
			t.Errorf("output should not contain prose '%s': %s", pattern, output)
		}
	}
}

func TestStableFieldOrder(t *testing.T) {
	r := NewResult("coverage")
	r.AddField("total", 63.3)
	r.AddField("min", 0.0)

	output := RenderLine(*r)

	// Field order should be alphabetical: min, then total
	minIdx := strings.Index(output, "min=")
	totalIdx := strings.Index(output, "total=")

	if minIdx > totalIdx {
		t.Errorf("fields should be sorted alphabetically: %s", output)
	}
}

func TestArtifactInOutput(t *testing.T) {
	r := NewResult("dupcode")
	r.SetArtifact(".factory/dupcode.txt")

	output := RenderLine(*r)
	if !strings.Contains(output, ".factory/dupcode.txt") {
		t.Error("expected artifact path in output")
	}
}

func TestSuccessIsOneLine(t *testing.T) {
	r := NewResult("coverage")
	r.SetOK()
	r.AddField("total", 100.0)

	output := RenderLine(*r)
	lines := strings.Split(output, "\n")

	if len(lines) != 1 {
		t.Errorf("success output should be one line, got %d: %v", len(lines), lines)
	}
}
