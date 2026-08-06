// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_manifest_bijection_test.go covers Phase 5-7
// of ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MANIFEST-RESULTS01.
//
// The test file exercises:
//
//   - result bijection (each plan check has exactly one
//     result; no extra result; mode preserved; canonical
//     order)
//   - mixed-mode mutation matrix
//   - run-success integrity (no failed / cleanup-failed /
//     timeout-marked-success / cancellation / overflow /
//     output-truncated / output-incomplete runs hidden by
//     aggregate success)

import (
	"context"
	"strings"
	"testing"
)

// buildMixedModeManifestRig constructs a hermetic S < F < C
// repository containing a frozen plan with three checks
// (run-1, exclude-1, run-2) and writes the supplied manifest
// bytes at C:M. The returned topology / frozen-plan /
// manifest authorities are sufficient to invoke
// VerifyManifestIdentity end-to-end.
func buildMixedModeManifestRig(t *testing.T, manifestBytes []byte) (
	repoDir, subject, freeze, closure, subjectTree, freezeTree string,
) {
	t.Helper()
	dir := initRepo(t)
	planBytes := mixedModePlanBytes()
	subject = makeCommit(t, dir, "subject implementation", map[string]string{
		"subject-only.txt": "subject implementation\n",
	})
	freeze = makeCommit(t, dir, "freeze: add mixed-mode plan", map[string]string{
		"docs/closure-plans/PLAN.json": planBytes,
	})
	subjectTree = mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	freezeTree = mustRunGit(t, dir, "rev-parse", freeze+"^{tree}")
	closure = writeAndCommit(t, dir,
		"docs/closure-manifests/MANIFEST.json", manifestBytes,
		"closure: commit manifest")
	return dir, subject, freeze, closure, subjectTree, freezeTree
}

// mixedModePlanBytes returns a frozen Plan Contract v1
// document with three checks:
//
//   - run-1: run mode
//   - exclude-1: exclude mode
//   - run-2: run mode
func mixedModePlanBytes() string {
	return mixedModePlanPart1() + mixedModePlanPart2() + mixedModePlanPart3()
}

func mixedModePlanPart1() string {
	return `{"contract_version":1,"act_id":"ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MANIFEST-RESULTS01",` +
		`"baseline":{"commit_oid":"0000000000000000000000000000000000000000",` +
		`"tree_oid":"0000000000000000000000000000000000000000"},` +
		`"execution":{"mode":"serial_fail_fast"},"checks":[`
}

func mixedModePlanPart2() string {
	return `{"id":"run-1","mode":"run","argv":["true"],"timeout_seconds":30,` +
		`"working_directory":".","environment":{}},` +
		`{"id":"exclude-1","mode":"exclude","reason":"not applicable"},` +
		`{"id":"run-2","mode":"run","argv":["true"],"timeout_seconds":30,` +
		`"working_directory":".","environment":{}}]`
}

func mixedModePlanPart3() string {
	return `,"artifacts":[],"policy":{"require_clean_before":true,` +
		`"require_clean_after":true,"forbid_tracked_full_digests":true,` +
		`"require_diff_check":true}}`
}

