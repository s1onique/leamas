// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure"
)

func TestParseV2AuthorityFlagsRejectsProtocolOne(t *testing.T) {
	_, err := parseV2AuthorityFlags([]string{
		"--protocol-version", "1",
		"--plan-contract-version", "1",
		"--repository", "/tmp/repo",
		"--subject", "a",
		"--freeze", "b",
		"--plan-path", "plan.json",
		"--evidence-directory", "/tmp/evidence",
		"--manifest-output", "/tmp/manifest.json",
	})
	if err == nil {
		t.Fatalf("protocol 1 must be rejected")
	}
	if !strings.Contains(err.Error(), "protocol-version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseV2AuthorityFlagsRejectsMissingRequired(t *testing.T) {
	_, err := parseV2AuthorityFlags([]string{"--repository", "/tmp/repo"})
	if err == nil {
		t.Fatalf("missing required must reject")
	}
	if !strings.Contains(err.Error(), "--subject") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseV2AuthorityFlagsRejectsUnknownFlag(t *testing.T) {
	_, err := parseV2AuthorityFlags([]string{"--unknown"})
	if err == nil {
		t.Fatalf("unknown flag must reject")
	}
}

func TestRunV2AuthorityUsageErrorHasStableExitCode(t *testing.T) {
	bin := buildLeamasForTest(t)
	stdout, stderr, exit := runSubprocessV2(t, bin, []string{
		"factory", "close", "run-v2-authority", "--unknown",
	})
	if exit == 0 {
		t.Fatalf("usage error must return non-zero")
	}
	if !strings.Contains(stderr, "flag provided but not defined") {
		t.Fatalf("stderr=%q stdout=%q", stderr, stdout)
	}
}

func TestRunV2AuthorityJSONFailureEnvelopeIsStable(t *testing.T) {
	bin := buildLeamasForTest(t)
	detached := t.TempDir()
	manifestOutput := filepath.Join(detached, "manifest.json")
	evidenceDirectory := filepath.Join(detached, "evidence")
	stdout, stderr, _ := runSubprocessV2(t, bin, []string{
		"factory", "close", "run-v2-authority", "--json",
		"--repository", detached,
		"--subject", "a",
		"--freeze", "b",
		"--plan-path", "plan.json",
		"--evidence-directory", evidenceDirectory,
		"--manifest-output", manifestOutput,
	})
	raw := strings.TrimSpace(stdout)
	if raw == "" {
		t.Fatalf("expected JSON envelope on stdout, stderr=%q", stderr)
	}
	var env struct {
		OK          bool                  `json:"ok"`
		Code        string                `json:"code"`
		Message     string                `json:"message"`
		Property    string                `json:"property_name"`
		Diagnostics closure.V2Diagnostics `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("decode envelope: %v\nraw=%q", err, raw)
	}
	if env.OK {
		t.Fatalf("envelope OK must be false on failure")
	}
	if env.Code == "" {
		t.Fatalf("envelope code is empty")
	}
	if env.Code != "binary_identity_invalid" && len(env.Diagnostics) == 0 {
		t.Fatalf("non-identity failure must carry diagnostics: %+v", env)
	}
}

func TestRunV2AuthorityIdentityValidatesResolution(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "leamas-cli-identity")
	if err := os.WriteFile(bin, []byte("binary\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := closure.ValidateV2BinaryIdentity(closure.V2BinaryIdentity{
		Path:          bin,
		SHA256:        "00",
		VCSRevision:   strings.Repeat("a", 40),
		VCSModified:   false,
		LeamasVersion: "0.1.0+test",
	}); err == nil {
		t.Fatalf("malformed SHA-256 must reject")
	}
}

// runSubprocessV2 runs the supplied leamas binary via os/exec
// from a temp directory. The LEAMAS_ environment is stripped so
// the production reentry guard sees a clean process. We avoid
// the bounded executor because invoking it from the same
// process would chain the reentry root, which the test must
// not allow.
func runSubprocessV2(t *testing.T, bin string, args []string) (string, string, int) {
	t.Helper()
	cwd := t.TempDir()
	cmd := exec.Command(bin, args...)
	cmd.Dir = cwd
	cmd.Env = envWithoutLeamas()
	var out, errBuf strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	exit := 0
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			exit = -1
		}
	}
	return out.String(), errBuf.String(), exit
}

func envWithoutLeamas() []string {
	env := os.Environ()
	out := env[:0]
	for _, kv := range env {
		if strings.HasPrefix(kv, "LEAMAS_") {
			continue
		}
		out = append(out, kv)
	}
	return out
}
