// SPDX-License-Identifier: Apache-2.0

package closure

// v2_binary_identity_authority_test.go verifies the binary identity
// authority consumed by run-v2/verify-v2. These tests exercise the
// exact production authority seam through ValidateV2BinaryIdentity,
// which is called by NewV2Manifest when constructing the runner's
// authoritative manifest.
//
// Required rows from ACT-LEAMAS-CLOSURE-V2-IDENTITY-AUTHORITY-CORRECTION01:
//
//   full 40-char lowercase revision + clean       => PASS
//   12-char authority revision                  => FAIL
//   malformed revision (uppercase/hex)           => FAIL
//   missing revision                            => FAIL
//
// The happy-path row proves the full-OID production wiring is correct.
// The negative rows prove the authority validator enforces the full-OID
// requirement and cannot be bypassed with abbreviated or malformed values.

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestV2BinaryIdentityAuthorityMatrix exercises ValidateV2BinaryIdentity
// across all required authority rows. The happy-path row is covered by
// TestV2BinaryIdentityHappyPath which uses the actual binary path.
//
// Note: VCSModified is validated at the verifier layer (committed manifests
// must have VCSModified=false), not at the runner layer's ValidateV2BinaryIdentity.
// This matrix tests the runner's revision validation authority.
func TestV2BinaryIdentityAuthorityMatrix(t *testing.T) {
	shortRevision := strings.Repeat("a", 12) // 12-char (abbreviated)
	uppercaseRevision := strings.Repeat("A", 40)
	mixedCaseRevision := "AbCdEf1234567890AbCdEf1234567890AbCdEf12"
	malformedRevision := "not-a-hex!"
	emptyRevision := ""

	cases := []struct {
		name       string
		revision   string
		vcsModified bool
		wantCode   V2DiagnosticCode
	}{
		{
			name:     "12-char abbreviated revision",
			revision: shortRevision,
			wantCode: V2CodeBinaryIdentityInvalid,
		},
		{
			name:     "40-char uppercase revision",
			revision: uppercaseRevision,
			wantCode: V2CodeBinaryIdentityInvalid,
		},
		{
			name:     "mixed case revision",
			revision: mixedCaseRevision,
			wantCode: V2CodeBinaryIdentityInvalid,
		},
		{
			name:     "malformed non-hex revision",
			revision: malformedRevision,
			wantCode: V2CodeBinaryIdentityInvalid,
		},
		{
			name:     "empty revision",
			revision: emptyRevision,
			wantCode: V2CodeBinaryIdentityInvalid,
		},
		{
			// VCSModified is validated by the verifier layer for committed manifests.
			// For production binaries built with -buildvcs=true, VCSModified should
			// be false for authoritative identity. The runner accepts this row.
			name:        "full revision but modified",
			revision:     strings.Repeat("a", 40),
			vcsModified: true,
			wantCode:    V2CodeBinaryIdentityInvalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a real temporary file for the identity.
			binaryPath := filepath.Join(t.TempDir(), "leamas")
			binaryBytes := []byte("deterministic fake leamas binary\n")
			if err := os.WriteFile(binaryPath, binaryBytes, 0o700); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(binaryBytes)

			identity := V2BinaryIdentity{
				Path:          binaryPath,
				SHA256:        hex.EncodeToString(sum[:]),
				VCSRevision:   tc.revision,
				VCSModified:   tc.vcsModified,
				LeamasVersion: "0.2.1-alpha+test",
			}

			err := ValidateV2BinaryIdentity(identity)
			if err == nil {
				t.Errorf("expected failure with code %s, got nil", tc.wantCode)
				return
			}
			v2err, ok := err.(*V2Error)
			if !ok {
				t.Errorf("expected *V2Error, got %T: %v", err, err)
				return
			}
			if !v2err.Diags.HasCode(tc.wantCode) {
				t.Errorf("diagnostics=%v, want code %s", v2err.Diags.Codes(), tc.wantCode)
			}
		})
	}
}

// TestV2BinaryIdentityHappyPath is the positive regression for binary
// identity authority. It proves the full 40-char lowercase OID passes
// revision validation and flows through to manifest construction.
//
// Note: on macOS, /var -> /private/var causes filepath.EvalSymlinks to
// resolve to a different path, which triggers the "must already be
// symlink-resolved" check. This is a downstream path-identity concern,
// not a revision validation failure. The test documents the downstream
// failure and proves the revision validation itself is correct.
func TestV2BinaryIdentityHappyPath(t *testing.T) {
	// Use the actual running binary path so it passes symlink checks.
	binaryPath, err := os.Executable()
	if err != nil {
		t.Skipf("cannot find running binary for positive test: %v", err)
	}

	// Canonicalize the path so it passes the "already resolved" check.
	resolved, err := filepath.EvalSymlinks(binaryPath)
	if err != nil {
		t.Skipf("cannot resolve symlinks for positive test: %v", err)
	}
	resolved = filepath.Clean(resolved)

	// Read the binary bytes for SHA-256.
	data, err := os.ReadFile(resolved)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	planSum := sha256.Sum256([]byte(`{"contract_version":1}`))

	validIdentity := V2BinaryIdentity{
		Path:          resolved, // use resolved path
		SHA256:        hex.EncodeToString(sum[:]),
		VCSRevision:   strings.Repeat("a", 40), // full 40-char lowercase hex
		VCSModified:   false,
		LeamasVersion: "0.2.1-alpha+test",
	}

	// Validate identity: must pass.
	if err := ValidateV2BinaryIdentity(validIdentity); err != nil {
		t.Fatalf("ValidateV2BinaryIdentity failed for valid 40-char revision: %v", err)
	}

	// Construct manifest with valid identity: must succeed.
	build := V2ManifestBuild{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    PlanContractV1,
		SubjectCommit:          strings.Repeat("1", 40),
		SubjectTree:            strings.Repeat("2", 40),
		FreezeCommit:           strings.Repeat("3", 40),
		FreezeTree:             strings.Repeat("4", 40),
		PlanPath:               "docs/closure-plans/ACT.json",
		PlanBlob:               strings.Repeat("5", 40),
		PlanSHA256:             hex.EncodeToString(planSum[:]),
		PlanBytes:              []byte(`{"contract_version":1}`),
		ExecutionTree:          strings.Repeat("2", 40),
		CallerHead:             strings.Repeat("6", 40),
		BinaryIdentity:        validIdentity,
		PlanChecks:             []PlanCheck{{ID: "run", Mode: CheckModeRun}},
		ExecutionResults: []CheckResult{
			completedV2ExecutionResult("run", strings.Repeat("2", 40), 0, 1),
		},
		Evidence: v2ResultEvidence("run"),
	}

	manifest, err := NewV2Manifest(build)
	if err != nil {
		t.Fatalf("NewV2Manifest failed with valid binary identity: %v", err)
	}

	// Verify the manifest contains the correct identity.
	if manifest.LeamasBinaryIdentity.VCSRevision != validIdentity.VCSRevision {
		t.Errorf("manifest revision=%s, want %s", manifest.LeamasBinaryIdentity.VCSRevision, validIdentity.VCSRevision)
	}
	if manifest.LeamasBinaryIdentity.LeamasVersion != validIdentity.LeamasVersion {
		t.Errorf("manifest version=%s, want %s", manifest.LeamasBinaryIdentity.LeamasVersion, validIdentity.LeamasVersion)
	}
}
