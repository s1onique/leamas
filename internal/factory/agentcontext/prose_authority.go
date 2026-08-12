// Authority source model.
//
// A guard phrase is only authoritative if it identifies a NAMED
// authority source: the current ACT, or a validated frozen closure
// plan. Generic vocabulary such as "explicitly authorized", "when
// delegated", or "leamas factory" MUST NOT by itself constitute
// authority. Such guards FAIL closed.
//
// Each guard phrase carries a non-empty AuthoritySource identifier.
// The detector accepts only phrases whose source is recognized.

package agentcontext

import "strings"

// AuthoritySource enumerates the named authority sources that a guard
// phrase may identify.
type AuthoritySource string

const (
	AuthorityNone          AuthoritySource = ""
	AuthorityCurrentACT    AuthoritySource = "current_act"
	AuthorityValidatedPlan AuthoritySource = "validated_frozen_plan"
)

// GuardPhrase is a single authoritative guard phrase. The Source
// field identifies the authority source that the phrase invokes.
type GuardPhrase struct {
	Source AuthoritySource
	Forms  []string
}

// GuardPhraseKind is a stable identifier for one guard phrase family.
type GuardPhraseKind string

const (
	GuardExplicitlyAuthorizedByACT GuardPhraseKind = "explicitly_authorized_by_act"
	GuardWhenDelegatedByACT        GuardPhraseKind = "when_delegated_by_act"
	GuardOnlyWhenACTAuth           GuardPhraseKind = "only_when_act_authorizes"
	GuardUnlessACTAuthorizes       GuardPhraseKind = "unless_act_authorizes"
	GuardWhenACTDelegates          GuardPhraseKind = "when_act_delegates"
	GuardOnlyIfDelegated           GuardPhraseKind = "only_if_delegated"
	GuardWhenValidatedPlanDeclares GuardPhraseKind = "when_validated_plan_declares"
	GuardWhenValidatedPlanAuth     GuardPhraseKind = "when_validated_plan_authorizes"
)

// authorityGuards is the canonical list of authoritative guard phrases.
// Each phrase MUST identify a named authority source. Generic
// vocabulary such as "when authorized" or "factory close" is
// deliberately excluded.
var authorityGuards = []GuardPhrase{
	// ----- current ACT guards -----
	{
		Source: AuthorityCurrentACT,
		Forms: []string{
			"explicitly authorized by the current act",
			"explicitly authorized by the act",
		},
	},
	{
		Source: AuthorityCurrentACT,
		Forms: []string{
			"when the current act authorizes this exact command",
			"when the act authorizes this exact command",
		},
	},
	{
		Source: AuthorityCurrentACT,
		Forms: []string{
			"only when the current act authorizes",
			"only when the act authorizes",
		},
	},
	{
		Source: AuthorityCurrentACT,
		Forms: []string{
			"unless the current act explicitly authorizes",
			"unless the act explicitly authorizes",
		},
	},
	{
		Source: AuthorityCurrentACT,
		Forms: []string{
			"when delegated by the current act",
			"when delegated by the act",
			"only when explicitly delegated by the current act",
			"only when explicitly delegated by the act",
		},
	},
	{
		Source: AuthorityCurrentACT,
		Forms: []string{
			"when the current act delegates",
			"when the act delegates",
			"only when the current act delegates",
			"only when the act delegates",
		},
	},
	{
		Source: AuthorityCurrentACT,
		Forms: []string{
			"only if delegated by the current act",
			"only if delegated by the act",
		},
	},
	{
		Source: AuthorityCurrentACT,
		Forms: []string{
			"only when the current act delegates commit authority",
			"only when the act delegates commit authority",
			"only when the current act delegates push authority",
			"only when the act delegates push authority",
			"only when the current act delegates tag authority",
			"only when the act delegates tag authority",
		},
	},

	// ----- validated frozen closure plan guards -----
	{
		Source: AuthorityValidatedPlan,
		Forms: []string{
			"when declared by the validated closure plan",
			"when authorized by the validated closure plan",
			"when declared by the frozen closure plan",
			"when authorized by the frozen closure plan",
			"when the validated closure plan declares",
			"when the frozen closure plan declares",
		},
	},
}

// GuardMatch is a positional match of an authority guard phrase in a
// unit. It records the authority source of the match.
type GuardMatch struct {
	Source AuthoritySource
	Kind   GuardPhraseKind
	Form   string
	Start  int
	End    int
}

// findGuardMatches returns every positional authority guard match in
// the lower-cased, whitespace-normalized unit. Generic phrases that
// do not identify a named authority source are not represented.
func findGuardMatches(lowerUnit string) []GuardMatch {
	var matches []GuardMatch
	for _, guard := range authorityGuards {
		for _, form := range guard.Forms {
			pos := 0
			for {
				idx := strings.Index(lowerUnit[pos:], form)
				if idx == -1 {
					break
				}
				absStart := pos + idx
				matches = append(matches, GuardMatch{
					Source: guard.Source,
					Form:   form,
					Start:  absStart,
					End:    absStart + len(form),
				})
				pos = absStart + len(form)
				if pos >= len(lowerUnit) {
					break
				}
			}
		}
	}
	return matches
}

// unitHasAuthoritySource reports whether the unit contains at least
// one guard phrase from any non-empty AuthoritySource.
func unitHasAuthoritySource(lowerUnit string) bool {
	for _, m := range findGuardMatches(lowerUnit) {
		if m.Source != AuthorityNone {
			return true
		}
	}
	return false
}
