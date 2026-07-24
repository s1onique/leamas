package closure

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// ForbiddenPlanKeys are keys that must not appear in a frozen plan.
var ForbiddenPlanKeys = []string{
	"freeze_commit",
	"freeze_tree",
	"subject_commit",
	"subject_tree",
	"closure_commit",
	"closure_tree",
	"tag_oid",
	"tag_object_oid",
	"tag_target",
	"peeled_target",
}

// ValidatePlanStructure recursively validates plan JSON has no forbidden keys.
func ValidatePlanStructure(planPath string) error {
	data, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("read plan: %w", err)
	}

	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse plan JSON: %w", err)
	}

	return checkForbiddenKeys(raw, "")
}

// checkForbiddenKeys recursively checks for forbidden keys at any nesting level.
func checkForbiddenKeys(node interface{}, path string) error {
	switch v := node.(type) {
	case map[string]interface{}:
		for key, value := range v {
			currentPath := key
			if path != "" {
				currentPath = path + "." + key
			}

			// Check if this key is forbidden
			for _, forbidden := range ForbiddenPlanKeys {
				if key == forbidden {
					return fmt.Errorf("forbidden key %q at path %q", key, currentPath)
				}
			}

			// Recursively check nested values
			if err := checkForbiddenKeys(value, currentPath); err != nil {
				return err
			}
		}
	case []interface{}:
		for i, item := range v {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			if err := checkForbiddenKeys(item, itemPath); err != nil {
				return err
			}
		}
		// Primitives (string, number, bool, null) cannot contain forbidden keys
	}
	return nil
}

// ValidatePlanBytes validates plan bytes directly.
func ValidatePlanBytes(data []byte) error {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse plan JSON: %w", err)
	}

	return checkForbiddenKeys(raw, "")
}

// ValidatePlanFromBytes validates plan bytes and returns self-reference info.
func ValidatePlanFromBytes(data []byte) (NoSelfReferenceInPlan, error) {
	var noSelf NoSelfReferenceInPlan

	// Parse JSON
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return noSelf, fmt.Errorf("parse JSON: %w", err)
	}

	// Check for forbidden keys recursively
	if err := checkForbiddenKeys(raw, ""); err != nil {
		return noSelf, err
	}

	// Also check using regex for SHA-1 OID patterns in specific fields
	content := string(data)
	selfRefFields := []struct {
		key    string
		setter func(bool)
	}{
		{"freeze_commit", func(b bool) { noSelf.PlanFreezeCommitInPlan = b }},
		{"freeze_tree", func(b bool) { noSelf.PlanFreezeTreeInPlan = b }},
		{"subject_commit", func(b bool) { noSelf.PlanSubjectCommitInPlan = b }},
		{"subject_tree", func(b bool) { noSelf.PlanSubjectTreeInPlan = b }},
		{"closure_commit", func(b bool) { noSelf.PlanClosureCommitInPlan = b }},
		{"closure_tree", func(b bool) { noSelf.PlanClosureTreeInPlan = b }},
		{"tag_oid", func(b bool) { noSelf.PlanTagOIDInPlan = b }},
		{"tag_target", func(b bool) { noSelf.PlanTagTargetInPlan = b }},
	}

	for _, f := range selfRefFields {
		pattern := fmt.Sprintf(`"%s"\s*:\s*"[0-9a-f]{40}"`, f.key)
		matched, _ := regexp.MatchString(pattern, content)
		f.setter(matched)
	}

	return noSelf, nil
}
