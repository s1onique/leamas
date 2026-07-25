// SPDX-License-Identifier: Apache-2.0

package closure

// v2TransactionState represents the current state of a v2 closure transaction.
type v2TransactionState int

const (
	v2StateNew v2TransactionState = iota
	v2StatePrepared
	v2StateRefsCommittedNeedsConvergence
	v2StateVerified
	v2StateInvalid
)

// TransactionResult contains the complete result of a v2 closure transaction.
type TransactionResult struct {
	ActID            string             `json:"act_id"`
	FreezeCommit     string             `json:"F"`
	SubjectCommit    string             `json:"S"`
	ClosureCommit    string             `json:"C"`
	ClosureTree      string             `json:"C_tree"`
	TagName          string             `json:"tag_name"`
	TagObject        string             `json:"tag_object,omitempty"`
	TagTarget        string             `json:"tag_target"`
	EvidencePath     string             `json:"evidence_path"`
	EvidenceHash     string             `json:"evidence_hash"`
	Runner           RunnerIdentity     `json:"runner"`
	Verdict          string             `json:"verdict"`
	TransactionState v2TransactionState `json:"transaction_state,omitempty"`
}

// RunnerIdentityProvider provides runner identity.
type RunnerIdentityProvider interface {
	Identity() (RunnerIdentity, error)
}
