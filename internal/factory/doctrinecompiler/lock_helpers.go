package doctrinecompiler

import (
	"encoding/json"
	"fmt"
)

// patchLockAddManaged inserts a managed-files entry into a JSON lock
// document. It is intended for tests only.
//
// The function round-trips the lock through encoding/json to normalise
// the array, then re-emits with the new entry. This avoids the
// fragility of string replacement.
func patchLockAddManaged(lock, path, digest string) (string, error) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(lock), &doc); err != nil {
		return "", fmt.Errorf("parse lock: %w", err)
	}
	arr, ok := doc["managed_files"].([]any)
	if !ok {
		return "", fmt.Errorf("managed_files not array")
	}
	entry := map[string]any{
		"path":   path,
		"digest": digest,
	}
	arr = append(arr, entry)
	doc["managed_files"] = arr
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
