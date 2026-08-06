// SPDX-License-Identifier: Apache-2.0

package closure

// evidence_verifier_r2b.go provides the R2B committed
// closure-manifest evidence verifier required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2B.
//
// The verifier loads a committed closure manifest, validates
// every hash-shaped field, and confirms the files_changed
// SHA-256 digests match the exact subject-commit bytes. Any
// descriptive text, abbreviated hash, or synthetic
// placeholder is rejected with a typed diagnostic.
//
// Splitting this from the closure-manifest production code
// keeps the verifier close to the test corpus while
// preserving the LLM-friendly 400-line threshold.

// R2C-R4 OBJECT-FORMAT POLICY:
//
// Leamas currently supports only the SHA-1 object format. All
// hash-shaped fields in the closure manifest must therefore
// have the lengths of the SHA-1 algorithm family:
//
//   - Git commit OIDs:        40 lowercase hex chars
//   - Git tree OIDs:           40 lowercase hex chars
//   - Git blob OIDs:           40 lowercase hex chars
//   - SHA-256 file digests:    64 lowercase hex chars
//
// Future object formats (e.g. SHA-256, sha256d) are out of
// scope for R2C-R4. The verifier rejects any field whose
// length does not match these expectations so a repository
// whose `git rev-parse --show-object-format` is not `sha1`
// produces a typed diagnostic rather than a silent corruption.
//
// R2C-R4 enforces the policy in-process: the verifier calls
// EnforceSHA1ObjectFormat, which in turn calls
// GitObjectResolver.ObjectFormat, which executes
// `git rev-parse --show-object-format` in the resolver's
// bound repository root. The format check happens BEFORE any
// OID validation so a sha256 repository is rejected with
// V2CodeUnsupportedObjectFormat before any OID-length check.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// evidenceSHA256Pattern enforces 64 lowercase hexadecimal characters.
var evidenceSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// gitOIDPattern enforces 40 lowercase hexadecimal characters.
var gitOIDPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// GitObjectResolver loads Git objects by OID. Implementations
// include the production RealGit wrapper and the test fakes
// that return the bytes of a synthetic repository.
type GitObjectResolver interface {
	// CatFile returns the raw bytes of the supplied Git object.
	// For commits it returns the commit object bytes; for
	// trees it returns the tree object bytes; for blobs it
	// returns the blob content bytes.
	CatFile(oid string) ([]byte, error)
	// ObjectFormat returns the Git object storage format
	// reported by `git rev-parse --show-object-format`.
	// Production implementations MUST execute the real git
	// command and propagate every observation failure.
	// Test fakes return a hard-coded format ("sha1" by
	// default; other values are used to drive the format
	// matrix tests).
	ObjectFormat() (string, error)
}

// EvidenceVerifierOptions configures the evidence verifier.
// All paths and OIDs are required.
type EvidenceVerifierOptions struct {
	// ManifestPath is the on-disk path of the closure manifest
	// JSON document to verify.
	ManifestPath string
	// SubjectCommit is the Git commit OID whose bytes define
	// the authoritative files_changed SHA-256 digests.
	SubjectCommit string
	// Resolver resolves Git objects (commits, trees, blobs)
	// for the verifier. Production uses RealGit{}; tests
	// inject a fake that returns synthetic bytes.
	Resolver GitObjectResolver
}

// evidenceManifest is the typed shape the verifier needs.
// Only the fields exercised by the R2B checks are decoded;
// additional fields are preserved via raw access in callers.
type evidenceManifest struct {
	ContractVersion int              `json:"contract_version"`
	ActID           string           `json:"act_id"`
	Subject         evidenceSubject  `json:"subject"`
	Runner          evidenceRunner   `json:"runner"`
	FilesChanged    []evidenceFile   `json:"files_changed"`
	Dogfood         *evidenceDogfood `json:"dogfood,omitempty"`
}

type evidenceSubject struct {
	CommitOID string `json:"commit_oid"`
	TreeOID   string `json:"tree_oid"`
}

