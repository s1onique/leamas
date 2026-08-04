// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"slices"

	"github.com/s1onique/leamas/internal/factory/registry"
)

// computeRegistryDigest computes a canonical SHA-256 digest of the
// authorized verifier set. The digest includes: verifier IDs, lanes,
// authorities, operations, and execution metadata. Ordering is
// canonical (sorted by verifier ID).
func (d *Dispatcher) computeRegistryDigest(requests []ProfileRequest, resolved []*registry.Verifier) [32]byte {
	type entry struct {
		id  string
		v   *registry.Verifier
		req ProfileRequest
	}
	entries := make([]entry, 0, len(resolved))
	for i, v := range resolved {
		if v != nil {
			entries = append(entries, entry{
				id:  v.Name,
				v:   v,
				req: requests[i],
			})
		}
	}
	slices.SortFunc(entries, func(left, right entry) int {
		return cmpString(left.id, right.id)
	})
	h := sha256.New()
	for _, e := range entries {
		writeString(h, e.v.Name)
		writeString(h, string(e.v.Lane))
		writeString(h, string(e.v.Authority))
		writeString(h, string(e.req.Operation))
		writeString(h, string(e.v.Execution.Kind))
		writeString(h, e.v.Execution.ImplementationID)
	}
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return digest
}

// writeString writes a length-prefixed string to the hash using
// streaming writes.
func writeString(h hash.Hash, s string) {
	var length [4]byte
	binary.LittleEndian.PutUint32(length[:], uint32(len(s)))
	h.Write(length[:])
	h.Write([]byte(s))
}

// cmpString compares two strings lexicographically.
func cmpString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
