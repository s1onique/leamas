// SPDX-License-Identifier: Apache-2.0

// Package digest: generator_binding_adapter.go wires the pure
// EvaluateGeneratorBinding classifier to the resolved identity
// the digest pipeline already produces. The adapter performs
// no Git subprocesses; it consumes only fields already resolved
// by the authority resolver and the local dirty-detection
// pre-pass.
//
// The adapter is the single boundary between the digest
// pipeline and the typed binding vocabulary. Renderers MUST
// consume the resulting GeneratorBinding record and MUST NOT
// re-derive any per-axis verdict from the underlying fields.
package digest

import (
	"regexp"
	"strings"
)

// fullOIDPattern matches 40+ hex chars used as a full Git SHA.
// Defined locally to avoid an import cycle on auto_range.go's
// exported symbols.
var generatorBindingFullOIDPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)

// ResolveGeneratorBinding translates a ResolvedMode (or
// pre-resolution analogue) into the typed binding inputs and
// invokes the pure classifier. It performs no I/O.
//
// The caller MUST supply:
//
//   - generator commit (the binary's embedded VCS revision)
//   - repository HEAD commit (already resolved by the authority
//     resolver or by the local pre-pass)
//   - digest subject commit (already resolved)
//   - dirty flag (already detected)
//
// The adapter's only job is to translate these strings into the
// GeneratorIdentity / RepositoryIdentity / DigestAuthoritySubject
// shape with proper validity flags. The classifier's verdict is
// the single source of truth.
func ResolveGeneratorBinding(generatorCommit, repoHead, subjectCommit string, dirty bool) GeneratorBinding {
	return EvaluateGeneratorBinding(
		buildGeneratorIdentity(generatorCommit),
		buildRepositoryIdentity(repoHead),
		buildDigestAuthoritySubject(subjectCommit, dirty),
	)
}

// buildGeneratorIdentity classifies the generator commit string
// into the typed GeneratorIdentity record. The validity flag is
// set true iff the string is a non-empty hex string that matches
// the 40-64 char full-OID pattern.
func buildGeneratorIdentity(commit string) GeneratorIdentity {
	commit = strings.TrimSpace(commit)
	if commit == "" {
		return GeneratorIdentity{Commit: "", CommitIsValid: false}
	}
	if generatorBindingFullOIDPattern.MatchString(strings.ToLower(commit)) {
		return GeneratorIdentity{Commit: strings.ToLower(commit), CommitIsValid: true}
	}
	return GeneratorIdentity{Commit: commit, CommitIsValid: false}
}

// buildRepositoryIdentity classifies the repository HEAD string
// into the typed RepositoryIdentity record using the same rules
// as buildGeneratorIdentity.
func buildRepositoryIdentity(head string) RepositoryIdentity {
	head = strings.TrimSpace(head)
	if head == "" {
		return RepositoryIdentity{HeadCommit: "", HeadCommitIsValid: false}
	}
	if generatorBindingFullOIDPattern.MatchString(strings.ToLower(head)) {
		return RepositoryIdentity{HeadCommit: strings.ToLower(head), HeadCommitIsValid: true}
	}
	return RepositoryIdentity{HeadCommit: head, HeadCommitIsValid: false}
}

// buildDigestAuthoritySubject classifies the digest subject
// string into the typed DigestAuthoritySubject record. The
// dirty flag is propagated verbatim from the caller.
func buildDigestAuthoritySubject(subject string, dirty bool) DigestAuthoritySubject {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return DigestAuthoritySubject{SubjectCommit: "", SubjectCommitIsValid: false, Dirty: dirty}
	}
	if generatorBindingFullOIDPattern.MatchString(strings.ToLower(subject)) {
		return DigestAuthoritySubject{SubjectCommit: strings.ToLower(subject), SubjectCommitIsValid: true, Dirty: dirty}
	}
	return DigestAuthoritySubject{SubjectCommit: subject, SubjectCommitIsValid: false, Dirty: dirty}
}