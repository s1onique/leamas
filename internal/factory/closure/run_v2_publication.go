// SPDX-License-Identifier: Apache-2.0

package closure

import "fmt"

// PublicationIncompleteError means object/evidence preparation succeeded but
// branch/tag refs and worktree convergence have not completed.
type PublicationIncompleteError struct {
	ClosureCommit string
	TagObject     string
	EvidenceHash  string
}

func (e *PublicationIncompleteError) Error() string {
	return fmt.Sprintf("closure publication incomplete: prepared C=%s T=%s E=%s; refs unchanged",
		e.ClosureCommit, e.TagObject, e.EvidenceHash)
}
