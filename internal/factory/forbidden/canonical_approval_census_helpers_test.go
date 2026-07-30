// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"
	"sort"
	"strings"
	"testing"
)

// approvalCensusRecord is one observation of the configured-approval census.
//
// The census deliberately uses only _test.go state so production callers see
// no new API. The record correlates each configured approval against the
// resolved caller declaration from the canonical package graph. There is no
// Effective field: validateApprovalSchema guarantees that the configured
// record equals the value the runtime would have used.
type approvalCensusRecord struct {
	Index int

	Configured ApprovedCaller

	ResolvedCallerPackage  string
	ResolvedCallerFunction string
	ResolvedCallerReceiver string
	ResolvedCallerKind     string

	CallerResolved  bool
	KindMatches     bool
	ReceiverMatches bool
}

// runProductionCanonicalAnalysisForCensus runs the production canonical
// analysis using the same loader invocation as runCanonicalAnalysis and
// returns both the public canonicalResult and the private analysis state
// required for census assertions.
//
// The seam exists entirely in _test.go so production callers see no new
// API. It mirrors the body of runCanonicalAnalysis so that types.Object
// identities are guaranteed to come from the same coherent package graph
// the approval validation uses.
func runProductionCanonicalAnalysisForCensus(t *testing.T) (canonicalResult, *canonicalAnalysis) {
	t.Helper()
	policy, err := NewDupcodeBypassPolicy(repoRoot(t), testModulePath)
	if err != nil {
		t.Fatalf("dupcode policy: %v", err)
	}
	config := productionCanonicalConfig()
	analysis := newCanonicalAnalysis(policy, config)
	discovered, err := policy.DiscoverProductionFilesRepoWide()
	if err != nil {
		analysis.addFinding(policy.repoRoot, "dupcode_discovery_error", err.Error(),
			tokenPosition{}, CallerIdentity{}, ProtectedSymbol{}, "")
		return analysis.result(), analysis
	}
	analysis.loadAndValidatePackages()
	if !analysis.invalid {
		analysis.resolveProtectedDeclarations()
		analysis.resolveCallerDeclarations()
		analysis.resolveConfiguredApprovals()
		analysis.scanProtectedReferences()
		analysis.validateObservedEdges()
		analysis.validateConfiguredApprovals()
	}
	analysis.reconcileCoverage(discovered)
	return analysis.result(), analysis
}

// buildApprovalCensus assembles one approvalCensusRecord per configured
// approval, correlating the configured approval with its resolved
// declaration.
func buildApprovalCensus(analysis *canonicalAnalysis) []approvalCensusRecord {
	configured := analysis.config.approvals
	records := make([]approvalCensusRecord, len(configured))
	for index, approval := range configured {
		identity := approvalCallerIdentity(approval)
		resolved := analysis.callersByIdentity[identity]

		record := approvalCensusRecord{
			Index:      index,
			Configured: approval,
		}
		if resolved != nil {
			record.CallerResolved = true
			record.ResolvedCallerPackage = resolved.Pkg().Path()
			if function, ok := resolved.(*types.Func); ok {
				record.ResolvedCallerFunction = function.Name()
				signature, _ := function.Type().(*types.Signature)
				if signature != nil {
					if signature.Recv() == nil {
						record.ResolvedCallerKind = CallerKindPackageFunction
						record.ResolvedCallerReceiver = ""
					} else {
						record.ResolvedCallerKind = CallerKindMethod
						record.ResolvedCallerReceiver = recvTypeNameFromSig(signature.Recv())
					}
				}
			} else if variable, ok := resolved.(*types.Var); ok {
				record.ResolvedCallerKind = CallerKindVariableInitializer
				record.ResolvedCallerReceiver = ""
				if variable.Parent() != nil {
					record.ResolvedCallerFunction = "<var-init:" + variable.Name() + ">"
				}
			}
		}

		record.KindMatches = record.CallerResolved &&
			record.ResolvedCallerKind == approval.CallerKind
		record.ReceiverMatches = record.CallerResolved &&
			record.ResolvedCallerReceiver == approval.Receiver

		records[index] = record
	}
	return records
}

// declarationObjectForRecord returns the resolved types.Object for the
// caller identity that the record was built from.
func declarationObjectForRecord(analysis *canonicalAnalysis, record approvalCensusRecord) types.Object {
	identity := approvalCallerIdentity(record.Configured)
	return analysis.callersByIdentity[identity]
}

// serializeCensusRecord emits the deterministic normalized representation
// of one record used to compute the frozen strict-schema oracle hash.
func serializeCensusRecord(record approvalCensusRecord) string {
	approval := record.Configured
	parts := []string{
		approval.PackagePath,
		approval.Function,
		approval.Receiver,
		approval.CallerKind,
		string(approval.Callee.Layer),
		approval.Callee.PackagePath,
		string(approval.Callee.Kind),
		approval.Callee.Receiver,
		approval.Callee.Name,
		string(approval.ReferenceClass),
		censusItoa(approval.Cardinality),
	}
	return strings.Join(parts, "|")
}

// censusItoa is a tiny strconv.Itoa equivalent used by the oracle
// serializer to keep the helpers file free of strconv dependencies.
func censusItoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := false
	if value < 0 {
		negative = true
		value = -value
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// censusRecordIdentity returns the canonical sort key for one record
// based on the complete approval identity.
func censusRecordIdentity(record approvalCensusRecord) []string {
	approval := record.Configured
	return []string{
		approval.PackagePath,
		approval.Function,
		approval.Receiver,
		approval.CallerKind,
		string(approval.Callee.Layer),
		approval.Callee.PackagePath,
		string(approval.Callee.Kind),
		approval.Callee.Receiver,
		approval.Callee.Name,
		string(approval.ReferenceClass),
		censusItoa(approval.Cardinality),
	}
}

// hashCensusRecords produces the deterministic SHA-256 hash of the
// frozen explicit approval set in canonical sorted order.
func hashCensusRecords(records []approvalCensusRecord) string {
	sorted := make([]approvalCensusRecord, len(records))
	copy(sorted, records)
	sort.Slice(sorted, func(i, j int) bool {
		left := censusRecordIdentity(sorted[i])
		right := censusRecordIdentity(sorted[j])
		for index := range left {
			if left[index] != right[index] {
				return left[index] < right[index]
			}
		}
		return false
	})
	var builder strings.Builder
	for index, record := range sorted {
		builder.WriteString(censusItoa(index))
		builder.WriteByte(0)
		builder.WriteString(serializeCensusRecord(record))
		builder.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

// findRecordsByFunction filters the census to records whose configured
// function name matches.
func findRecordsByFunction(records []approvalCensusRecord, function string) []approvalCensusRecord {
	var matches []approvalCensusRecord
	for _, record := range records {
		if record.Configured.Function == function {
			matches = append(matches, record)
		}
	}
	return matches
}

// countByKind returns the number of records whose getter returns kind.
func countByKind(records []approvalCensusRecord, getter func(approvalCensusRecord) string, kind string) int {
	count := 0
	for _, record := range records {
		if getter(record) == kind {
			count++
		}
	}
	return count
}

// configuredCallerKind reads the configured caller kind for a record.
func configuredCallerKind(record approvalCensusRecord) string {
	return record.Configured.CallerKind
}

// resolvedCallerKind reads the resolved caller kind for a record.
func resolvedCallerKind(record approvalCensusRecord) string {
	return record.ResolvedCallerKind
}