type evidenceRunner struct {
	LeamasVersion string `json:"leamas_version"`
	BinarySHA256  string `json:"binary_sha256"`
	VCSRevision   string `json:"vcs_revision"`
	VCSModified   bool   `json:"vcs_modified"`
}

type evidenceFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type evidenceDogfood struct {
	BinaryCommit  string `json:"binary_commit"`
	BinarySHA256  string `json:"binary_sha256"`
	BinaryVCSRev  string `json:"binary_vcs_revision"`
	SubjectCommit string `json:"subject_commit"`
	SubjectTree   string `json:"subject_tree"`
	FreezeCommit  string `json:"freeze_commit"`
	FreezeTree    string `json:"freeze_tree"`
	CallerHead    string `json:"caller_head"`
	ExecutionTree string `json:"execution_tree"`
	PlanBlob      string `json:"plan_blob"`
	PlanSHA256    string `json:"plan_sha256"`
}

// EvidenceVerifierDiagnostic is a typed diagnostic the
// verifier returns when it rejects a field.
type EvidenceVerifierDiagnostic struct {
	Field   string
	Message string
}

func (d EvidenceVerifierDiagnostic) Error() string {
	return fmt.Sprintf("evidence_verifier: field=%s message=%s", d.Field, d.Message)
}

// EvidenceVerifierResult is the verifier's verdict.
type EvidenceVerifierResult struct {
	OK          bool
	Diagnostics []EvidenceVerifierDiagnostic
}

// VerifyClosureManifestR2B reads the committed closure
// manifest at opts.ManifestPath and validates every
// hash-shaped field against the supplied subject commit's
// exact file bytes. The function returns an
// EvidenceVerifierResult whose OK field is true only when
// every required field is present, syntactically valid, and
// the files_changed digests match the subject-commit bytes.
//
// Required validations:
//
//   - subject.commit_oid is a 40-char lowercase Git OID
//   - subject.tree_oid   is a 40-char lowercase Git OID
//   - runner.binary_sha256 is a 64-char lowercase SHA-256
//   - runner.vcs_revision  is a 40-char lowercase Git OID
//   - every files_changed[i].sha256 is a 64-char lowercase
//     SHA-256 digest that equals SHA-256(exact subject
//     commit bytes for files_changed[i].path)
//   - dogfood fields (when present) are full Git OIDs / SHA-256
//   - dogfood.plan_blob     is a 40-char lowercase Git OID
//   - dogfood.plan_sha256   is a 64-char lowercase SHA-256
func VerifyClosureManifestR2B(opts EvidenceVerifierOptions) (EvidenceVerifierResult, error) {
	if opts.ManifestPath == "" {
		return EvidenceVerifierResult{}, errors.New("manifest path is required")
	}
	if opts.SubjectCommit == "" {
		return EvidenceVerifierResult{}, errors.New("subject commit is required")
	}
	if opts.Resolver == nil {
		return EvidenceVerifierResult{}, errors.New("git object resolver is required")
	}
	// R2C-R4 SHA-1 policy: programmatic object-format
	// enforcement runs BEFORE any OID validation. A
	// sha256 repository whose OIDs are 64 chars would
	// otherwise be rejected by the length check with a
	// misleading "not a 40-char OID" diagnostic; the
	// typed unsupported_object_format diagnostic is the
	// authoritative verdict.
	if v2err := EnforceSHA1ObjectFormat(opts.Resolver); v2err != nil {
		return EvidenceVerifierResult{}, v2err
	}
	commitBytes, err := opts.Resolver.CatFile(opts.SubjectCommit)
	if err != nil {
		return EvidenceVerifierResult{}, fmt.Errorf("read subject commit %s: %w", opts.SubjectCommit, err)
	}
	treeOID := extractCommitTreeOID(string(commitBytes))
	if treeOID == "" {
		return EvidenceVerifierResult{}, fmt.Errorf("subject commit %s has no tree header", opts.SubjectCommit)
	}
	treeBytes, err := opts.Resolver.CatFile(treeOID)
	if err != nil {
		return EvidenceVerifierResult{}, fmt.Errorf("read subject tree %s: %w", treeOID, err)
	}
	treeEntries, err := parseGitTree(treeBytes, opts.Resolver)
	if err != nil {
		return EvidenceVerifierResult{}, fmt.Errorf("parse subject tree %s: %w", treeOID, err)
	}
	doc, err := os.ReadFile(opts.ManifestPath)
	if err != nil {
		return EvidenceVerifierResult{}, fmt.Errorf("read manifest: %w", err)
	}
	var m evidenceManifest
	if err := json.Unmarshal(doc, &m); err != nil {
		return EvidenceVerifierResult{}, fmt.Errorf("decode manifest: %w", err)
	}
	var diags []EvidenceVerifierDiagnostic
	diags = append(diags, evidenceValidateOID("subject.commit_oid", m.Subject.CommitOID)...)
	diags = append(diags, evidenceValidateOID("subject.tree_oid", m.Subject.TreeOID)...)
	diags = append(diags, evidenceValidateSHA256("runner.binary_sha256", m.Runner.BinarySHA256)...)
	diags = append(diags, evidenceValidateOID("runner.vcs_revision", m.Runner.VCSRevision)...)
	for _, fc := range m.FilesChanged {
		diags = append(diags, validateFilesChangedEntry(treeEntries, fc)...)
	}
	if m.Dogfood != nil {
		diags = append(diags, validateDogfood(*m.Dogfood)...)
	}
	return EvidenceVerifierResult{
		OK:          len(diags) == 0,
		Diagnostics: diags,
	}, nil
}

