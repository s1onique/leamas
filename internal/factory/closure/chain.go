package closure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Placeholder patterns for rejection.
var placeholderPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(a1b2c3d4e5f6|deadbeef)$`),
	regexp.MustCompile(`(?i)^(TODO|TBD|UNKNOWN|RUNNING|TO BE RECORDED)$`),
	regexp.MustCompile(`(?i)<COMMIT>|<TREE>|<HASH>|<OID>`),
	regexp.MustCompile(`\(SEE GIT REV-PARSE\)`),
}

// OID patterns for different Git object formats.
var (
	sha1Pattern   = regexp.MustCompile(`^[0-9a-f]{40}$`)
	sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// ObjectFormat represents Git object storage format.
type ObjectFormat string

const (
	ObjectFormatSHA1    ObjectFormat = "sha1"
	ObjectFormatSHA256  ObjectFormat = "sha256"
	ObjectFormatUnknown ObjectFormat = "unknown"
)

// ValidationError represents a chain validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// DetectObjectFormat detects Git object format from an OID length.
func DetectObjectFormat(oid string) ObjectFormat {
	switch len(oid) {
	case 40:
		return ObjectFormatSHA1
	case 64:
		return ObjectFormatSHA256
	default:
		return ObjectFormatUnknown
	}
}

// RejectPlaceholder checks if a value is a placeholder.
func RejectPlaceholder(fieldName, value string) error {
	if value == "" {
		return nil
	}
	normalized := strings.TrimSpace(strings.ToUpper(value))
	for _, pattern := range placeholderPatterns {
		if pattern.MatchString(normalized) || pattern.MatchString(value) {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("rejected placeholder value %q", value),
			}
		}
	}
	return nil
}

// ValidateOID checks if an OID is valid (40 hex chars for SHA-1).
func ValidateOID(fieldName, value string) error {
	return ValidateOIDWithFormat(fieldName, value, ObjectFormatSHA1)
}

// ValidateOIDWithFormat validates OID for the given Git object format.
func ValidateOIDWithFormat(fieldName, value string, format ObjectFormat) error {
	if err := RejectPlaceholder(fieldName, value); err != nil {
		return err
	}
	var validPattern *regexp.Regexp
	switch format {
	case ObjectFormatSHA1:
		validPattern = sha1Pattern
	case ObjectFormatSHA256:
		validPattern = sha256Pattern
	default:
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("unknown object format %q", format),
		}
	}
	if !validPattern.MatchString(value) {
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("invalid OID format %q for %s", value, format),
		}
	}
	return nil
}

// Attestation represents a post-closure attestation.
type Attestation struct {
	AttestationVersion int                   `json:"attestation_version"`
	ActID              string                `json:"act_id"`
	ProtocolVersion    string                `json:"protocol_version"`
	AttestedAt         string                `json:"attested_at"`
	Description        string                `json:"description"`
	ClosureReference   ClosureReference      `json:"closure_reference"`
	TagIdentity        TagIdentity           `json:"tag_identity"`
	FreezeReference    FreezeReference       `json:"freeze_reference"`
	SubjectReference   SubjectReference      `json:"subject_reference"`
	ChainValidity      ChainValidity         `json:"chain_validity"`
	NoSelfReference    NoSelfReferenceInPlan `json:"no_self_reference_in_plan"`
}

// ClosureReference records closure commit and tree.
type ClosureReference struct {
	ClosureCommit string `json:"closure_commit"`
	ClosureTree   string `json:"closure_tree"`
}

// TagIdentity records tag information.
type TagIdentity struct {
	TagName      string `json:"tag_name"`
	TagObjectOID string `json:"tag_object_oid"`
	TagType      string `json:"tag_type"`
	PeeledTarget string `json:"peeled_target"`
}

// FreezeReference records freeze commit and tree.
type FreezeReference struct {
	FreezeCommit string `json:"freeze_commit"`
	FreezeTree   string `json:"freeze_tree"`
	PlanBlobOID  string `json:"plan_blob_oid"`
}

// SubjectReference records subject commit and tree.
type SubjectReference struct {
	SubjectCommit string `json:"subject_commit"`
	SubjectTree   string `json:"subject_tree"`
}

// ChainValidity records chain validation results.
type ChainValidity struct {
	FNotEqualS                 bool `json:"F_not_equal_S"`
	FIsAncestorOfS             bool `json:"F_is_ancestor_of_S"`
	PlanBytesFEqualsPlanBytesS bool `json:"plan_bytes_F_equals_plan_bytes_S"`
	ManifestFMatchesActualF    bool `json:"manifest.F_matches_actual_F"`
	ManifestFTreeMatchesFTree  bool `json:"manifest.F_TREE_matches_F_tree"`
	ManifestSMatchesActualS    bool `json:"manifest.S_matches_actual_S"`
	ManifestSTreeMatchesSTree  bool `json:"manifest.S_TREE_matches_S_tree"`
	TagPeeledTargetMatchesC    bool `json:"tag_peeled_target_matches_C"`
}

// NoSelfReferenceInPlan records whether plan has self-references.
type NoSelfReferenceInPlan struct {
	PlanFreezeCommitInPlan  bool `json:"plan_freeze_commit_in_plan"`
	PlanFreezeTreeInPlan    bool `json:"plan_freeze_tree_in_plan"`
	PlanSubjectCommitInPlan bool `json:"plan_subject_commit_in_plan"`
	PlanSubjectTreeInPlan   bool `json:"plan_subject_tree_in_plan"`
	PlanClosureCommitInPlan bool `json:"plan_closure_commit_in_plan"`
	PlanClosureTreeInPlan   bool `json:"plan_closure_tree_in_plan"`
	PlanTagOIDInPlan        bool `json:"plan_tag_oid_in_plan"`
	PlanTagTargetInPlan     bool `json:"plan_tag_target_in_plan"`
}

// ChainValidationRequest contains parameters for chain validation.
type ChainValidationRequest struct {
	RepoRoot string
	Git      gitClient
	Freeze   string
	Subject  string
	Closure  string
	Tag      string
	PlanPath string
	Manifest *Manifest
}

// ChainValidationResult contains validation results.
type ChainValidationResult struct {
	Verdict                    string
	Errors                     []string
	FNotEqualS                 bool
	FIsAncestorOfS             bool
	SIsAncestorOfC             bool
	FIsAncestorOfC             bool
	PlanBytesFEqualsPlanBytesS bool
	ManifestFMatchesActualF    bool
	ManifestFTreeMatchesFTree  bool
	ManifestSMatchesActualS    bool
	ManifestSTreeMatchesSTree  bool
	TagIsAnnotated             bool
	TagObjectIsTag             bool
	TagPeeledTargetMatchesC    bool
	AllChecks                  []string
}

// CheckPlanNoSelfReference verifies plan does not contain self-referential identities.
func CheckPlanNoSelfReference(planPath string) (NoSelfReferenceInPlan, error) {
	data, err := os.ReadFile(planPath)
	if err != nil {
		return NoSelfReferenceInPlan{}, fmt.Errorf("read plan: %w", err)
	}

	var noSelf NoSelfReferenceInPlan
	content := string(data)

	selfRefFields := []string{
		"freeze_commit", "freeze_tree",
		"subject_commit", "subject_tree",
		"closure_commit", "closure_tree",
		"tag_oid", "tag_target",
	}

	for _, field := range selfRefFields {
		pattern := fmt.Sprintf(`"%s"\s*:\s*"[0-9a-f]{40}"`, field)
		matched, _ := regexp.MatchString(pattern, content)
		switch field {
		case "freeze_commit":
			noSelf.PlanFreezeCommitInPlan = matched
		case "freeze_tree":
			noSelf.PlanFreezeTreeInPlan = matched
		case "subject_commit":
			noSelf.PlanSubjectCommitInPlan = matched
		case "subject_tree":
			noSelf.PlanSubjectTreeInPlan = matched
		case "closure_commit":
			noSelf.PlanClosureCommitInPlan = matched
		case "closure_tree":
			noSelf.PlanClosureTreeInPlan = matched
		case "tag_oid":
			noSelf.PlanTagOIDInPlan = matched
		case "tag_target":
			noSelf.PlanTagTargetInPlan = matched
		}
	}

	return noSelf, nil
}

// PlanBytesAtCommit retrieves plan bytes at a specific commit.
func PlanBytesAtCommit(ctx context.Context, repoRoot, commit, planPath string) ([]byte, error) {
	git := RealGit{}
	result := git.Run(ctx, repoRoot, "cat-file", "blob", commit+":"+planPath)
	if result.ExitCode != 0 || result.Err != nil {
		return nil, fmt.Errorf("git cat-file blob %s:%s failed", commit, planPath)
	}
	return result.Stdout, nil
}

// ChainsEqual compares plan bytes at two commits.
func ChainsEqual(ctx context.Context, repoRoot, commit1, commit2, planPath string) (bool, error) {
	bytes1, err := PlanBytesAtCommit(ctx, repoRoot, commit1, planPath)
	if err != nil {
		return false, err
	}
	bytes2, err := PlanBytesAtCommit(ctx, repoRoot, commit2, planPath)
	if err != nil {
		return false, err
	}
	return bytes.Equal(bytes1, bytes2), nil
}

// DecodeAttestation decodes and validates an attestation.
func DecodeAttestation(data []byte) (Attestation, error) {
	var a Attestation
	if err := json.Unmarshal(data, &a); err != nil {
		return Attestation{}, fmt.Errorf("decode attestation: %w", err)
	}
	if err := ValidateAttestation(a); err != nil {
		return Attestation{}, err
	}
	return a, nil
}

// ValidateAttestation validates attestation structure and truth.
func ValidateAttestation(a Attestation) error {
	if a.AttestationVersion != 1 {
		return fmt.Errorf("unsupported attestation_version %d", a.AttestationVersion)
	}
	// Require annotated tags only
	if a.TagIdentity.TagType != "annotated" {
		return fmt.Errorf("tag_type must be annotated, got %q", a.TagIdentity.TagType)
	}
	// Require non-empty ACT ID
	if a.ActID == "" {
		return fmt.Errorf("act_id is required")
	}
	// Require closure identity
	if err := ValidateOID("closure_reference.closure_commit", a.ClosureReference.ClosureCommit); err != nil {
		return err
	}
	if err := ValidateOID("closure_reference.closure_tree", a.ClosureReference.ClosureTree); err != nil {
		return err
	}
	// Require freeze identity
	if err := ValidateOID("freeze_reference.freeze_commit", a.FreezeReference.FreezeCommit); err != nil {
		return err
	}
	if err := ValidateOID("freeze_reference.freeze_tree", a.FreezeReference.FreezeTree); err != nil {
		return err
	}
	// Require subject identity
	if err := ValidateOID("subject_reference.subject_commit", a.SubjectReference.SubjectCommit); err != nil {
		return err
	}
	if err := ValidateOID("subject_reference.subject_tree", a.SubjectReference.SubjectTree); err != nil {
		return err
	}
	// Require tag identity
	if err := ValidateOID("tag_identity.tag_object_oid", a.TagIdentity.TagObjectOID); err != nil {
		return err
	}
	if err := ValidateOID("tag_identity.peeled_target", a.TagIdentity.PeeledTarget); err != nil {
		return err
	}
	// Verify identity separation
	if a.FreezeReference.FreezeCommit == a.SubjectReference.SubjectCommit {
		return fmt.Errorf("freeze_commit must differ from subject_commit")
	}
	if a.SubjectReference.SubjectCommit == a.ClosureReference.ClosureCommit {
		return fmt.Errorf("subject_commit must differ from closure_commit")
	}
	// Require all chain validity fields to be true
	if !a.ChainValidity.FNotEqualS {
		return fmt.Errorf("F_not_equal_S must be true")
	}
	if !a.ChainValidity.FIsAncestorOfS {
		return fmt.Errorf("F_is_ancestor_of_S must be true")
	}
	if !a.ChainValidity.PlanBytesFEqualsPlanBytesS {
		return fmt.Errorf("plan_bytes_F_equals_plan_bytes_S must be true")
	}
	if !a.ChainValidity.ManifestFMatchesActualF {
		return fmt.Errorf("manifest.F_matches_actual_F must be true")
	}
	if !a.ChainValidity.ManifestFTreeMatchesFTree {
		return fmt.Errorf("manifest.F_TREE_matches_F_tree must be true")
	}
	if !a.ChainValidity.ManifestSMatchesActualS {
		return fmt.Errorf("manifest.S_matches_actual_S must be true")
	}
	if !a.ChainValidity.ManifestSTreeMatchesSTree {
		return fmt.Errorf("manifest.S_TREE_matches_S_tree must be true")
	}
	if !a.ChainValidity.TagPeeledTargetMatchesC {
		return fmt.Errorf("tag_peeled_target_matches_C must be true")
	}
	// Require no self-reference in plan (all should be false)
	if a.NoSelfReference.PlanFreezeCommitInPlan {
		return fmt.Errorf("plan must not contain freeze_commit")
	}
	if a.NoSelfReference.PlanFreezeTreeInPlan {
		return fmt.Errorf("plan must not contain freeze_tree")
	}
	if a.NoSelfReference.PlanSubjectCommitInPlan {
		return fmt.Errorf("plan must not contain subject_commit")
	}
	if a.NoSelfReference.PlanSubjectTreeInPlan {
		return fmt.Errorf("plan must not contain subject_tree")
	}
	if a.NoSelfReference.PlanClosureCommitInPlan {
		return fmt.Errorf("plan must not contain closure_commit")
	}
	if a.NoSelfReference.PlanClosureTreeInPlan {
		return fmt.Errorf("plan must not contain closure_tree")
	}
	if a.NoSelfReference.PlanTagOIDInPlan {
		return fmt.Errorf("plan must not contain tag_oid")
	}
	if a.NoSelfReference.PlanTagTargetInPlan {
		return fmt.Errorf("plan must not contain tag_target")
	}
	// Cross-check: tag_identity.peeled_target must equal closure_reference.closure_commit
	if a.TagIdentity.PeeledTarget != a.ClosureReference.ClosureCommit {
		return fmt.Errorf("tag_identity.peeled_target %q != closure_reference.closure_commit %q", a.TagIdentity.PeeledTarget, a.ClosureReference.ClosureCommit)
	}
	return nil
}
