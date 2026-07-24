// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSidecarEncodingIsDeterministic(t *testing.T) {
	records := []SidecarRecord{
		{LogicalName: "b.out", MediaType: "text/plain; charset=utf-8", SHA256: strings.Repeat("b", 64), ByteCount: 7},
		{LogicalName: "a.out", MediaType: "text/plain; charset=utf-8", SHA256: strings.Repeat("a", 64), ByteCount: 9},
	}
	file, err := BuildSidecarFile("ACT-LEAMAS-SIDECAR01", strings.Repeat("2", 64), SidecarKindChecks, records)
	if err != nil {
		t.Fatal(err)
	}
	data1, hash1, _, err := EncodeSidecarFile(file)
	if err != nil {
		t.Fatal(err)
	}
	records2 := []SidecarRecord{records[1], records[0]}
	file2, _ := BuildSidecarFile("ACT-LEAMAS-SIDECAR01", strings.Repeat("2", 64), SidecarKindChecks, records2)
	data2, hash2, _, _ := EncodeSidecarFile(file2)
	if string(data1) != string(data2) || hash1 != hash2 {
		t.Fatalf("sidecar encoding is not deterministic\n%s\nvs\n%s", data1, data2)
	}
}

func TestSidecarValidatesLimits(t *testing.T) {
	long := strings.Repeat("a", SidecarMaxStringLength+1)
	if _, err := BuildSidecarFile("ACT-LEAMAS-SIDECAR02", strings.Repeat("2", 64), SidecarKindChecks, []SidecarRecord{{LogicalName: long, MediaType: "text/plain", SHA256: strings.Repeat("a", 64), ByteCount: 1}}); err == nil {
		t.Fatal("oversized logical name accepted")
	}
	if _, err := BuildSidecarFile("ACT-LEAMAS-SIDECAR02", strings.Repeat("2", 64), SidecarKindChecks, []SidecarRecord{{LogicalName: "x", MediaType: "text/plain", SHA256: "bad", ByteCount: 1}}); err == nil {
		t.Fatal("bad sha256 accepted")
	}
}

func TestSidecarByteLimitEnforced(t *testing.T) {
	records := []SidecarRecord{{LogicalName: "a.out", MediaType: "text/plain", SHA256: strings.Repeat("a", 64), ByteCount: int64(SidecarPerFileMaxBytes + 1)}}
	if _, err := BuildSidecarFile("ACT-LEAMAS-SIDECAR03", strings.Repeat("2", 64), SidecarKindChecks, records); err == nil {
		t.Fatal("oversized record accepted")
	}
}

func TestSidecarDepthLimitEnforced(t *testing.T) {
	records := make([]SidecarRecord, 0, SidecarMaxRecordCount+1)
	for i := 0; i <= SidecarMaxRecordCount; i++ {
		records = append(records, SidecarRecord{LogicalName: "r", MediaType: "text/plain", SHA256: strings.Repeat("a", 64), ByteCount: 1})
	}
	if _, err := BuildSidecarFile("ACT-LEAMAS-SIDECAR04", strings.Repeat("2", 64), SidecarKindChecks, records); err == nil {
		t.Fatal("record count cap not enforced")
	}
}

func TestSidecarRoundTripBindsIdentity(t *testing.T) {
	dir := t.TempDir()
	records := []SidecarRecord{{LogicalName: "checks.test.stdout", MediaType: "text/plain", SHA256: strings.Repeat("c", 64), ByteCount: 64}}
	summary, err := WriteSidecarFile(dir, "ACT-LEAMAS-SIDECAR05", strings.Repeat("2", 64), SidecarKindChecks, records)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, sidecarFileName("ACT-LEAMAS-SIDECAR05", SidecarKindChecks))); err != nil {
		t.Fatalf("sidecar file missing: %v", err)
	}
	if err := verifySidecarFile(dir, "ACT-LEAMAS-SIDECAR05", summary, SidecarFile{SchemaVersion: 1, ActID: "ACT-LEAMAS-SIDECAR05", Kind: string(SidecarKindChecks), SubjectTree: strings.Repeat("2", 64), Records: records}); err != nil {
		t.Fatalf("verify sidecar: %v", err)
	}
}
