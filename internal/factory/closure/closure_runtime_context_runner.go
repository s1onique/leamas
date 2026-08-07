// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"fmt"
)

// RunClosureProtocolRuntimeContext is the public entry point
// for the immutable-subject execution path. It validates the
// closure protocol version, then delegates to the single
// private sealed runner that selects the F < S topology mode.
func RunClosureProtocolRuntimeContext(ctx context.Context, req V2Request, identity V2BinaryIdentity) (V2Manifest, error) {
	if req.ClosureProtocolVersion != ClosureProtocolV2 {
		return V2Manifest{}, NewV2ErrorWith(V2CodeUnsupportedClosureProtocolVersion,
			fmt.Sprintf("runtime context requires closure_protocol_version=%q, got %q",
				string(ClosureProtocolV2),
				string(req.ClosureProtocolVersion)),
			"closure_protocol_version", string(req.ClosureProtocolVersion))
	}
	deps := DefaultV2RunnerDeps()
	deps.BinaryIdentity = identity
	return runClosureProtocolV2WithDepsAndTopology(ctx, req, deps, executionTopologyFreezeBeforeSubject)
}
