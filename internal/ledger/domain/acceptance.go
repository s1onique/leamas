package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Acceptance authority errors.
var (
	// ErrEvidenceNotAssociated is returned when evidence is not associated
	// with the criterion it is offered to support.
	ErrEvidenceNotAssociated = errors.New("evidence is not associated with the criterion")

	// ErrEvidenceClassInsufficient is returned when the evidence class
	// cannot establish the claimed criterion.
	ErrEvidenceClassInsufficient = errors.New("evidence class cannot establish the criterion")
)

// AcceptanceAuthority is the assessment of whether an acceptance decision
// is backed by admissible evidence.
//
// The assessment separates four concerns that must never substitute for one
// another:
//
//	artifact integrity   the artifact exists with the declared identity
//	evidence class       what kind of claim the artifact may support
//	claim binding        which criterion the artifact supports
//	engineering accept   the scoped, explicit acceptance decision
//
// The first three are established mechanically by this package. The fourth
// is the caller's explicit decision, and it is only permitted when the
// first three hold.
//
// EvaluateAcceptance is the SINGLE acceptance authority. The explicit
// acceptance operation and the canonical validator consult this exact
// predicate; there is no second, competing acceptance rule.
type AcceptanceAuthority struct {
	// Admissible reports whether the acceptance may proceed.
	Admissible bool
	// Reason explains a rejection. It is empty when Admissible.
	Reason string
	// SupportingEvidence lists the evidence identities that support the
	// acceptance.
	SupportingEvidence []EvidenceID
}

// CriterionBinding returns the criterion this work item is bound to.
//
// A work item that declares exactly one acceptance criterion is bound to
// it. A work item that declares several criteria has no single binding:
// the evidence must name the criterion it supports.
func (w WorkItem) CriterionBinding() string {
	if len(w.AcceptanceCriteria) == 1 {
		return w.AcceptanceCriteria[0]
	}
	return ""
}

// RequiredCriteria returns the acceptance criteria the item declares.
//
// The list is the REQUIRED obligation set. Every declared criterion must be
// covered by admissible evidence: an unrelated criterion never satisfies an
// obligation, and a partially covered set never establishes acceptance.
func (w WorkItem) RequiredCriteria() []string {
	out := make([]string, 0, len(w.AcceptanceCriteria))
	for _, c := range w.AcceptanceCriteria {
		trimmed := strings.TrimSpace(c)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

// EvaluateAcceptance is the SINGLE acceptance authority.
//
// It is the only place in the ledger that decides whether a work item may
// be accepted. The explicit acceptance operation (service.Accept) and the
// canonical validator (domain.ValidateLedger) both consult this exact
// predicate: there is no second, competing acceptance rule.
//
// The rules are:
//
//   - the item must have at least one linked evidence record;
//   - every linked evidence record must be REGISTERED in the ledger;
//   - every linked evidence record must be VERIFIED: the artifact identity
//     was actually checked, not merely asserted by a caller;
//   - the evidence must be associated with THIS item;
//   - the evidence must name a valid criterion of THIS item;
//   - EVERY required acceptance criterion of the item must be covered by
//     admissible evidence;
//   - when the item requires a live qualification gate, the supporting
//     evidence must carry the live qualification class.
//
// A caller-supplied quality label never satisfies any of these rules.
//
// The assessment is a pure function of the canonical state. It never
// touches the filesystem: an artifact that changed after its verification
// is detected by the re-verification pass (see RevalidateEvidence), not by
// this function. A cached verification flag therefore never silently
// authorizes acceptance of a changed artifact: the re-verification pass is
// what withdraws the verified status, and this predicate then rejects the
// item.
func EvaluateAcceptance(l Ledger, item WorkItem) AcceptanceAuthority {
	if len(item.Evidence) == 0 {
		return AcceptanceAuthority{
			Reason: "no evidence record is linked to the work item",
		}
	}

	required := item.RequiredCriteria()
	covered := map[string]bool{}

	supporting := make([]EvidenceID, 0, len(item.Evidence))
	for _, ref := range item.Evidence {
		rec, ok := l.EvidenceByID(ref.EvidenceID)
		if !ok {
			return AcceptanceAuthority{
				Reason: fmt.Sprintf("evidence %q is not a registered evidence record",
					string(ref.EvidenceID)),
			}
		}
		if !rec.Verified {
			return AcceptanceAuthority{
				Reason: fmt.Sprintf("evidence %q is not verified", string(ref.EvidenceID)),
			}
		}
		if rec.Quality != EvidenceQualityVerified {
			return AcceptanceAuthority{
				Reason: fmt.Sprintf("evidence %q does not carry verified quality",
					string(ref.EvidenceID)),
			}
		}
		// The evidence must be associated with THIS work item. Evidence
		// that is bound elsewhere, or to no item at all, can never
		// satisfy this item's obligation.
		if rec.ItemID != item.Scoped() {
			return AcceptanceAuthority{
				Reason: fmt.Sprintf("evidence %q is bound to %q, not %q",
					string(ref.EvidenceID), rec.ItemID.String(), item.Scoped().String()),
			}
		}
		// The criterion association must name a criterion that this item
		// actually declares. An unrelated criterion never satisfies an
		// obligation of this item.
		claim := strings.TrimSpace(rec.Criterion)
		if len(required) > 0 {
			if claim == "" {
				return AcceptanceAuthority{
					Reason: fmt.Sprintf("evidence %q is not associated with a criterion",
						string(ref.EvidenceID)),
				}
			}
			if !containsCriterion(required, claim) {
				return AcceptanceAuthority{
					Reason: fmt.Sprintf("evidence %q is associated with criterion %q, which is not an acceptance criterion of %q",
						string(ref.EvidenceID), rec.Criterion, item.Scoped().String()),
				}
			}
			covered[claim] = true
		}
		if item.RequiresLiveGate && !rec.Class.SatisfiesLiveQualification() {
			return AcceptanceAuthority{
				Reason: fmt.Sprintf("evidence %q of class %q cannot establish a live qualification",
					string(ref.EvidenceID), string(rec.Class)),
			}
		}
		supporting = append(supporting, rec.EvidenceID)
	}

	// Every required acceptance criterion must be covered. A partially
	// covered set never establishes acceptance.
	for _, c := range required {
		if !covered[c] {
			return AcceptanceAuthority{
				Reason: fmt.Sprintf("acceptance criterion %q of %q is not covered by admissible evidence",
					c, item.Scoped().String()),
			}
		}
	}

	return AcceptanceAuthority{
		Admissible:         true,
		SupportingEvidence: supporting,
	}
}

// containsCriterion reports whether the trimmed claim names one of the
// required criteria.
func containsCriterion(required []string, claim string) bool {
	for _, c := range required {
		if c == claim {
			return true
		}
	}
	return false
}
