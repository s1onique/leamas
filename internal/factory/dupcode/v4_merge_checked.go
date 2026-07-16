package dupcode

import "fmt"

// v4MergeToNWayCloneChecked validates a complete merge group before the
// compatibility panic-based merger is invoked. Production component
// materialization uses the error-returning invariant helper directly.
func v4MergeToNWayCloneChecked(group []v4InternalFinding) (v4InternalFinding, error) {
	if len(group) == 0 {
		return v4InternalFinding{}, nil
	}
	count := group[0].TokenCount
	var flattened []maximalOccurrence
	for _, finding := range group {
		if finding.TokenCount != count {
			return v4InternalFinding{}, fmt.Errorf("dupcode: inconsistent token counts in v4 merge group: %d vs %d", count, finding.TokenCount)
		}
		flattened = append(flattened, finding.Occurrences...)
	}
	if err := validateOccurrenceIdentityInvariants(flattened); err != nil {
		return v4InternalFinding{}, err
	}
	return v4MergeToNWayClone(group), nil
}
