package doctrine

import (
	"os"
	"path/filepath"
	"strings"
)

// ecfMaxSymlinkDepth bounds the number of symlink hops the resolver
// will follow before declaring the path too complex to classify.
const ecfMaxSymlinkDepth = 40

// ecfConfinedByWalk resolves the configured relative path under a
// persistent os.Root, following symlinks but rejecting any chain that
// ultimately references a location outside the supplied root.
//
// The resolver maintains a queue of pending path components and
// processes them one at a time. When a symlink is encountered its
// target is read with r.Readlink (which does not require the target to
// exist), absolute targets are rejected unconditionally, and relative
// targets have their components spliced into the pending queue so
// multi-hop chains are classified correctly.
//
// The resolver stops after ecfMaxSymlinkDepth hops so a symlink loop
// cannot hang the verifier; the caller is expected to convert that
// condition into a deterministic ECF010 finding.
//
// The absolute position of the resolver cursor is maintained as
// absRoot + relCursor. Every step (regular, "..", or symlink splice)
// is verified to keep absCursor lexically inside absRoot.
func ecfConfinedByWalk(r *os.Root, root, rel string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}

	initial := splitPathComponents(rel)
	if len(initial) == 0 {
		return false
	}

	pending := append([]string(nil), initial...)
	// relCursor is the relative path under root that has been resolved
	// so far. The absolute position of the cursor is filepath.Join(absRoot, relCursor).
	relCursor := ""
	hops := 0

	for len(pending) > 0 {
		p := pending[0]
		pending = pending[1:]
		if p == "" || p == "." {
			continue
		}

		// Compute the new absolute cursor for this step. For symlink
		// targets, this is the position after following the link.
		var newAbsCursor string
		if p == ".." {
			// Step up from current relCursor. Use relCursor directly to
			// avoid losing the root prefix.
			if relCursor == "" {
				// Already at root; ".." cannot step up.
				return true
			}
			newRel := filepath.Dir(relCursor)
			if newRel == relCursor {
				// Already at a top-level component; ".." cannot step up.
				return true
			}
			newAbsCursor = filepath.Join(absRoot, newRel)
			relCursor = newRel
			continue
		}

		stepRel := filepath.Join(relCursor, p)
		newAbsCursor = filepath.Join(absRoot, stepRel)

		// Pre-check: confirm this step stays inside the root
		// lexically. r.Lstat below will also reject escapes, but doing
		// it lexically lets us classify confinement eagerly for ".."
		// steps and dangling symlink targets.
		if !strings.HasPrefix(newAbsCursor, absRoot+string(filepath.Separator)) &&
			newAbsCursor != absRoot {
			return true
		}

		info, err := r.Lstat(stepRel)
		if err != nil {
			// Lstat fails when the path escapes root or when a
			// component is missing. The lexical check above handles the
			// escape case; other failures (missing/inaccessible) are
			// not confinement claims and are deferred to the open path.
			return false
		}

		if info.Mode()&os.ModeSymlink == 0 {
			relCursor = stepRel
			continue
		}

		// Symlink. Bound the hop count.
		hops++
		if hops > ecfMaxSymlinkDepth {
			return true
		}

		target, err := r.Readlink(stepRel)
		if err != nil {
			return false
		}

		// Absolute symlink targets are always rejected (the os.Root
		// contract forbids absolute symlinks; any absolute target is
		// by definition outside the root).
		if filepath.IsAbs(target) {
			return true
		}

		// Relative target. Splice its components into the pending
		// queue so multi-hop chains are resolved correctly. The
		// components resolve relative to relCursor (the symlink's
		// directory), not the absolute cursor.
		pending = append(splitPathComponents(target), pending...)
	}

	return false
}

// splitPathComponents splits a relative or absolute path into
// non-empty components, removing "." entries.
func splitPathComponents(p string) []string {
	parts := strings.Split(filepath.Clean(p), string(filepath.Separator))
	out := make([]string, 0, len(parts))
	for _, x := range parts {
		if x == "" || x == "." {
			continue
		}
		out = append(out, x)
	}
	return out
}

// pathInsideRoot reports whether path is inside (or equal to) root.
// Both arguments are expected to be absolute. Comparison is lexical.
func pathInsideRoot(path, root string) bool {
	if path == root {
		return true
	}
	return strings.HasPrefix(path, root+string(filepath.Separator))
}
