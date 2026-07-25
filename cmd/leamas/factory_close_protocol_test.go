package main

import (
	"testing"
)

func TestParseProtocolFlag(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantProtocol  string
		wantRemaining []string
		wantErr       bool
	}{
		{
			name:          "default v1",
			args:          []string{"--plan", "foo.json"},
			wantProtocol:  "v1",
			wantRemaining: []string{"--plan", "foo.json"},
			wantErr:       false,
		},
		{
			name:          "explicit v1",
			args:          []string{"--protocol", "v1", "--plan", "foo.json"},
			wantProtocol:  "v1",
			wantRemaining: []string{"--plan", "foo.json"},
			wantErr:       false,
		},
		{
			name:          "--protocol v2 separated",
			args:          []string{"--protocol", "v2", "--plan", "foo.json"},
			wantProtocol:  "v2",
			wantRemaining: []string{"--plan", "foo.json"},
			wantErr:       false,
		},
		{
			name:          "--protocol=v2 equals",
			args:          []string{"--protocol=v2", "--plan", "foo.json"},
			wantProtocol:  "v2",
			wantRemaining: []string{"--plan", "foo.json"},
			wantErr:       false,
		},
		{
			name:          "duplicate separated flags",
			args:          []string{"--protocol", "v2", "--protocol", "v1"},
			wantProtocol:  "",
			wantRemaining: nil,
			wantErr:       true,
		},
		{
			name:          "duplicate equals flags",
			args:          []string{"--protocol=v2", "--protocol=v1"},
			wantProtocol:  "",
			wantRemaining: nil,
			wantErr:       true,
		},
		{
			name:          "mixed duplicate forms",
			args:          []string{"--protocol", "v2", "--protocol=v1"},
			wantProtocol:  "",
			wantRemaining: nil,
			wantErr:       true,
		},
		{
			name:          "missing value",
			args:          []string{"--protocol"},
			wantProtocol:  "",
			wantRemaining: nil,
			wantErr:       true,
		},
		{
			name:          "empty equals value",
			args:          []string{"--protocol=", "--plan", "x"},
			wantProtocol:  "",
			wantRemaining: nil,
			wantErr:       true,
		},
		{
			name:          "unsupported value",
			args:          []string{"--protocol", "v3"},
			wantProtocol:  "",
			wantRemaining: nil,
			wantErr:       true,
		},
		{
			name:          "preserves unrelated argument order",
			args:          []string{"--plan", "a.json", "--output", "/tmp/out"},
			wantProtocol:  "v1",
			wantRemaining: []string{"--plan", "a.json", "--output", "/tmp/out"},
			wantErr:       false,
		},
		{
			name:          "v2 in middle",
			args:          []string{"--plan", "a.json", "--protocol", "v2", "--json"},
			wantProtocol:  "v2",
			wantRemaining: []string{"--plan", "a.json", "--json"},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, remaining, err := parseProtocolFlag(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if protocol != tt.wantProtocol {
				t.Errorf("protocol = %q, want %q", protocol, tt.wantProtocol)
			}
			if !equalStringSlices(remaining, tt.wantRemaining) {
				t.Errorf("remaining = %v, want %v", remaining, tt.wantRemaining)
			}
		})
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
