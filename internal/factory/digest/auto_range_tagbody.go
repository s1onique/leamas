// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"strings"
)

// parseTagBodyLoose extracts freeze_commit, subject_commit_oid, and
// closure_commit_oid fields from the structured trailer block of an
// annotated tag. Returns empty strings when a field is missing.
func parseTagBodyLoose(body string) (freeze, subject, closureRef string) {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "freeze_commit:"):
			freeze = strings.TrimSpace(strings.TrimPrefix(line, "freeze_commit:"))
		case strings.HasPrefix(line, "subject_commit_oid:"):
			subject = strings.TrimSpace(strings.TrimPrefix(line, "subject_commit_oid:"))
		case strings.HasPrefix(line, "closure_commit_oid:"):
			closureRef = strings.TrimSpace(strings.TrimPrefix(line, "closure_commit_oid:"))
		}
	}
	return freeze, subject, closureRef
}
