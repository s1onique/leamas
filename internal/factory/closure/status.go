package closure

import (
	"bytes"
	"context"
	"fmt"
)

type StatusOptions struct {
	RepositoryDirectory string
	ManifestPath        string
	ReportPath          string
	TagName             string
	Remote              string
}

type StatusResult struct {
	State            string
	RemoteDiagnostic string
}

func Status(ctx context.Context, options StatusOptions) (StatusResult, error) {
	return statusWithGit(ctx, options, RealGit{})
}

func statusWithGit(ctx context.Context, options StatusOptions, git gitClient) (StatusResult, error) {
	if options.RepositoryDirectory == "" {
		options.RepositoryDirectory = "."
	}
	if err := validateClosureTagName(options.TagName); err != nil {
		return StatusResult{}, err
	}
	root, err := runGitValue(ctx, git, options.RepositoryDirectory, "rev-parse", "--show-toplevel")
	if err != nil {
		return StatusResult{}, err
	}
	closure, err := loadCommittedClosure(ctx, git, TagOptions{
		RepositoryDirectory: root,
		ManifestPath:        options.ManifestPath,
		ReportPath:          options.ReportPath,
		TagName:             options.TagName,
		Target:              "HEAD",
	}, root)
	if err != nil {
		return StatusResult{}, err
	}
	if closure.Manifest.Verdict != VerdictPass {
		return StatusResult{State: LifecycleImplemented}, nil
	}
	verified := StatusResult{State: LifecycleVerified}
	if !closureTagExists(ctx, git, root, options.TagName) {
		return verified, nil
	}
	localTagOID, err := localTagObjectOID(ctx, git, root, options.TagName)
	if err != nil {
		return StatusResult{}, err
	}
	expectedMessage := BuildTagMessage(closure)
	if err := verifyLocalTag(ctx, git, root, options.TagName, closure.TargetCommitOID, expectedMessage); err != nil {
		return StatusResult{}, fmt.Errorf("local closure tag integrity failure: %w", err)
	}
	closed := StatusResult{State: LifecycleClosedLocal}
	if options.Remote == "" {
		return closed, nil
	}
	published, diagnostic, err := verifyRemotePublication(ctx, git, closure, options.TagName, options.Remote, localTagOID)
	if err != nil {
		return StatusResult{}, err
	}
	if !published {
		closed.RemoteDiagnostic = diagnostic
		return closed, nil
	}
	return StatusResult{State: LifecyclePublished}, nil
}

func localTagObjectOID(ctx context.Context, git gitClient, root, tagName string) (string, error) {
	value, err := runGitValue(ctx, git, root, "rev-parse", "--verify", "refs/tags/"+tagName)
	if err != nil {
		return "", err
	}
	if err := validateOID("local tag object", value); err != nil {
		return "", err
	}
	return value, nil
}

func validateTagMessage(raw []byte, expected []byte) error {
	_, message, found := bytes.Cut(raw, []byte("\n\n"))
	if !found || !bytes.Equal(message, expected) {
		return fmt.Errorf("annotated tag message contract mismatch")
	}
	return nil
}
