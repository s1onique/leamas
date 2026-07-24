package closure

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
)

var closureTagPattern = regexp.MustCompile(`^act/[a-z0-9][a-z0-9._/-]{0,190}[a-z0-9]$`)

type TagOptions struct {
	RepositoryDirectory string
	ManifestPath        string
	ReportPath          string
	TagName             string
	Target              string
}

type committedClosure struct {
	RepositoryRoot  string
	Manifest        Manifest
	ManifestBytes   []byte
	ManifestPath    string
	ReportBytes     []byte
	ReportPath      string
	Plan            Plan
	PlanBytes       []byte
	TargetCommitOID string
	TargetTreeOID   string
}

func CreateTag(ctx context.Context, options TagOptions) ([]byte, error) {
	if options.RepositoryDirectory == "" {
		options.RepositoryDirectory = "."
	}
	if err := validateClosureTagName(options.TagName); err != nil {
		return nil, err
	}
	if options.Target == "" {
		return nil, fmt.Errorf("closure tag target is required")
	}
	git := RealGit{}
	root, err := runGitValue(ctx, git, options.RepositoryDirectory, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	clean, err := workingTreeClean(ctx, git, root)
	if err != nil || !clean {
		return nil, fmt.Errorf("closure tag refused because the working tree is dirty")
	}
	closure, err := loadCommittedClosure(ctx, git, options, root)
	if err != nil {
		return nil, err
	}
	if closure.Manifest.Verdict != VerdictPass {
		return nil, fmt.Errorf("closure tag requires a passing manifest")
	}
	if err := requireSubjectAncestor(ctx, git, closure); err != nil {
		return nil, err
	}
	if closureTagExists(ctx, git, root, options.TagName) {
		return nil, fmt.Errorf("closure tag %q already exists", options.TagName)
	}
	message := BuildTagMessage(closure)
	result := git.Run(ctx, root,
		"tag", "--annotate", "--cleanup=verbatim", "--message", string(message), "--", options.TagName, closure.TargetCommitOID)
	if result.Err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("create annotated closure tag: %s", sanitizeDiagnostic(string(result.Stderr)))
	}
	if err := verifyLocalTag(ctx, git, root, options.TagName, closure.TargetCommitOID, message); err != nil {
		return nil, err
	}
	return message, nil
}

func BuildTagMessage(closure committedClosure) []byte {
	return []byte(fmt.Sprintf(
		"LEAMAS_CLOSURE_TAG_CONTRACT_VERSION: 1\n"+
			"act_id: %s\n"+
			"verdict: %s\n"+
			"subject_commit_oid: %s\n"+
			"subject_tree_oid: %s\n"+
			"closure_commit_oid: %s\n"+
			"closure_tree_oid: %s\n"+
			"plan_path: %s\n"+
			"plan_sha256: %s\n"+
			"manifest_path: %s\n"+
			"manifest_sha256: %s\n"+
			"report_path: %s\n"+
			"report_sha256: %s\n",
		closure.Manifest.ActID,
		closure.Manifest.Verdict,
		closure.Manifest.Subject.CommitOID,
		closure.Manifest.Subject.TreeOID,
		closure.TargetCommitOID,
		closure.TargetTreeOID,
		closure.Manifest.Plan.Path,
		closure.Manifest.Plan.SHA256,
		closure.ManifestPath,
		SHA256Hex(closure.ManifestBytes),
		closure.ReportPath,
		SHA256Hex(closure.ReportBytes),
	))
}

func validateClosureTagName(name string) error {
	if !closureTagPattern.MatchString(name) || strings.Contains(name, "..") || strings.Contains(name, "//") || strings.HasSuffix(name, ".lock") {
		return fmt.Errorf("malformed immutable closure tag name %q", name)
	}
	return nil
}

func closureTagExists(ctx context.Context, git gitClient, root, name string) bool {
	result := git.Run(ctx, root, "show-ref", "--verify", "--quiet", "refs/tags/"+name)
	return result.Err == nil && result.ExitCode == 0
}

func requireSubjectAncestor(ctx context.Context, git gitClient, closure committedClosure) error {
	result := git.Run(ctx, closure.RepositoryRoot, "merge-base", "--is-ancestor", closure.Manifest.Subject.CommitOID, closure.TargetCommitOID)
	if result.Err != nil || result.ExitCode != 0 {
		return fmt.Errorf("verified subject is not an ancestor of closure commit")
	}
	return nil
}

func verifyLocalTag(ctx context.Context, git gitClient, root, name, target string, expectedMessage []byte) error {
	objectType, err := runGitValue(ctx, git, root, "cat-file", "-t", "refs/tags/"+name)
	if err != nil || objectType != "tag" {
		return fmt.Errorf("created closure ref is not an annotated tag")
	}
	peeled, err := resolveGitObject(ctx, git, root, "refs/tags/"+name+"^{commit}")
	if err != nil || peeled != target {
		return fmt.Errorf("created closure tag target mismatch")
	}
	raw := git.Run(ctx, root, "cat-file", "tag", "refs/tags/"+name)
	if raw.Err != nil || raw.ExitCode != 0 {
		return fmt.Errorf("read created closure tag object")
	}
	_, message, found := bytes.Cut(raw.Stdout, []byte("\n\n"))
	if !found || !bytes.Equal(message, expectedMessage) {
		return fmt.Errorf("created closure tag message mismatch")
	}
	return nil
}
