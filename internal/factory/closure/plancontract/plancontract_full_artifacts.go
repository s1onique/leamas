// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full_artifacts.go owns
// the per-artifact validation for the optional Plan
// Contract v1 "artifacts" array.
//
// Each entry is a JSON object with id (canonical pattern),
// path (repository-relative), required (bool), max_bytes
// (positive integer), and media_type (non-empty string).
// Duplicate IDs are rejected.
package plancontract

import (
	"encoding/json"
	"fmt"
	"strings"
)

// validateArtifactsOptional enforces:
//   - artifacts MUST be a JSON array when present.
//   - len(artifacts) <= MaxArtifacts.
//   - each entry satisfies validateArtifactMap.
//   - JSON null is treated as "no artifacts" so the typed
//     Plan's nil slice round-trips cleanly through the leaf.
func validateArtifactsOptional(obj map[string]any) error {
	rawArtifacts, ok := obj["artifacts"]
	if !ok || rawArtifacts == nil {
		return nil
	}
	artifacts, ok := rawArtifacts.([]any)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        "artifacts",
			InstancePath: "/artifacts",
			Message:      "artifacts is not an array",
		}
	}
	if len(artifacts) > MaxArtifacts {
		return &DecodeError{
			Code:         "too_many_artifacts",
			Field:        "artifacts",
			InstancePath: "/artifacts",
			Message:      fmt.Sprintf("artifacts count %d exceeds %d", len(artifacts), MaxArtifacts),
		}
	}
	seenArtifactIDs := map[string]struct{}{}
	for i, rawArtifact := range artifacts {
		if err := validateArtifactMap(i, rawArtifact, seenArtifactIDs); err != nil {
			return err
		}
	}
	return nil
}

// validateArtifactMap enforces the per-artifact rules:
//   - id is present, matches ItemIDPattern, and has no placeholder.
//   - id is unique within the artifacts array.
//   - path is a repository-relative path.
//   - required is present (boolean).
//   - max_bytes is a positive integer.
//   - media_type is a non-empty string with no placeholder.
func validateArtifactMap(index int, raw any, seenIDs map[string]struct{}) error {
	artifact, ok := raw.(map[string]any)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        fmt.Sprintf("artifacts[%d]", index),
			InstancePath: fmt.Sprintf("/artifacts/%d", index),
			Message:      fmt.Sprintf("artifacts[%d] is not an object", index),
		}
	}
	id, ok := artifact["id"].(string)
	if !ok || id == "" {
		return &DecodeError{
			Code:         "missing_field",
			Field:        fmt.Sprintf("artifacts[%d].id", index),
			InstancePath: fmt.Sprintf("/artifacts/%d/id", index),
			Message:      fmt.Sprintf("artifacts[%d].id is required", index),
		}
	}
	if !ItemIDPattern.MatchString(id) || containsClosurePlaceholder(id) {
		return &DecodeError{
			Code:         "invalid_artifact_id",
			Field:        fmt.Sprintf("artifacts[%d].id", index),
			InstancePath: fmt.Sprintf("/artifacts/%d/id", index),
			Message:      fmt.Sprintf("artifacts[%d].id %q is invalid", index, id),
		}
	}
	if _, dup := seenIDs[id]; dup {
		return &DecodeError{
			Code:         "duplicate_artifact_id",
			Field:        fmt.Sprintf("artifacts[%d].id", index),
			InstancePath: fmt.Sprintf("/artifacts/%d/id", index),
			Message:      fmt.Sprintf("duplicate artifact id %q at artifacts[%d]", id, index),
		}
	}
	seenIDs[id] = struct{}{}

	path, ok := artifact["path"].(string)
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        fmt.Sprintf("artifacts[%d].path", index),
			InstancePath: fmt.Sprintf("/artifacts/%d/path", index),
			Message:      fmt.Sprintf("artifacts[%d].path is required", index),
		}
	}
	if err := validateRepositoryRelativePath(path, false, false); err != nil {
		return &DecodeError{
			Code:         "invalid_artifact_path",
			Field:        fmt.Sprintf("artifacts[%d].path", index),
			InstancePath: fmt.Sprintf("/artifacts/%d/path", index),
			Message:      fmt.Sprintf("artifacts[%d].path %q is invalid: %s", index, path, err),
		}
	}

	// B2-R6: required MUST be a JSON boolean. The previous
	// "is the key present" check accepted "required":"yes"
	// which broke the typed-Plan pointer semantics.
	if _, ok := artifact["required"]; !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        fmt.Sprintf("artifacts[%d].required", index),
			InstancePath: fmt.Sprintf("/artifacts/%d/required", index),
			Message:      fmt.Sprintf("artifacts[%d].required is required", index),
		}
	}
	if _, ok := artifact["required"].(bool); !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        fmt.Sprintf("artifacts[%d].required", index),
			InstancePath: fmt.Sprintf("/artifacts/%d/required", index),
			Message:      fmt.Sprintf("artifacts[%d].required must be a boolean", index),
		}
	}

	rawMax, ok := artifact["max_bytes"]
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        fmt.Sprintf("artifacts[%d].max_bytes", index),
			InstancePath: fmt.Sprintf("/artifacts/%d/max_bytes", index),
			Message:      fmt.Sprintf("artifacts[%d].max_bytes is required", index),
		}
	}
	maxN, ok := rawMax.(json.Number)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        fmt.Sprintf("artifacts[%d].max_bytes", index),
			InstancePath: fmt.Sprintf("/artifacts/%d/max_bytes", index),
			Message:      fmt.Sprintf("artifacts[%d].max_bytes is not a number", index),
		}
	}
	maxIv, err := maxN.Int64()
	if err != nil || maxIv <= 0 {
		return &DecodeError{
			Code:         "invalid_max_bytes",
			Field:        fmt.Sprintf("artifacts[%d].max_bytes", index),
			InstancePath: fmt.Sprintf("/artifacts/%d/max_bytes", index),
			Message:      fmt.Sprintf("artifacts[%d].max_bytes %s is not a positive integer", index, maxN.String()),
		}
	}

	// B2-R6: media_type MUST reject whitespace-only values.
	// The historical typed-Plan validator trimmed whitespace
	// before checking emptiness; preserve that contract.
	mediaType, ok := artifact["media_type"].(string)
	if !ok || strings.TrimSpace(mediaType) == "" || containsClosurePlaceholder(mediaType) {
		return &DecodeError{
			Code:         "invalid_media_type",
			Field:        fmt.Sprintf("artifacts[%d].media_type", index),
			InstancePath: fmt.Sprintf("/artifacts/%d/media_type", index),
			Message:      fmt.Sprintf("artifacts[%d].media_type is invalid", index),
		}
	}

	return nil
}