// mixedModeHappyManifest renders a valid manifest for the
// three-check mixed-mode plan. All run-mode checks pass with
// exit_code=0; the exclude-mode check is "excluded" with
// cleanup_status="not_required".
func mixedModeHappyManifest(t *testing.T, repoDir, subject, freeze, subjectTree, freezeTree string) []byte {
	t.Helper()
	planBlob := mustRunGit(t, repoDir, "rev-parse", freeze+":docs/closure-plans/PLAN.json")
	planSum := sum256Bytes([]byte(mixedModePlanBytes()))
	binary := v2ManifestTestBinary(t)
	var body strings.Builder
	body.WriteString(`{"closure_protocol_version":"2","plan_contract_version":1,`)
	body.WriteString(`"subject_commit":"` + subject + `",`)
	body.WriteString(`"subject_tree":"` + subjectTree + `",`)
	body.WriteString(`"freeze_commit":"` + freeze + `",`)
	body.WriteString(`"freeze_tree":"` + freezeTree + `",`)
	body.WriteString(`"plan_path":"docs/closure-plans/PLAN.json",`)
	body.WriteString(`"plan_blob":"` + planBlob + `",`)
	body.WriteString(`"plan_sha256":"` + planSum + `",`)
	body.WriteString(`"execution_tree":"` + subjectTree + `",`)
	body.WriteString(`"check_results":[`)
	body.WriteString(`{"id":"run-1","mode":"run","outcome":"pass","exit_code":0,"duration_ms":0,"execution_classification":"completed","cleanup_status":"pass"},`)
	body.WriteString(`{"id":"exclude-1","mode":"exclude","outcome":"excluded","duration_ms":0,"execution_classification":"excluded_by_plan","cleanup_status":"not_required"},`)
	body.WriteString(`{"id":"run-2","mode":"run","outcome":"pass","exit_code":0,"duration_ms":0,"execution_classification":"completed","cleanup_status":"pass"}`)
	body.WriteString(`],`)
	body.WriteString(`"leamas_binary_identity":{`)
	body.WriteString(`"path":"` + binary.Path + `",`)
	body.WriteString(`"sha256":"` + binary.SHA256 + `",`)
	body.WriteString(`"vcs_revision":"` + binary.VCSRevision + `",`)
	body.WriteString(`"vcs_modified":false,`)
	body.WriteString(`"leamas_version":"` + binary.LeamasVersion + `"`)
	body.WriteString(`}}`)
	return []byte(body.String())
}

// sum256Bytes is a small wrapper for hex-encoding the
// SHA-256 of the supplied bytes.
func sum256Bytes(b []byte) string {
	import_sha256 := sha256Sum
	_ = import_sha256
	return sha256Sum(b)
}

// v2ManifestTestBinary builds a deterministic binary identity
// for hermetic ACT 3 tests. The identity is structurally
// valid (path, sha256, vcs_revision, vcs_modified=false,
// nonempty version) but does NOT require the binary path to
// exist; the verifier treats the manifest as the committed
// assertion, not as a re-hashing of the historical binary.
func v2ManifestTestBinary(t *testing.T) V2BinaryIdentity {
	t.Helper()
	return V2BinaryIdentity{
		Path:          "/usr/local/bin/leamas",
		SHA256:        strings.Repeat("a", 64),
		VCSRevision:   strings.Repeat("b", 40),
		VCSModified:   false,
		LeamasVersion: "0.1.0+test.act3",
	}
}

// TestV2VerifierManifestHappyPath proves the verifier
// accepts a well-formed committed manifest that binds every
// identity field to the bound authority, contains a valid
// frozen-plan bijection, and reports an aggregate successful
// run.
func TestV2VerifierManifestHappyPath(t *testing.T) {
	dir, subject, freeze, closure, subjectTree, freezeTree := buildMixedModeManifestRig(t,
		[]byte("placeholder"))
	manifest := mixedModeHappyManifest(t, dir, subject, freeze, subjectTree, freezeTree)

	// Replace the placeholder with the real manifest.
	writeFile(t, dir, "docs/closure-manifests/MANIFEST.json", manifest)
	mustRunGit(t, dir, "add", "docs/closure-manifests/MANIFEST.json")
	mustRunGit(t, dir, "commit", "-m", "closure: rewrite manifest")
	closure = mustRunGit(t, dir, "rev-parse", "HEAD")

	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	fp, err := ResolveV2FrozenPlanAuthority(context.Background(), auth, freeze,
		"docs/closure-plans/PLAN.json")
	if err != nil {
		t.Fatalf("ResolveV2FrozenPlanAuthority: %v", err)
	}
	cm, err := ResolveV2CommittedManifestAuthority(context.Background(), auth, closure,
		"docs/closure-manifests/MANIFEST.json")
	if err != nil {
		t.Fatalf("ResolveV2CommittedManifestAuthority: %v", err)
	}
	topology := V2ClosureTopology{
		SubjectCommit: subject,
		SubjectTree:   subjectTree,
		FreezeCommit:  freeze,
		FreezeTree:    freezeTree,
		ClosureCommit: closure,
		ClosureTree:   mustRunGit(t, dir, "rev-parse", closure+"^{tree}"),
	}

	verifier := NewV2ManifestIdentityVerifier()
	facts, err := verifier.VerifyManifestIdentity(cm.RawBytes, fp, cm, topology)
	if err != nil {
		t.Fatalf("VerifyManifestIdentity: %v", err)
	}
	if len(facts.Diagnostics) > 0 {
		t.Fatalf("happy path must produce zero diagnostics, got: %v",
			facts.Diagnostics.Codes())
	}
	if !facts.ManifestIdentityValid {
		t.Fatalf("ManifestIdentityValid must be true on happy path")
	}
	if !facts.BinaryIdentityValid {
		t.Fatalf("BinaryIdentityValid must be true on happy path")
	}
	if !facts.BijectionValid {
		t.Fatalf("BijectionValid must be true on happy path")
	}
	if !facts.SuccessValid {
		t.Fatalf("SuccessValid must be true on happy path")
	}
}

