package closure

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// DetectStorageFormat detects the repository storage format.
func DetectStorageFormat(ctx context.Context, git gitClient, repoRoot string) (ObjectFormat, error) {
	format, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--show-object-format=storage")
	if err != nil {
		return ObjectFormatUnknown, fmt.Errorf("detect storage format: %w", err)
	}
	switch format {
	case "sha1":
		return ObjectFormatSHA1, nil
	case "sha256":
		return ObjectFormatSHA256, nil
	default:
		return ObjectFormatUnknown, fmt.Errorf("unsupported storage format: %s", format)
	}
}

// resolveAndTypeCheck resolves an OID and verifies its type using Git's typed revision syntax.
func resolveAndTypeCheck(ctx context.Context, git gitClient, repoRoot, revision, expectedType string) (string, error) {
	// Use Git's ^{type} syntax for type peeling
	typedRevision := revision
	if !strings.Contains(revision, "^") {
		typedRevision = revision + "^{" + expectedType + "}"
	}
	oid, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options", typedRevision)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", typedRevision, err)
	}
	objType, err := runGitValue(ctx, git, repoRoot, "cat-file", "-t", oid)
	if err != nil {
		return "", fmt.Errorf("get type of %s: %w", oid, err)
	}
	if objType != expectedType {
		return "", fmt.Errorf("%s is %s, expected %s", oid, objType, expectedType)
	}
	return oid, nil
}

// isAncestor checks if commitA is an ancestor of commitB.
func isAncestor(ctx context.Context, git gitClient, repoRoot, commitA, commitB string) (bool, error) {
	result := git.Run(ctx, repoRoot, "merge-base", "--is-ancestor", commitA, commitB)
	if result.Err != nil {
		return false, fmt.Errorf("merge-base: %w", result.Err)
	}
	switch result.ExitCode {
	case 0:
		return true, nil
	case 1:
		return false, nil
	default:
		return false, fmt.Errorf("merge-base failed with exit %d", result.ExitCode)
	}
}

// getTree returns the tree OID for a commit using ^{tree} syntax.
func getTree(ctx context.Context, git gitClient, repoRoot, commit string) (string, error) {
	// Ensure ^{tree} suffix is present
	treeRef := commit
	if !strings.HasSuffix(commit, "^{tree}") {
		treeRef = commit + "^{tree}"
	}
	return runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options", treeRef)
}

// getPlanBytes retrieves plan bytes at a specific commit.
func getPlanBytes(ctx context.Context, git gitClient, repoRoot, commit, planPath string) ([]byte, error) {
	result := git.Run(ctx, repoRoot, "cat-file", "blob", commit+":"+planPath)
	if result.Err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("get plan blob at %s:%s", commit, planPath)
	}
	return result.Stdout, nil
}

// getTagInfo retrieves annotated tag information using proper ^{tag} and ^{commit} syntax.
func getTagInfo(ctx context.Context, git gitClient, repoRoot, tagName string) (string, string, bool, error) {
	tagRef := "refs/tags/" + tagName

	// First verify the tag exists
	if _, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options", tagRef); err != nil {
		return "", "", false, fmt.Errorf("tag %s not found: %w", tagName, err)
	}

	// Resolve tag object using ^{tag} syntax
	tagObj, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options", tagRef+"^{tag}")
	if err != nil {
		return "", "", false, fmt.Errorf("tag %s is lightweight, annotated tag required: %w", tagName, err)
	}

	// Verify it's actually a tag object
	tagType, err := runGitValue(ctx, git, repoRoot, "cat-file", "-t", tagObj)
	if err != nil {
		return "", "", false, fmt.Errorf("get tag object type: %w", err)
	}
	if tagType != "tag" {
		return "", "", false, fmt.Errorf("tag %s is lightweight, annotated tag required", tagName)
	}

	// Get peeled target using ^{commit} syntax
	peeled, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options", tagRef+"^{commit}")
	if err != nil {
		return "", "", false, fmt.Errorf("peel tag %s: %w", tagName, err)
	}
	return tagObj, peeled, true, nil
}

