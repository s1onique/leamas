package service

import (
	"fmt"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// checkLifecycleDoesNotHideObligation rejects a generic lifecycle update
// that would move unfinished work into a terminal state.
//
// A work item whose acceptance obligation is still outstanding must not be
// hidden behind the closed or archived lifecycle state through a generic
// update. The obligation stays visible on the board only while the item
// remains live work.
//
// A work item that IS accepted carries no outstanding obligation, so the
// generic update may move it.
func checkLifecycleDoesNotHideObligation(it domain.WorkItem, next domain.LifecycleState) error {
	return checkLifecycleTransition(it, next, false)
}

// checkLifecycleTransition is the SINGLE lifecycle-transition authority.
//
// It rejects a terminal transition that would CONCEAL an unfinished
// engineering obligation. The generic update path never tolerates it.
//
// The checkpoint path tolerates a preserved HISTORICAL disposition: a
// record that already carries a terminal disposition keeps that
// disposition even when its qualification is unknown. The historical
// disposition is not hidden by the transition — it is conserved by it.
//
// The allowance is narrow: it applies to a transition that CONSERVES an
// existing historical disposition, never to a transition that creates a
// terminal state out of an unfinished obligation.
func checkLifecycleTransition(it domain.WorkItem, next domain.LifecycleState, allowHistoricalDisposition bool) error {
	if next != domain.LifecycleClosed && next != domain.LifecycleArchived {
		return nil
	}
	if domain.IsTerminalAcceptance(it.Acceptance) {
		return nil
	}
	if allowHistoricalDisposition && it.TerminalDisposition != "" {
		// The record already carries a conserved terminal disposition.
		// Its qualification may be unknown, but the disposition is
		// historical fact, not a concealment.
		return nil
	}
	return fmt.Errorf("%w: %s has acceptance state %q; a lifecycle transition may not hide an unresolved obligation",
		ErrUnfinishedObligation, it.Scoped().String(), string(it.Acceptance))
}