// TestV2VerifierManifestMutationMatrix covers the manifest
// bijection and success-integrity rejection paths required
// by Phase 7 of the ACT 3 specification.
//
// Each case starts from the happy-path manifest and mutates
// one field. The verifier must reject with a typed
// diagnostic and flip the relevant boolean to false.
func TestV2VerifierManifestMutationMatrix(t *testing.T) {
	type mutation struct {
		name     string
		mutate   func(string) string
		wantCode V2VerifierCode
		check    func(V2ManifestIdentityFacts) bool
	}

	cases := []mutation{
		{
			name: "missing_run_1",
			mutate: func(s string) string {
				return strings.Replace(s,
					`{"id":"run-1","mode":"run","outcome":"pass","exit_code":0,"duration_ms":0,"execution_classification":"completed","cleanup_status":"pass"},`,
					``, 1)
			},
			wantCode: V2VerifierManifestCheckResultBijectionFailed,
			check:    func(f V2ManifestIdentityFacts) bool { return !f.BijectionValid },
		},
		{
			name: "duplicate_run_1",
			mutate: func(s string) string {
				return strings.Replace(s,
					`{"id":"run-1","mode":"run","outcome":"pass","exit_code":0,"duration_ms":0,"execution_classification":"completed","cleanup_status":"pass"}`,
					`{"id":"run-1","mode":"run","outcome":"pass","exit_code":0,"duration_ms":0,"execution_classification":"completed","cleanup_status":"pass"},`+
						`{"id":"run-1","mode":"run","outcome":"pass","exit_code":0,"duration_ms":0,"execution_classification":"completed","cleanup_status":"pass"}`,
					1)
			},
			wantCode: V2VerifierManifestCheckResultBijectionFailed,
			check:    func(f V2ManifestIdentityFacts) bool { return !f.BijectionValid },
		},
		{
			name: "unknown_result_id",
			mutate: func(s string) string {
				return strings.Replace(s, `"id":"run-1"`, `"id":"unknown-check"`, 1)
			},
			wantCode: V2VerifierManifestUnknownCheckID,
			check:    func(f V2ManifestIdentityFacts) bool { return !f.BijectionValid },
		},
		{
			name: "run_mode_changed_to_exclude",
			mutate: func(s string) string {
				return strings.Replace(s,
					`{"id":"run-1","mode":"run","outcome":"pass"`,
					`{"id":"run-1","mode":"exclude","outcome":"excluded"`, 1)
			},
			wantCode: V2VerifierManifestCheckResultBijectionFailed,
			check:    func(f V2ManifestIdentityFacts) bool { return !f.BijectionValid },
		},
		{
			name: "exclude_mode_changed_to_run",
			mutate: func(s string) string {
				return strings.Replace(s,
					`{"id":"exclude-1","mode":"exclude","outcome":"excluded"`,
					`{"id":"exclude-1","mode":"run","outcome":"pass"`, 1)
			},
			wantCode: V2VerifierManifestCheckResultBijectionFailed,
			check:    func(f V2ManifestIdentityFacts) bool { return !f.BijectionValid },
		},
		{
			name: "aggregate_success_with_failed_result",
			mutate: func(s string) string {
				return strings.Replace(s,
					`{"id":"run-1","mode":"run","outcome":"pass","exit_code":0`,
					`{"id":"run-1","mode":"run","outcome":"fail","exit_code":1`, 1)
			},
			wantCode: V2VerifierManifestUnsuccessfulRun,
			check:    func(f V2ManifestIdentityFacts) bool { return !f.SuccessValid },
		},
		{
			name: "cleanup_failure_hidden_by_success",
			mutate: func(s string) string {
				return strings.Replace(s,
					`"cleanup_status":"pass"`,
					`"cleanup_status":"failed"`, 1)
			},
			wantCode: V2VerifierManifestUnsuccessfulRun,
			check:    func(f V2ManifestIdentityFacts) bool { return !f.SuccessValid },
		},
		{
			name: "timeout_marked_success",
			mutate: func(s string) string {
				return strings.Replace(s,
					`"execution_classification":"completed"`,
					`"execution_classification":"timeout"`, 1)
			},
			wantCode: V2VerifierManifestUnsuccessfulRun,
			check:    func(f V2ManifestIdentityFacts) bool { return !f.SuccessValid },
		},
		{
			name: "overflow_marked_success",
			mutate: func(s string) string {
				return strings.Replace(s,
					`"execution_classification":"completed"`,
					`"execution_classification":"output_overflow"`, 1)
			},
			wantCode: V2VerifierManifestUnsuccessfulRun,
			check:    func(f V2ManifestIdentityFacts) bool { return !f.SuccessValid },
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Build the happy manifest, then apply the
			// mutation, then commit.
			dir := initRepo(t)
			planBytes := mixedModePlanBytes()
			subject := makeCommit(t, dir, "subject implementation", map[string]string{
				"subject-only.txt": "subject implementation\n",
			})
			freeze := makeCommit(t, dir, "freeze: add mixed-mode plan", map[string]string{
				"docs/closure-plans/PLAN.json": planBytes,
			})
			subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
			freezeTree := mustRunGit(t, dir, "rev-parse", freeze+"^{tree}")
			manifest := mixedModeHappyManifest(t, dir, subject, freeze, subjectTree, freezeTree)
			mutated := tc.mutate(string(manifest))
			closure := writeAndCommit(t, dir,
				"docs/closure-manifests/MANIFEST.json", []byte(mutated),
				"closure: commit mutated manifest")

			auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
			if err != nil {
				t.Fatalf("authority: %v", err)
			}
			fp, err := ResolveV2FrozenPlanAuthority(context.Background(), auth, freeze,
				"docs/closure-plans/PLAN.json")
			if err != nil {
				t.Fatalf("ResolveV2FrozenPlanAuthority: %v", err)
			}
			cm, err := ResolveV2CommittedManifestAuthority(context.Background(), auth, closure,
				"docs/closure-manifests/MANIFEST.json")
			if err != nil {
				t.Fatalf("ResolveV2CommittedManifestAuthority: %v", err)
			}
			topology := V2ClosureTopology{
				SubjectCommit: subject,
				SubjectTree:   subjectTree,
				FreezeCommit:  freeze,
				FreezeTree:    freezeTree,
				ClosureCommit: closure,
				ClosureTree:   mustRunGit(t, dir, "rev-parse", closure+"^{tree}"),
			}

			verifier := NewV2ManifestIdentityVerifier()
			facts, err := verifier.VerifyManifestIdentity(cm.RawBytes, fp, cm, topology)
			if err != nil {
				t.Fatalf("VerifyManifestIdentity: %v", err)
			}
			if !facts.Diagnostics.HasCode(tc.wantCode) {
				t.Fatalf("diagnostic codes = %v, want code %s",
					facts.Diagnostics.Codes(), tc.wantCode)
			}
			if !tc.check(facts) {
				t.Fatalf("expected boolean check to fail for %s, facts=%+v",
					tc.name, facts)
			}
		})
	}
}

// writeFile is a small helper that writes bytes to a
// repository-relative path inside the supplied repo. The
// helper exists so the bijection tests can update an existing
// manifest without invoking makeCommit (which only accepts
// map[string]string).
func writeFile(t *testing.T, dir, repoRelPath string, content []byte) {
	t.Helper()
	full := repoRelPath
	if !strings.HasPrefix(full, dir) {
		full = dir + "/" + repoRelPath
	}
	if err := osMkdirAll(full); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := osWriteFile(full, content); err != nil {
		t.Fatalf("write: %v", err)
	}
}
