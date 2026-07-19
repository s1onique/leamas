package releasedeb

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/version"
)

// PublicationConfig is the fail-closed Git/tag/asset contract used before gh
// can create a release.
type PublicationConfig struct {
	Repository string
	Remote     string
	Tag        string
	Assets     []string
}

// CheckPublication proves that the local tree and pushed tag identify the
// same commit and that the requested asset names are unique.
func CheckPublication(ctx context.Context, cfg PublicationConfig) error {
	if cfg.Repository == "" {
		return fmt.Errorf("publication repository is empty")
	}
	if cfg.Remote == "" {
		return fmt.Errorf("publication remote is empty")
	}
	if err := validateReleaseTag(cfg.Tag); err != nil {
		return err
	}
	if err := validateAssetNames(cfg.Assets); err != nil {
		return err
	}

	status, err := commandOutput(ctx, cfg.Repository, "git", "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("check publication tree: %w", err)
	}
	if strings.TrimSpace(string(status)) != "" {
		return fmt.Errorf("publication tree is dirty")
	}
	head, err := gitValue(ctx, cfg.Repository, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	pointsAt, err := gitValue(ctx, cfg.Repository, "tag", "--points-at", "HEAD")
	if err != nil {
		return err
	}
	if pointsAt != cfg.Tag {
		return fmt.Errorf("tag %s does not point at HEAD (tags at HEAD: %q)", cfg.Tag, pointsAt)
	}
	localCommit, err := gitValue(ctx, cfg.Repository, "rev-parse", cfg.Tag+"^{commit}")
	if err != nil {
		return fmt.Errorf("resolve local tag %s: %w", cfg.Tag, err)
	}
	if localCommit != head {
		return fmt.Errorf("tag %s commit %s does not match HEAD %s", cfg.Tag, localCommit, head)
	}

	remoteOutput, err := commandOutput(ctx, cfg.Repository, "git", "ls-remote", "--tags", cfg.Remote,
		"refs/tags/"+cfg.Tag, "refs/tags/"+cfg.Tag+"^{}")
	if err != nil {
		return fmt.Errorf("check remote tag %s: %w", cfg.Tag, err)
	}
	remoteCommit := remoteTagCommit(string(remoteOutput), cfg.Tag)
	if remoteCommit == "" {
		return fmt.Errorf("remote tag %s does not exist", cfg.Tag)
	}
	if remoteCommit != head {
		return fmt.Errorf("remote tag %s commit %s does not match HEAD %s", cfg.Tag, remoteCommit, head)
	}
	return nil
}

func validateReleaseTag(tag string) error {
	if !strings.HasPrefix(tag, "v") {
		return fmt.Errorf("release tag must start with v: %q", tag)
	}
	value := strings.TrimPrefix(tag, "v")
	parts, ok := version.ParseSemVer(value)
	if !ok || len(parts.Pre) != 0 || parts.Build != "" {
		return fmt.Errorf("release tag must be a strict stable SemVer tag: %q", tag)
	}
	return nil
}

func validateAssetNames(assets []string) error {
	if len(assets) == 0 {
		return fmt.Errorf("release asset list is empty")
	}
	seen := make(map[string]string, len(assets))
	for _, asset := range assets {
		name := filepath.Base(asset)
		if name == "." || name == string(filepath.Separator) || name == "" {
			return fmt.Errorf("release asset has no basename: %q", asset)
		}
		if previous, ok := seen[name]; ok {
			return fmt.Errorf("duplicate release asset name %q from %q and %q", name, previous, asset)
		}
		seen[name] = asset
	}
	return nil
}

func gitValue(ctx context.Context, repository string, args ...string) (string, error) {
	output, err := commandOutput(ctx, repository, "git", args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func remoteTagCommit(output, tag string) string {
	var direct, peeled string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		switch fields[1] {
		case "refs/tags/" + tag:
			direct = fields[0]
		case "refs/tags/" + tag + "^{}":
			peeled = fields[0]
		}
	}
	if peeled != "" {
		return peeled
	}
	return direct
}