// evidenceValidateOID returns a diagnostic when v is not a 40-char
// lowercase Git OID.
func evidenceValidateOID(field, v string) []EvidenceVerifierDiagnostic {
	if v == "" {
		return []EvidenceVerifierDiagnostic{{Field: field, Message: "missing OID"}}
	}
	if !gitOIDPattern.MatchString(v) {
		return []EvidenceVerifierDiagnostic{{
			Field:   field,
			Message: fmt.Sprintf("not a 40-char lowercase Git OID: %q", evidenceTruncate(v, 16)),
		}}
	}
	return nil
}

// evidenceValidateSHA256 returns a diagnostic when v is not a 64-char
// lowercase SHA-256 digest.
func evidenceValidateSHA256(field, v string) []EvidenceVerifierDiagnostic {
	if v == "" {
		return []EvidenceVerifierDiagnostic{{Field: field, Message: "missing SHA-256"}}
	}
	if !evidenceSHA256Pattern.MatchString(v) {
		return []EvidenceVerifierDiagnostic{{
			Field:   field,
			Message: fmt.Sprintf("not a 64-char lowercase SHA-256: %q", evidenceTruncate(v, 16)),
		}}
	}
	return nil
}

// validateFilesChangedEntry asserts the SHA-256 matches the
// bytes of path in treeEntries.
func validateFilesChangedEntry(treeEntries map[string][]byte, fc evidenceFile) []EvidenceVerifierDiagnostic {
	if fc.Path == "" {
		return []EvidenceVerifierDiagnostic{{Field: "files_changed[].path", Message: "missing path"}}
	}
	if !evidenceSHA256Pattern.MatchString(fc.SHA256) {
		return []EvidenceVerifierDiagnostic{{
			Field:   fmt.Sprintf("files_changed[%s].sha256", fc.Path),
			Message: fmt.Sprintf("not a 64-char lowercase SHA-256: %q", evidenceTruncate(fc.SHA256, 32)),
		}}
	}
	blob, ok := treeEntries[fc.Path]
	if !ok {
		return []EvidenceVerifierDiagnostic{{
			Field:   fmt.Sprintf("files_changed[%s]", fc.Path),
			Message: "path not present in subject tree",
		}}
	}
	sum := sha256.Sum256(blob)
	got := hex.EncodeToString(sum[:])
	if got != fc.SHA256 {
		return []EvidenceVerifierDiagnostic{{
			Field:   fmt.Sprintf("files_changed[%s].sha256", fc.Path),
			Message: fmt.Sprintf("subject SHA-256 %s does not match declared %s", got, fc.SHA256),
		}}
	}
	return nil
}

