// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_manifest.go provides the validated v2
// manifest constructor. NewV2Manifest enforces the manifest
// invariants before serialisation so callers cannot
// accidentally publish an incomplete manifest.
//
// Splitting this from closure_protocol_v2.go keeps the file
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// V2ManifestBuild is the validated set of inputs the runner
// passes to NewV2Manifest. Every field is required; the
// constructor rejects incomplete or inconsistent values.
type V2ManifestBuild struct {
	ClosureProtocolVersion ClosureProtocolVersion
	PlanContractVersion    PlanContractVersion
	SubjectCommit          string
	SubjectTree            string
	FreezeCommit           string
	FreezeTree             string
	PlanPath               string
	PlanBlob               string
	PlanSHA256             string
	PlanBytes              []byte
	ExecutionTree          string
	CallerHead             string
	BinaryIdentity         V2BinaryIdentity
	CheckResults           []V2CheckResult
}

// NewV2Manifest validates the supplied build inputs and
// constructs a V2Manifest. The constructor enforces:
//
//   - execution_tree == subject_tree
//   - plan SHA-256 matches the frozen plan bytes
//   - all required identity fields are well-formed OIDs
//
// On failure a typed V2Error is returned. On success the
// returned V2Manifest is ready for deterministic rendering.
func NewV2Manifest(b V2ManifestBuild) (V2Manifest, error) {
	if !b.ClosureProtocolVersion.IsSupported() {
		return V2Manifest{}, NewV2ErrorWith(V2CodeUnsupportedClosureProtocolVersion,
			fmt.Sprintf("closure protocol version %q is not supported", string(b.ClosureProtocolVersion)),
			"closure_protocol_version", "")
	}
	if !b.PlanContractVersion.IsSupported() {
		return V2Manifest{}, NewV2ErrorWith(V2CodeUnsupportedPlanContractVersion,
			fmt.Sprintf("plan contract version %d is not supported", int(b.PlanContractVersion)),
			"plan_contract_version", "")
	}
	if b.SubjectCommit == "" || b.SubjectTree == "" {
		return V2Manifest{}, NewV2ErrorWith(V2CodeRequestIncomplete,
			"subject commit and tree are required", "subject", "")
	}
	if b.FreezeCommit == "" || b.FreezeTree == "" {
		return V2Manifest{}, NewV2ErrorWith(V2CodeRequestIncomplete,
			"freeze commit and tree are required", "freeze", "")
	}
	if b.PlanPath == "" || b.PlanBlob == "" || b.PlanSHA256 == "" {
		return V2Manifest{}, NewV2ErrorWith(V2CodeRequestIncomplete,
			"plan path, blob, and SHA-256 are required", "plan", "")
	}
	if b.ExecutionTree == "" {
		return V2Manifest{}, NewV2ErrorWith(V2CodeExecutionTreeMismatch,
			"execution tree is required", "execution_tree", "")
	}
	if b.ExecutionTree != b.SubjectTree {
		return V2Manifest{}, NewV2ErrorWith(V2CodeExecutionTreeMismatch,
			fmt.Sprintf("execution_tree %s must equal subject_tree %s", b.ExecutionTree, b.SubjectTree),
			"execution_tree", b.ExecutionTree)
	}
	if len(b.PlanBytes) == 0 {
		return V2Manifest{}, NewV2ErrorWith(V2CodeRequestIncomplete,
			"plan bytes are required to validate SHA-256", "plan_bytes", "")
	}
	sum := sha256.Sum256(b.PlanBytes)
	actualSHA := hex.EncodeToString(sum[:])
	if actualSHA != b.PlanSHA256 {
		return V2Manifest{}, NewV2ErrorWith(V2CodeFrozenPlanNotBlob,
			fmt.Sprintf("plan bytes SHA-256 %s does not match declared %s",
				actualSHA, b.PlanSHA256),
			"plan_sha256", actualSHA)
	}
	if b.CallerHead == "" {
		return V2Manifest{}, NewV2ErrorWith(V2CodeRequestIncomplete,
			"caller_head is required", "caller_head", "")
	}
	checks := append([]V2CheckResult(nil), b.CheckResults...)
	return V2Manifest{
		ClosureProtocolVersion: b.ClosureProtocolVersion,
		PlanContractVersion:    int(b.PlanContractVersion),
		SubjectCommit:          b.SubjectCommit,
		SubjectTree:            b.SubjectTree,
		FreezeCommit:           b.FreezeCommit,
		FreezeTree:             b.FreezeTree,
		PlanPath:               b.PlanPath,
		PlanBlob:               b.PlanBlob,
		PlanSHA256:             b.PlanSHA256,
		ExecutionTree:          b.ExecutionTree,
		CheckResults:           checks,
		LeamasBinaryIdentity:   b.BinaryIdentity,
		CallerHead:             b.CallerHead,
	}, nil
}

// V2ManifestJSON renders the manifest as deterministic JSON.
// The function delegates to encoding/json which preserves
// the declared struct field order.
func V2ManifestRender(m V2Manifest) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// AtomicWriteV2Manifest writes the supplied manifest bytes
// to the given absolute path using a temp-file rename so a
// crash mid-write cannot leave a partial manifest on disk.
//
// On failure a typed V2Error carrying V2CodeManifestWriteFailed
// is returned.
func AtomicWriteV2Manifest(absPath string, data []byte) error {
	if absPath == "" {
		return NewV2ErrorWith(V2CodeManifestWriteFailed,
			"manifest output path is empty", "manifest_output", "")
	}
	if !filepath.IsAbs(absPath) {
		return NewV2ErrorWith(V2CodeManifestWriteFailed,
			fmt.Sprintf("manifest output %q must be absolute", absPath),
			"manifest_output", absPath)
	}
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return NewV2ErrorWith(V2CodeManifestWriteFailed,
			fmt.Sprintf("mkdir %s: %s", dir, err.Error()),
			"manifest_output", err.Error())
	}
	tmp, err := os.CreateTemp(dir, ".v2manifest.*.tmp")
	if err != nil {
		return NewV2ErrorWith(V2CodeManifestWriteFailed,
			fmt.Sprintf("create temp manifest: %s", err.Error()),
			"manifest_output", err.Error())
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return NewV2ErrorWith(V2CodeManifestWriteFailed,
			fmt.Sprintf("write temp manifest: %s", err.Error()),
			"manifest_output", err.Error())
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return NewV2ErrorWith(V2CodeManifestWriteFailed,
			fmt.Sprintf("sync temp manifest: %s", err.Error()),
			"manifest_output", err.Error())
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return NewV2ErrorWith(V2CodeManifestWriteFailed,
			fmt.Sprintf("close temp manifest: %s", err.Error()),
			"manifest_output", err.Error())
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		cleanup()
		return NewV2ErrorWith(V2CodeManifestWriteFailed,
			fmt.Sprintf("rename temp manifest to %s: %s", absPath, err.Error()),
			"manifest_output", err.Error())
	}
	return nil
}
