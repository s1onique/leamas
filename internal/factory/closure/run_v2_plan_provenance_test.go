// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"strings"
	"testing"
)

func TestParsePlanTreeRecordStrict(t *testing.T) {
	path := "docs/closure-plans/ACT-TEST.json"
	sha1 := strings.Repeat("a", 40)
	tests := []struct {
		name    string
		output  string
		format  ObjectFormat
		wantOID string
		present bool
		wantErr string
	}{
		{name: "absent", format: ObjectFormatSHA1},
		{name: "valid blob", output: "blob\t" + sha1 + "\t" + path + "\n", format: ObjectFormatSHA1, wantOID: sha1, present: true},
		{name: "multiple records", output: "blob\t" + sha1 + "\t" + path + "\nblob\t" + sha1 + "\t" + path + "\n", format: ObjectFormatSHA1, wantErr: "exactly one"},
		{name: "malformed record", output: "blob " + sha1 + " " + path + "\n", format: ObjectFormatSHA1, wantErr: "malformed"},
		{name: "tree object", output: "tree\t" + sha1 + "\t" + path + "\n", format: ObjectFormatSHA1, wantErr: "object type"},
		{name: "wrong path", output: "blob\t" + sha1 + "\tdocs/closure-plans/OTHER.json\n", format: ObjectFormatSHA1, wantErr: "path"},
		{name: "invalid blob OID", output: "blob\tnot-an-oid\t" + path + "\n", format: ObjectFormatSHA1, wantErr: "blob(P:plan)"},
		{name: "blank record", output: "\n", format: ObjectFormatSHA1, wantErr: "malformed"},
		{name: "SHA-256 blob", output: "blob\t" + strings.Repeat("b", 64) + "\t" + path + "\n", format: ObjectFormatSHA256, wantOID: strings.Repeat("b", 64), present: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, present, err := parsePlanTreeRecord([]byte(tt.output), path, tt.format)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if oid != tt.wantOID || present != tt.present {
				t.Fatalf("got oid=%q present=%v, want oid=%q present=%v", oid, present, tt.wantOID, tt.present)
			}
		})
	}
}