// VerifyChain validates the F → S → C → tag chain using Git.
// A PASS requires: Freeze, Subject, Closure, PlanPath, Tag all provided,
// and all chain invariants to be satisfied.
func VerifyChain(ctx context.Context, req ChainValidationRequest) (ChainValidationResult, error) {
	var result ChainValidationResult

	// Repository root is required - fail closed if missing
	if req.RepoRoot == "" {
		result.Errors = append(result.Errors, "repository root is required")
		result.Verdict = "FAIL"
		return result, nil
	}

	// All chain fields are required for PASS
	if req.Freeze == "" {
		result.Errors = append(result.Errors, "freeze commit is required")
	}
	if req.Subject == "" {
		result.Errors = append(result.Errors, "subject commit is required")
	}
	if req.Closure == "" {
		result.Errors = append(result.Errors, "closure commit is required")
	}
	if req.PlanPath == "" {
		result.Errors = append(result.Errors, "plan_path is required")
	}
	if req.Tag == "" {
		result.Errors = append(result.Errors, "tag is required")
	}
	if len(result.Errors) > 0 {
		result.Verdict = "FAIL"
		return result, nil
	}

	// Detect repository storage format
	format, err := DetectStorageFormat(ctx, req.Git, req.RepoRoot)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Verdict = "FAIL"
		return result, nil
	}

	// Validate all OIDs against storage format
	for name, val := range map[string]string{
		"freeze":  req.Freeze,
		"subject": req.Subject,
		"closure": req.Closure,
	} {
		if err := ValidateOIDWithFormat(name, val, format); err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}
	if len(result.Errors) > 0 {
		result.Verdict = "FAIL"
		return result, nil
	}

	// F != S check
	result.FNotEqualS = req.Freeze != req.Subject
	if !result.FNotEqualS {
		result.Errors = append(result.Errors, "freeze_commit equals subject_commit")
	}

	// Resolve and type-check commits using ^{commit} syntax
	fCommit, err := resolveAndTypeCheck(ctx, req.Git, req.RepoRoot, req.Freeze+"^{commit}", "commit")
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		result.AllChecks = append(result.AllChecks, fmt.Sprintf("freeze=%s (commit)", fCommit))
	}

	sCommit, err := resolveAndTypeCheck(ctx, req.Git, req.RepoRoot, req.Subject+"^{commit}", "commit")
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		result.AllChecks = append(result.AllChecks, fmt.Sprintf("subject=%s (commit)", sCommit))
	}

	cCommit, err := resolveAndTypeCheck(ctx, req.Git, req.RepoRoot, req.Closure+"^{commit}", "commit")
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		result.AllChecks = append(result.AllChecks, fmt.Sprintf("closure=%s (commit)", cCommit))
	}

	// Resolve trees using ^{tree} syntax
	fTree, err := getTree(ctx, req.Git, req.RepoRoot, req.Freeze)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		result.AllChecks = append(result.AllChecks, fmt.Sprintf("freeze_tree=%s", fTree))
	}
	sTree, err := getTree(ctx, req.Git, req.RepoRoot, req.Subject)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		result.AllChecks = append(result.AllChecks, fmt.Sprintf("subject_tree=%s", sTree))
	}
	cTree, err := getTree(ctx, req.Git, req.RepoRoot, req.Closure)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		result.AllChecks = append(result.AllChecks, fmt.Sprintf("closure_tree=%s", cTree))
	}

	// Verify ancestry with error propagation
	if result.FNotEqualS && fCommit != "" && sCommit != "" {
		isAncestor, err := isAncestor(ctx, req.Git, req.RepoRoot, fCommit, sCommit)
		result.FIsAncestorOfS = isAncestor
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("ancestry check F→S: %v", err))
		} else if !isAncestor {
			result.Errors = append(result.Errors, "freeze is not an ancestor of subject")
		}
	}

	if sCommit != "" && cCommit != "" {
		isAncestor, err := isAncestor(ctx, req.Git, req.RepoRoot, sCommit, cCommit)
		result.SIsAncestorOfC = isAncestor
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("ancestry check S→C: %v", err))
		} else if !isAncestor {
			result.Errors = append(result.Errors, "subject is not an ancestor of closure")
		}
	}

	if fCommit != "" && cCommit != "" {
		isAncestor, err := isAncestor(ctx, req.Git, req.RepoRoot, fCommit, cCommit)
		result.FIsAncestorOfC = isAncestor
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("ancestry check F→C: %v", err))
		} else if !isAncestor {
			result.Errors = append(result.Errors, "freeze is not an ancestor of closure")
		}
	}

	// Verify plan bytes at F equal plan bytes at S
	if req.PlanPath != "" && fCommit != "" && sCommit != "" {
		fPlan, err := getPlanBytes(ctx, req.Git, req.RepoRoot, fCommit, req.PlanPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("get plan at freeze: %v", err))
		} else {
			sPlan, err := getPlanBytes(ctx, req.Git, req.RepoRoot, sCommit, req.PlanPath)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("get plan at subject: %v", err))
			} else {
				result.PlanBytesFEqualsPlanBytesS = bytes.Equal(fPlan, sPlan)
				if !result.PlanBytesFEqualsPlanBytesS {
					result.Errors = append(result.Errors, "plan bytes differ between freeze and subject")
				}
				// Validate plan structure
				if err := ValidatePlanBytes(fPlan); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("freeze plan structure invalid: %v", err))
				}
				if err := ValidatePlanBytes(sPlan); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("subject plan structure invalid: %v", err))
				}
			}
		}
	}

	// Manifest binding validation
	if req.Manifest != nil && fCommit != "" && sCommit != "" {
		// manifest.F == actual F
		result.ManifestFMatchesActualF = (req.Manifest.PlanFreeze.FreezeCommit == fCommit)
		if !result.ManifestFMatchesActualF {
			result.Errors = append(result.Errors, fmt.Sprintf("manifest freeze_commit %s != actual %s", req.Manifest.PlanFreeze.FreezeCommit, fCommit))
		}

		// manifest.F_TREE == actual F^{tree} (exact comparison)
		result.ManifestFTreeMatchesFTree = (req.Manifest.PlanFreeze.FreezeTree != "" && req.Manifest.PlanFreeze.FreezeTree == fTree)
		if !result.ManifestFTreeMatchesFTree {
			result.Errors = append(result.Errors, fmt.Sprintf("manifest freeze_tree %s != actual F^{tree} %s", req.Manifest.PlanFreeze.FreezeTree, fTree))
		}

		// manifest.S == actual S
		result.ManifestSMatchesActualS = (req.Manifest.Subject.CommitOID == sCommit)
		if !result.ManifestSMatchesActualS {
			result.Errors = append(result.Errors, fmt.Sprintf("manifest subject.commit_oid %s != actual %s", req.Manifest.Subject.CommitOID, sCommit))
		}

		// manifest.S_TREE == actual S^{tree} (exact comparison)
		result.ManifestSTreeMatchesSTree = (req.Manifest.Subject.TreeOID != "" && req.Manifest.Subject.TreeOID == sTree)
		if !result.ManifestSTreeMatchesSTree {
			result.Errors = append(result.Errors, fmt.Sprintf("manifest subject.tree_oid %s != actual S^{tree} %s", req.Manifest.Subject.TreeOID, sTree))
		}

		// Plan path binding: manifest.plan.path == manifest.plan_freeze.plan_path == CLI --plan-path
		planPathMatch := (req.Manifest.Plan.Path != "" && req.Manifest.PlanFreeze.PlanPath != "")
		if planPathMatch {
			planPathMatch = (req.Manifest.Plan.Path == req.Manifest.PlanFreeze.PlanPath && req.Manifest.PlanFreeze.PlanPath == req.PlanPath)
		}
		if req.Manifest.Plan.Path == "" || req.Manifest.PlanFreeze.PlanPath == "" || req.PlanPath == "" {
			result.Errors = append(result.Errors, "plan path missing in manifest")
		} else if !planPathMatch {
			result.Errors = append(result.Errors, fmt.Sprintf("plan path mismatch: manifest.plan.path=%s manifest.plan_freeze.plan_path=%s CLI=%s", req.Manifest.Plan.Path, req.Manifest.PlanFreeze.PlanPath, req.PlanPath))
		}

		// Plan blob OID binding: manifest.plan_freeze.plan_blob_oid == F:<plan-path>
		if req.Manifest.PlanFreeze.PlanBlobOID != "" && req.Manifest.PlanFreeze.PlanPath != "" {
			actualBlobOID, err := runGitValue(ctx, req.Git, req.RepoRoot, "rev-parse", "--verify", fCommit+":"+req.Manifest.PlanFreeze.PlanPath)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("resolve plan blob at %s:%s: %v", fCommit, req.Manifest.PlanFreeze.PlanPath, err))
			} else if actualBlobOID != req.Manifest.PlanFreeze.PlanBlobOID {
				result.Errors = append(result.Errors, fmt.Sprintf("manifest plan_blob_oid %s != actual %s", req.Manifest.PlanFreeze.PlanBlobOID, actualBlobOID))
			}
		}

		// Plan SHA-256 binding: both manifest fields must match SHA-256(F:<plan-path>)
		if req.Manifest.PlanFreeze.PlanPath != "" {
			planBytes, err := getPlanBytes(ctx, req.Git, req.RepoRoot, fCommit, req.Manifest.PlanFreeze.PlanPath)
			if err == nil {
				actualSHA256 := fmt.Sprintf("%x", sha256.Sum256(planBytes))
				if req.Manifest.Plan.SHA256 != "" && req.Manifest.Plan.SHA256 != actualSHA256 {
					result.Errors = append(result.Errors, fmt.Sprintf("manifest.plan.sha256 %s != actual %s", req.Manifest.Plan.SHA256, actualSHA256))
				}
				if req.Manifest.PlanFreeze.PlanSHA256 != "" && req.Manifest.PlanFreeze.PlanSHA256 != actualSHA256 {
					result.Errors = append(result.Errors, fmt.Sprintf("manifest.plan_freeze.plan_sha256 %s != actual %s", req.Manifest.PlanFreeze.PlanSHA256, actualSHA256))
				}
			}
		}
	}

	// Verify tag is annotated and peeled target equals C
	if req.Tag != "" {
		tagObjOID, peeledTarget, isAnnotated, err := getTagInfo(ctx, req.Git, req.RepoRoot, req.Tag)
		result.TagIsAnnotated = isAnnotated
		result.TagObjectIsTag = isAnnotated
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.AllChecks = append(result.AllChecks, fmt.Sprintf("tag_object=%s (tag)", tagObjOID))
			result.TagPeeledTargetMatchesC = (peeledTarget == cCommit)
			if !result.TagPeeledTargetMatchesC {
				result.Errors = append(result.Errors, fmt.Sprintf("tag peeled target %s does not match closure %s", peeledTarget, cCommit))
			} else {
				result.AllChecks = append(result.AllChecks, fmt.Sprintf("peeled_target=%s (commit)", peeledTarget))
			}
		}
	}

	if len(result.Errors) == 0 {
		result.Verdict = "PASS"
	} else {
		result.Verdict = "FAIL"
	}

	return result, nil
}

// Output formats the validation result for output.
func (r ChainValidationResult) Output(w io.Writer, jsonFormat bool) {
	if jsonFormat {
		data, _ := json.MarshalIndent(r, "", "  ")
		fmt.Fprintln(w, string(data))
	} else {
		fmt.Fprintf(w, "Verdict: %s\n", r.Verdict)
		if len(r.Errors) > 0 {
			fmt.Fprintln(w, "Errors:")
			for _, e := range r.Errors {
				fmt.Fprintf(w, "  - %s\n", e)
			}
		}
		if len(r.AllChecks) > 0 {
			fmt.Fprintln(w, "Checks:")
			for _, c := range r.AllChecks {
				fmt.Fprintf(w, "  - %s\n", c)
			}
		}
	}
}
