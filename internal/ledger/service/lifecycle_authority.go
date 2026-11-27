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
	return checkLifecycleTransition(it, next)
}

// checkLifecycleTransition is the SINGLE lifecycle-transition authority.
//
// It rejects a terminal transition that would CONCEAL an unfinished
// engineering obligation.
//
// A preserved HISTORICAL disposition is HISTORICAL FACT: it is conserved on
// the record and it stays visible there. It is NEVER an authority for a new
// terminal transition. A record whose qualification was never established
// still carries an outstanding engineering obligation, and a historical
// disposition may never retire it: the disposition describes what the
// historical source decided, not that the obligation is settled.
func checkLifecycleTransition(it domain.WorkItem, next domain.LifecycleState) error {
	if next != domain.LifecycleClosed && next != domain.LifecycleArchived {
		return nil
	}
	if domain.IsTerminalAcceptance(it.Acceptance) {
		return nil
	}
	return fmt.Errorf("%w: %s has acceptance state %q; a lifecycle transition may not hide an unresolved obligation",
		ErrUnfinishedObligation, it.Scoped().String(), string(it.Acceptance))
}
