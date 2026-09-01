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

// TestClosureWorktreeInventoryHashCanonical is the B3-R5
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

// TestClosureWorktreeInventoryHashMatrix is the F2.1-F2.5
// matrix mandated by the F2 protocol-authority correction.
// Each row exercises one branch of the v1 + Path-only
// encoding the production code now emits.
func TestClosureWorktreeInventoryHashMatrix(t *testing.T) {
	const prefix = "worktree-inventory-v1\n"
	// F2.1 empty inventory still hashes to the versioned
	// prefix alone.
	sumEmpty := sha256.Sum256([]byte(prefix))
	wantEmpty := hex.EncodeToString(sumEmpty[:])
	if got := sha256OfWorktreeInventory(v2WorktreeRegistrationSet{}); got != wantEmpty {
		t.Fatalf("F2.1 empty hash = %q, want %q", got, wantEmpty)
	}
	// F2.2 single-entry inventory.
	single := v2WorktreeRegistrationSet{{Path: "/repo/alpha"}}
	buf := bytes.NewBufferString(prefix)
	buf.WriteString("/repo/alpha")
	buf.WriteByte(0)
	wantSingleSum := sha256.Sum256(buf.Bytes())
	wantSingle := hex.EncodeToString(wantSingleSum[:])
	if got := sha256OfWorktreeInventory(single); got != wantSingle {
		t.Fatalf("F2.2 single hash = %q, want %q", got, wantSingle)
	}
	// F2.3 multi-entry inventory hashes to a different
	// value and matches the explicit encoding.
	multi := v2WorktreeRegistrationSet{{Path: "/repo/alpha"}, {Path: "/repo/beta"}}
	if got := sha256OfWorktreeInventory(multi); got == wantSingle {
		t.Fatalf("F2.3 multi hash %q equals single hash %q", got, wantSingle)
	}
	buf.Reset()
	buf.WriteString(prefix)
	buf.WriteString("/repo/alpha")
	buf.WriteByte(0)
	buf.WriteString("/repo/beta")
	buf.WriteByte(0)
	wantMultiSum := sha256.Sum256(buf.Bytes())
	wantMulti := hex.EncodeToString(wantMultiSum[:])
	if got := sha256OfWorktreeInventory(multi); got != wantMulti {
		t.Fatalf("F2.3 multi hash = %q, want %q", got, wantMulti)
	}
	// F2.4 insertion order independence.
	reversed := v2WorktreeRegistrationSet{{Path: "/repo/beta"}, {Path: "/repo/alpha"}}
	if sha256OfWorktreeInventory(multi) != sha256OfWorktreeInventory(reversed) {
		t.Fatalf("F2.4 hash must be order-independent")
	}
	// F2.5 single-field mutation changes the hash.
	mutated := v2WorktreeRegistrationSet{{Path: "/repo/alpha"}, {Path: "/repo/BETA"}}
	if sha256OfWorktreeInventory(multi) == sha256OfWorktreeInventory(mutated) {
		t.Fatalf("F2.5 hash must distinguish beta/BETA case")
	}
	// Hash field changes MUST NOT change the hash because
	// the v1 protocol hashes paths only.
	hashed := v2WorktreeRegistrationSet{
		{Path: "/repo/alpha", Hash: "1111111111111111111111111111111111111111"},
		{Path: "/repo/beta", Hash: "2222222222222222222222222222222222222222"},
	}
	if sha256OfWorktreeInventory(multi) != sha256OfWorktreeInventory(hashed) {
		t.Fatalf("F2.5b Hash field MUST NOT influence the hash")
	}
}

// TestClosureWorktreeInventoryHashProtocolInvariant is
// the F2 protocol-coherence check: empty and non-empty
// must use the SAME declared protocol. The historical
// defect (empty -> v1, non-empty -> v2) cannot recur.
func TestClosureWorktreeInventoryHashProtocolInvariant(t *testing.T) {
	empty := sha256OfWorktreeInventory(v2WorktreeRegistrationSet{})
	nonEmpty := sha256OfWorktreeInventory(v2WorktreeRegistrationSet{
		{Path: "/repo/anything"},
	})
	if len(empty) != 64 {
		t.Fatalf("empty hash %q is not 64 chars", empty)
	}
	if len(nonEmpty) != 64 {
		t.Fatalf("non-empty hash %q is not 64 chars", nonEmpty)
	}
	if empty == nonEmpty {
		t.Fatalf("empty and non-empty hashes MUST differ")
	}
}
