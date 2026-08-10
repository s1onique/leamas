// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// TestClosureObservationHashEmptyBytes is the B3-R5 regression
// proof: a clean observation (zero bytes) must hash to the
// SHA-256 of empty bytes, never an empty string. The B2
// barrier rejects non-64-hex hashes; returning "" for an
// observed empty stream would make a normal clean caller
// state look structurally invalid.
func TestClosureObservationHashEmptyBytes(t *testing.T) {
	sum := sha256.Sum256(nil)
	want := hex.EncodeToString(sum[:])
	if got := sha256OfBytes(nil); got != want {
		t.Fatalf("sha256OfBytes(nil) = %q, want %q", got, want)
	}
	if got := sha256OfBytes([]byte{}); got != want {
		t.Fatalf("sha256OfBytes([]byte{}) = %q, want %q", got, want)
	}
}

//TestClosureWorktreeInventoryHashCanonical is the B3-R5
// canonicality proof: the worktree-inventory hash is
// deterministic across insertion order and distinguishes
// different registration sets.
func TestClosureWorktreeInventoryHashCanonical(t *testing.T) {
	a := v2WorktreeRegistrationSet{{Path: "/repo/alpha"}, {Path: "/repo/beta"}}
	b := v2WorktreeRegistrationSet{{Path: "/repo/beta"}, {Path: "/repo/alpha"}}
	if sha256OfWorktreeInventory(a) != sha256OfWorktreeInventory(b) {
		t.Fatalf("worktree hash must be order-independent: a=%s b=%s",
			sha256OfWorktreeInventory(a), sha256OfWorktreeInventory(b))
	}
	c := v2WorktreeRegistrationSet{{Path: "/repo/alpha"}, {Path: "/repo/gamma"}}
	if sha256OfWorktreeInventory(a) == sha256OfWorktreeInventory(c) {
		t.Fatalf("worktree hash must distinguish different sets: a=%s c=%s",
			sha256OfWorktreeInventory(a), sha256OfWorktreeInventory(c))
	}
	// Empty set still hashes to a valid 64-hex value
	// (the SHA-256 of the versioned prefix alone).
	empty := v2WorktreeRegistrationSet{}
	sum := sha256.Sum256([]byte("worktree-inventory-v1\n"))
	if got, want := sha256OfWorktreeInventory(empty), hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("empty worktree hash = %q, want %q", got, want)
	}
	// The canonical serialization must start with the
	// versioned prefix and use NUL as a delimiter.
	buf := bytes.NewBufferString("worktree-inventory-v1\n")
	for _, e := range a {
		buf.WriteString(e.Path)
		buf.WriteByte(0)
	}
	sum = sha256.Sum256(buf.Bytes())
	if got, want := sha256OfWorktreeInventory(a), hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("non-empty worktree hash = %q, want %q", got, want)
	}
}
