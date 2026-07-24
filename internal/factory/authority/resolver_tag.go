// SPDX-License-Identifier: Apache-2.0

// Package authority: resolver_tag.go contains the annotation-tag
// verifier and the small OID/identity helpers used by the resolver.
package authority

import (
	"fmt"
	"strings"
)

func verifyTag(git GitRunner, repoRoot string, resolved *ResolvedAuthority, headOID string) error {
	if resolved.TagName == "" {
		return nil
	}
	ref := "refs/tags/" + resolved.TagName
	objType, err := git(repoRoot, "cat-file", "-t", ref)
	if err != nil {
		return &AuthorityResolutionError{
			Status: AuthorityInvalidGitObject,
			Reason: fmt.Sprintf("tag %s missing: %v", resolved.TagName, err),
		}
	}
	if strings.TrimSpace(objType) != "tag" {
		return &AuthorityResolutionError{
			Status: AuthorityTagMismatch,
			Reason: fmt.Sprintf("tag %s is lightweight (type=%q); annotated required", resolved.TagName, strings.TrimSpace(objType)),
		}
	}
	peeled, err := git(repoRoot, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return &AuthorityResolutionError{
			Status: AuthorityInvalidGitObject,
			Reason: fmt.Sprintf("tag %s does not peel to a commit: %v", resolved.TagName, err),
		}
	}
	if resolved.TagObject == "" {
		if objOID, err := git(repoRoot, "rev-parse", "--verify", "--end-of-options", ref); err == nil {
			resolved.TagObject = objOID
		}
	}
	if resolved.TagTarget == "" {
		resolved.TagTarget = peeled
	}
	expected := headOID
	if resolved.ClosureCommit != "" {
		expected = resolved.ClosureCommit
	}
	if peeled != expected {
		return &AuthorityResolutionError{
			Status: AuthorityTagMismatch,
			Reason: fmt.Sprintf("tag %s peels to %s; expected %s", resolved.TagName, shortSHA(peeled), shortSHA(expected)),
		}
	}
	return nil
}

// mustResolveOID returns the full 40-char OID for short or partial
// refs, or empty when the resolution fails.
func mustResolveOID(git GitRunner, repoRoot, ref string) string {
	if strings.TrimSpace(ref) == "" {
		return ""
	}
	full, err := git(repoRoot, "rev-parse", "--verify", "--end-of-options", ref)
	if err != nil {
		return ""
	}
	return full
}

// readBlobAt returns the decoded blob content for the path at the
// given commit.
func readBlobAt(git GitRunner, repoRoot, commit, path string) (string, error) {
	out, err := git(repoRoot, "cat-file", "blob", commit+":"+path)
	if err != nil {
		return "", err
	}
	return out, nil
}

// isAncestor returns true when ancestor is an ancestor of
// descendant according to `git merge-base --is-ancestor`.
func isAncestor(git GitRunner, repoRoot, ancestor, descendant string) bool {
	if ancestor == "" || descendant == "" {
		return false
	}
	_, err := git(repoRoot, "merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}

// isValidActID enforces the canonical ACT identifier shape.
func isValidActID(id string) bool {
	if len(id) < 4 || len(id) > 200 {
		return false
	}
	if !strings.HasPrefix(id, "ACT-") {
		return false
	}
	for _, r := range id[4:] {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

// shortSHA returns the first 12 chars of an OID for diagnostic
// rendering. Long OIDs are not truncated to fewer than 7 chars so
// callers always get a usable prefix.
func shortSHA(oid string) string {
	if len(oid) <= 12 {
		return oid
	}
	return oid[:12]
}

// requireValidOID rejects placeholders and non-hex inputs.
func requireValidOID(oid string) error {
	if len(oid) < 7 {
		return fmt.Errorf("oid %q is shorter than 7 chars", oid)
	}
	for _, r := range oid {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return fmt.Errorf("oid %q contains non-hex characters", oid)
		}
	}
	return nil
}
