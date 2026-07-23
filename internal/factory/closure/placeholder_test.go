package closure

import "testing"

func TestClosurePlaceholderDetection(t *testing.T) {
	cases := map[string]bool{
		"TBD":                                    true,
		"todo":                                   true,
		"UNKNOWN":                                true,
		"to be recorded":                         true,
		"RUNNING":                                true,
		"please (see git rev-parse) for context": true,
		"unknown subject <commit>":               true,
		"normal prose without placeholders":      false,
		"ready for review":                       false,
	}
	for input, want := range cases {
		if got := containsClosurePlaceholder(input); got != want {
			t.Fatalf("containsClosurePlaceholder(%q) = %v, want %v", input, got, want)
		}
	}
}