// validateDogfood asserts every required identity field is
// present and full-length.
func validateDogfood(d evidenceDogfood) []EvidenceVerifierDiagnostic {
	var diags []EvidenceVerifierDiagnostic
	diags = append(diags, evidenceValidateOID("dogfood.binary_commit", d.BinaryCommit)...)
	diags = append(diags, evidenceValidateSHA256("dogfood.binary_sha256", d.BinarySHA256)...)
	diags = append(diags, evidenceValidateOID("dogfood.binary_vcs_revision", d.BinaryVCSRev)...)
	diags = append(diags, evidenceValidateOID("dogfood.subject_commit", d.SubjectCommit)...)
	diags = append(diags, evidenceValidateOID("dogfood.subject_tree", d.SubjectTree)...)
	diags = append(diags, evidenceValidateOID("dogfood.freeze_commit", d.FreezeCommit)...)
	diags = append(diags, evidenceValidateOID("dogfood.freeze_tree", d.FreezeTree)...)
	diags = append(diags, evidenceValidateOID("dogfood.caller_head", d.CallerHead)...)
	diags = append(diags, evidenceValidateOID("dogfood.execution_tree", d.ExecutionTree)...)
	// plan_blob and plan_sha256 are optional. An
	// authority-only ACT may not bind a frozen plan; the
	// R1 close report documents that path. The validator
	// only complains when the field is present but
	// malformed.
	if d.PlanBlob != "" {
		diags = append(diags, evidenceValidateOID("dogfood.plan_blob", d.PlanBlob)...)
	}
	if d.PlanSHA256 != "" {
		diags = append(diags, evidenceValidateSHA256("dogfood.plan_sha256", d.PlanSHA256)...)
	}
	return diags
}

// truncate shortens s to max chars with an ellipsis for
// diagnostic messages.
func evidenceTruncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// extractCommitTreeOID parses the first `tree <oid>` line of
// the cat-file commit output.
func extractCommitTreeOID(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "tree ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "tree "))
		}
	}
	return ""
}

// parseGitTree parses `git cat-file -p <tree>` output and
// returns a map from path to blob bytes. Tree entries are
// recursively walked so nested directory paths appear with
// their full name. The parsed format is:
//
//	<mode> <type> <hex-oid>\t<name>\n
//
// for each tree entry, with no final newline.
func parseGitTree(raw []byte, resolver GitObjectResolver) (map[string][]byte, error) {
	return walkTree("", raw, resolver)
}

// walkTree recursively walks the parsed tree output,
// prepending parent to each entry's name and resolving
// blob bytes via the supplied resolver.
func walkTree(prefix string, raw []byte, resolver GitObjectResolver) (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, line := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") {
		tab := strings.IndexByte(line, '\t')
		if tab < 0 {
			continue
		}
		meta := strings.SplitN(line[:tab], " ", 3)
		if len(meta) != 3 {
			continue
		}
		oid := meta[2]
		entryType := meta[1]
		name := line[tab+1:]
		fullName := name
		if prefix != "" {
			fullName = prefix + "/" + name
		}
		if entryType == "tree" {
			subBytes, err := resolver.CatFile(oid)
			if err != nil {
				return nil, fmt.Errorf("read sub-tree %s: %w", oid, err)
			}
			sub, err := walkTree(fullName, subBytes, resolver)
			if err != nil {
				return nil, err
			}
			for k, v := range sub {
				out[k] = v
			}
			continue
		}
		blob, err := resolver.CatFile(oid)
		if err != nil {
			return nil, fmt.Errorf("read blob %s for %s: %w", oid, fullName, err)
		}
		out[fullName] = blob
	}
	return out, nil
}
