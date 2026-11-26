package domain

import (
	"errors"
	"fmt"
	"regexp"
)

// Identity validation errors.
var (
	ErrEmptyIdentity   = errors.New("identity must not be empty")
	ErrInvalidIdentity = errors.New("invalid identity")
)

// identityPattern is the canonical pattern for every scoped identity
// component: an uppercase-led token of uppercase letters, digits and
// dashes, optionally followed by a dotted revision suffix.
var identityPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*(-[A-Z0-9]+)*(\.[A-Za-z0-9][A-Za-z0-9.-]*)?$`)

// itemLocalPattern is the canonical pattern for a work-item local identity.
var itemLocalPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*(-[A-Z0-9]+)*$`)

// ValidateEpicID validates an epic identity.
func ValidateEpicID(id EpicID) error {
	if id == "" {
		return fmt.Errorf("%w: epic id", ErrEmptyIdentity)
	}
	if !identityPattern.MatchString(string(id)) {
		return fmt.Errorf("%w: epic id %q", ErrInvalidIdentity, string(id))
	}
	return nil
}

// ValidateItemID validates a work-item local identity.
func ValidateItemID(id ItemID) error {
	if id == "" {
		return fmt.Errorf("%w: item id", ErrEmptyIdentity)
	}
	if !itemLocalPattern.MatchString(string(id)) {
		return fmt.Errorf("%w: item id %q", ErrInvalidIdentity, string(id))
	}
	return nil
}

// ValidateActID validates an ACT execution record identity.
func ValidateActID(id ActID) error {
	if id == "" {
		return fmt.Errorf("%w: act id", ErrEmptyIdentity)
	}
	if !identityPattern.MatchString(string(id)) {
		return fmt.Errorf("%w: act id %q", ErrInvalidIdentity, string(id))
	}
	return nil
}

// ValidateEvidenceID validates an evidence reference identity.
func ValidateEvidenceID(id EvidenceID) error {
	if id == "" {
		return fmt.Errorf("%w: evidence id", ErrEmptyIdentity)
	}
	if !identityPattern.MatchString(string(id)) {
		return fmt.Errorf("%w: evidence id %q", ErrInvalidIdentity, string(id))
	}
	return nil
}

// ValidateDecisionID validates a decision identity.
func ValidateDecisionID(id DecisionID) error {
	if id == "" {
		return fmt.Errorf("%w: decision id", ErrEmptyIdentity)
	}
	if !identityPattern.MatchString(string(id)) {
		return fmt.Errorf("%w: decision id %q", ErrInvalidIdentity, string(id))
	}
	return nil
}

// ValidateKnowledgeID validates a knowledge reference identity.
func ValidateKnowledgeID(id KnowledgeID) error {
	if id == "" {
		return fmt.Errorf("%w: knowledge id", ErrEmptyIdentity)
	}
	if !identityPattern.MatchString(string(id)) {
		return fmt.Errorf("%w: knowledge id %q", ErrInvalidIdentity, string(id))
	}
	return nil
}

// ValidateScopedItemID validates a scoped work-item identity and
// requires the "<EPIC-ID>/<ITEM-ID>" shape.
func ValidateScopedItemID(id ScopedItemID) error {
	if id == "" {
		return fmt.Errorf("%w: scoped item id", ErrEmptyIdentity)
	}
	scope := id.Scope()
	local := id.Local()
	if scope == "" || local == "" {
		return fmt.Errorf("%w: scoped item id %q must be <EPIC-ID>/<ITEM-ID>", ErrInvalidIdentity, string(id))
	}
	if err := ValidateEpicID(scope); err != nil {
		return fmt.Errorf("scoped item id %q: %w", string(id), err)
	}
	if err := ValidateItemID(local); err != nil {
		return fmt.Errorf("scoped item id %q: %w", string(id), err)
	}
	return nil
}

// NewScopedItemID builds a scoped work-item identity from its parts.
func NewScopedItemID(epic EpicID, item ItemID) ScopedItemID {
	return ScopedItemID(string(epic) + "/" + string(item))
}
