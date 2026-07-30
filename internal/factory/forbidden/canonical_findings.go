// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"crypto/sha256"
	"encoding/hex"
	"go/token"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

func (a *canonicalAnalysis) addFinding(
	path, kind, message string,
	position tokenPosition,
	caller CallerIdentity,
	callee ProtectedSymbol,
	class ReferenceClass,
) {
	a.findings = append(a.findings, canonicalFinding{
		finding: checks.Finding{
			Path:     path,
			Kind:     kind,
			Message:  message,
			Severity: checks.SeverityError,
		},
		position: position,
		caller:   caller,
		callee:   callee,
		class:    class,
	})
}

// tokenPosition aliases token.Position without importing go/token at every
// finding call site.
type tokenPosition = token.Position

func (a *canonicalAnalysis) result() canonicalResult {
	sort.Slice(a.findings, func(i, j int) bool {
		left, right := a.findings[i], a.findings[j]
		if left.finding.Path != right.finding.Path {
			return left.finding.Path < right.finding.Path
		}
		if left.position.Line != right.position.Line {
			return left.position.Line < right.position.Line
		}
		if left.position.Column != right.position.Column {
			return left.position.Column < right.position.Column
		}
		if left.finding.Kind != right.finding.Kind {
			return left.finding.Kind < right.finding.Kind
		}
		if callerIdentityString(left.caller) != callerIdentityString(right.caller) {
			return callerIdentityString(left.caller) < callerIdentityString(right.caller)
		}
		if protectedSymbolString(left.callee) != protectedSymbolString(right.callee) {
			return protectedSymbolString(left.callee) < protectedSymbolString(right.callee)
		}
		if left.class != right.class {
			return left.class < right.class
		}
		return left.finding.Message < right.finding.Message
	})

	findings := make([]checks.Finding, len(a.findings))
	for i := range a.findings {
		findings[i] = a.findings[i].finding
	}
	sortObservedEdges(a.observedEdges)
	return canonicalResult{
		Findings:      findings,
		ObservedEdges: append([]ObservedEdge(nil), a.observedEdges...),
		Stats: canonicalStats{
			ConfiguredProtectedSymbols: len(a.config.protected),
			ResolvedProtectedObjects:   len(a.protectedByObject),
			ConfiguredApprovals:        len(a.config.approvals),
			ObservedEdges:              len(a.observedEdges),
			ValidatedApprovals:         countValidatedApprovals(a.approvalStates),
		},
		NormalizedFindingHash: normalizedFindingHash(findings),
	}
}

func sortObservedEdges(edges []ObservedEdge) {
	sort.Slice(edges, func(i, j int) bool {
		left, right := edges[i], edges[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.Position.Line != right.Position.Line {
			return left.Position.Line < right.Position.Line
		}
		if left.Position.Column != right.Position.Column {
			return left.Position.Column < right.Position.Column
		}
		if callerIdentityString(left.Caller) != callerIdentityString(right.Caller) {
			return callerIdentityString(left.Caller) < callerIdentityString(right.Caller)
		}
		if protectedSymbolString(left.Callee) != protectedSymbolString(right.Callee) {
			return protectedSymbolString(left.Callee) < protectedSymbolString(right.Callee)
		}
		return left.ReferenceClass < right.ReferenceClass
	})
}

func normalizedFindingHash(findings []checks.Finding) string {
	var normalized strings.Builder
	for _, finding := range findings {
		normalized.WriteString(finding.Path)
		normalized.WriteByte(0)
		normalized.WriteString(finding.Kind)
		normalized.WriteByte(0)
		normalized.WriteString(finding.Message)
		normalized.WriteByte(0)
		normalized.WriteString(string(finding.Severity))
		normalized.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(normalized.String()))
	return hex.EncodeToString(sum[:])
}

func callerIdentityString(identity CallerIdentity) string {
	return identity.PackagePath + "|" + identity.Kind + "|" + identity.Receiver + "|" + identity.Function
}

func protectedSymbolString(symbol ProtectedSymbol) string {
	return string(symbol.Layer) + "|" + symbol.PackagePath + "|" + string(symbol.Kind) + "|" + symbol.Receiver + "|" + symbol.Name
}
