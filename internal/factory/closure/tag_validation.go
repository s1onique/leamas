package closure

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func loadCommittedClosure(ctx context.Context, git gitClient, options TagOptions, root string) (committedClosure, error) {
	target, err := resolveGitObject(ctx, git, root, options.Target+"^{commit}")
	if err != nil {
		return committedClosure{}, fmt.Errorf("resolve requested closure target: %w", err)
	}
	head, err := resolveGitObject(ctx, git, root, "HEAD^{commit}")
	if err != nil {
		return committedClosure{}, err
	}
	if target != head {
		return committedClosure{}, fmt.Errorf("closure target must equal current HEAD")
	}
	tree, err := resolveGitObject(ctx, git, root, target+"^{tree}")
	if err != nil {
		return committedClosure{}, err
	}
	manifest, manifestBytes, err := LoadManifest(options.ManifestPath)
	if err != nil {
		return committedClosure{}, err
	}
	manifestPath, err := canonicalCommittedPath(root, options.ManifestPath, "docs/closure-manifests/"+manifest.ActID+".json")
	if err != nil {
		return committedClosure{}, err
	}
	reportBytes, err := readBoundedFile(options.ReportPath, MaxReportBytes)
	if err != nil {
		return committedClosure{}, fmt.Errorf("read close report: %w", err)
	}
	reportPath, err := canonicalCommittedPath(root, options.ReportPath, "docs/close-reports/"+manifest.ActID+".md")
	if err != nil {
		return committedClosure{}, err
	}
	if err := requireCommittedBytes(ctx, git, root, target, manifestPath, manifestBytes); err != nil {
		return committedClosure{}, err
	}
	if err := requireCommittedBytes(ctx, git, root, target, reportPath, reportBytes); err != nil {
		return committedClosure{}, err
	}
	planBytes, err := committedBlob(ctx, git, root, target, manifest.Plan.Path)
	if err != nil {
		return committedClosure{}, fmt.Errorf("read committed closure plan: %w", err)
	}
	plan, err := DecodePlan(planBytes)
	if err != nil {
		return committedClosure{}, fmt.Errorf("decode committed closure plan: %w", err)
	}
	if SHA256Hex(planBytes) != manifest.Plan.SHA256 {
		return committedClosure{}, fmt.Errorf("committed plan hash does not match manifest")
	}
	if err := VerifyManifestAgainstPlan(manifest, plan); err != nil {
		return committedClosure{}, err
	}
	rendered, err := Render(manifest, plan)
	if err != nil {
		return committedClosure{}, err
	}
	if !bytes.Equal(rendered, reportBytes) {
		return committedClosure{}, fmt.Errorf("committed report differs from deterministic manifest rendering")
	}
	return committedClosure{
		RepositoryRoot:  root,
		Manifest:        manifest,
		ManifestBytes:   manifestBytes,
		ManifestPath:    manifestPath,
		ReportBytes:     reportBytes,
		ReportPath:      reportPath,
		Plan:            plan,
		PlanBytes:       planBytes,
		TargetCommitOID: target,
		TargetTreeOID:   tree,
	}, nil
}

func canonicalCommittedPath(root, path, expected string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(absolute)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("closure artifact must be a non-symlink regular file")
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}
	relative = filepath.ToSlash(relative)
	if relative != expected {
		return "", fmt.Errorf("closure artifact path must be canonical %q", expected)
	}
	return relative, nil
}

func requireCommittedBytes(ctx context.Context, git gitClient, root, target, path string, working []byte) error {
	committed, err := committedBlob(ctx, git, root, target, path)
	if err != nil {
		return err
	}
	if !bytes.Equal(committed, working) {
		return fmt.Errorf("working bytes for %s differ from closure commit", path)
	}
	return nil
}

func committedBlob(ctx context.Context, git gitClient, root, target, path string) ([]byte, error) {
	result := git.Run(ctx, root, "cat-file", "blob", target+":"+path)
	if result.Err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("closure commit does not contain %s", path)
	}
	return result.Stdout, nil
}
