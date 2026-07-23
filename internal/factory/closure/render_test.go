package closure

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

func TestClosureRenderExactGolden(t *testing.T) {
	report, err := Render(passingManifest(), canonicalPlan())
	if err != nil {
		t.Fatal(err)
	}
	for _, exact := range []string{
		"# ACT-LEAMAS-TEST01 Close Report\n",
		"## Verdict\n\nPASS\n",
		"| focused | PASS | 1000ms | 0 |",
		"- `dupcode` — No dupcode-owned source changed.",
		"Verification state: VERIFIED",
	} {
		if !bytes.Contains(report, []byte(exact)) {
			t.Fatalf("report missing %q:\n%s", exact, report)
		}
	}
}

func TestClosureRenderDeterministic(t *testing.T) {
	first, err := Render(passingManifest(), canonicalPlan())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render(passingManifest(), canonicalPlan())
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("render mismatch: %v", err)
	}
}

func TestClosureRenderConcurrentDeterministic(t *testing.T) {
	want, err := Render(passingManifest(), canonicalPlan())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errors := make(chan string, 20)
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, renderErr := Render(passingManifest(), canonicalPlan())
			if renderErr != nil || !bytes.Equal(got, want) {
				errors <- "non-deterministic render"
			}
		}()
	}
	wg.Wait()
	close(errors)
	for message := range errors {
		t.Fatal(message)
	}
}

func TestClosureRenderRejectsInvalidManifest(t *testing.T) {
	manifest := passingManifest()
	manifest.Verdict = VerdictFail
	if _, err := Render(manifest, canonicalPlan()); err == nil {
		t.Fatal("Render() accepted mismatched verdict")
	}
}

func TestClosureRenderContainsNoPlaceholders(t *testing.T) {
	report := mustRender(t)
	for placeholder := range exactClosurePlaceholders {
		if strings.Contains(string(report), placeholder) {
			t.Fatalf("report contains placeholder %q", placeholder)
		}
	}
}

func TestClosureRenderContainsNoRawLogs(t *testing.T) {
	report := mustRender(t)
	if bytes.Contains(report, []byte("secret raw output")) || bytes.Contains(report, []byte("stdout:")) {
		t.Fatalf("report contains raw output:\n%s", report)
	}
}

func TestClosureRenderContainsNoTagObjectOID(t *testing.T) {
	report := mustRender(t)
	if bytes.Contains(report, []byte("tag_object_oid")) {
		t.Fatalf("report contains tag object identity field:\n%s", report)
	}
}

func TestClosureRenderEndsWithOneLF(t *testing.T) {
	report := mustRender(t)
	if !bytes.HasSuffix(report, []byte("\n")) || bytes.HasSuffix(report, []byte("\n\n")) {
		t.Fatalf("report has wrong ending %q", report[len(report)-4:])
	}
}

func TestClosureRenderWithinLineAndByteBudgets(t *testing.T) {
	report := mustRender(t)
	if len(report) > MaxReportBytes || bytes.Count(report, []byte{'\n'}) > MaxReportLines {
		t.Fatalf("report size=%d lines=%d", len(report), bytes.Count(report, []byte{'\n'}))
	}
}

func mustRender(t *testing.T) []byte {
	t.Helper()
	report, err := Render(passingManifest(), canonicalPlan())
	if err != nil {
		t.Fatal(err)
	}
	return report
}
