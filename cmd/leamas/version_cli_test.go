package main

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestVersionCLI_Output(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/s1onique/leamas/cmd/leamas", "version")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("leamas version failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), output)
	}

	expected := []string{"version:", "commit:", "build_time:"}
	for i, prefix := range expected {
		if !strings.HasPrefix(lines[i], prefix) {
			t.Errorf("line %d: expected prefix %q, got %q", i, prefix, lines[i])
		}
	}
}

func TestVersionCLI_ExitCode(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/s1onique/leamas/cmd/leamas", "version")
	err := cmd.Run()
	if err != nil {
		t.Errorf("leamas version should exit 0, got error: %v", err)
	}
}

func TestVersionCLI_JSON(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/s1onique/leamas/cmd/leamas", "version", "--json")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("leamas version --json failed: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(output, &data); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, output)
	}

	if data["version"] == "" {
		t.Error("JSON missing 'version' field")
	}
	if data["commit"] == "" {
		t.Error("JSON missing 'commit' field")
	}
	if data["build_time"] == "" {
		t.Error("JSON missing 'build_time' field")
	}
}

func TestVersionCLI_NoExtraOutput(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/s1onique/leamas/cmd/leamas", "version")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("leamas version failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "leamas") && !strings.Contains(line, "version") {
			t.Errorf("unexpected output line: %q", line)
		}
	}
}
