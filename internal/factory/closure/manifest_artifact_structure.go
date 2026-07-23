package closure

import (
	"fmt"
)

func validateArtifactResult(index int, artifact ArtifactResult) error {
	prefix := fmt.Sprintf("artifacts[%d]", index)
	if !itemIDPattern.MatchString(artifact.ArtifactID) || containsClosurePlaceholder(artifact.ArtifactID) {
		return fmt.Errorf("%s.artifact_id is invalid", prefix)
	}
	if err := validateRepositoryRelativePath(artifact.Path, false); err != nil {
		return fmt.Errorf("%s.path: %w", prefix, err)
	}
	if artifact.MediaType == "" || containsClosurePlaceholder(artifact.MediaType) {
		return fmt.Errorf("%s.media_type is invalid", prefix)
	}
	if artifact.ByteCount < 0 {
		return fmt.Errorf("%s.byte_count is negative", prefix)
	}
	switch artifact.Status {
	case ArtifactStatusPass:
		if err := validateSHA256(prefix+".sha256", artifact.SHA256); err != nil {
			return err
		}
		if artifact.Diagnostic != "" {
			return fmt.Errorf("%s passing artifact has a diagnostic", prefix)
		}
	case ArtifactStatusMissing, ArtifactStatusFail:
		if artifact.SHA256 != "" || artifact.ByteCount != 0 || artifact.Diagnostic == "" {
			return fmt.Errorf("%s failed artifact evidence is inconsistent", prefix)
		}
	default:
		return fmt.Errorf("%s.status is invalid", prefix)
	}
	return nil
}

func validateEvidenceRecords(records []EvidenceRecord) error {
	seen := make(map[string]struct{}, len(records))
	for i, record := range records {
		prefix := fmt.Sprintf("detached_evidence[%d]", i)
		if !itemIDPattern.MatchString(record.LogicalName) || containsClosurePlaceholder(record.LogicalName) {
			return fmt.Errorf("%s.logical_name is invalid", prefix)
		}
		if _, exists := seen[record.LogicalName]; exists {
			return fmt.Errorf("duplicate detached evidence logical_name %q", record.LogicalName)
		}
		seen[record.LogicalName] = struct{}{}
		if record.MediaType == "" || containsClosurePlaceholder(record.MediaType) {
			return fmt.Errorf("%s.media_type is invalid", prefix)
		}
		if err := validateSHA256(prefix+".sha256", record.SHA256); err != nil {
			return err
		}
		if record.ByteCount < 0 {
			return fmt.Errorf("%s.byte_count is negative", prefix)
		}
		if record.Availability != "detached" {
			return fmt.Errorf("%s.availability must be detached", prefix)
		}
	}
	return nil
}
