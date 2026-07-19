package releasedeb

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// InstallSmoke installs through apt, checks the installed command and stamps,
// executes a safe command, and removes the package before returning.
func (c Config) InstallSmoke(ctx context.Context, out io.Writer) error {
	if err := c.Inspect(ctx, out); err != nil {
		return err
	}
	if _, err := commandOutput(ctx, "", "dpkg-query", "-W", "-f=${Status}\n", "leamas"); err == nil {
		return fmt.Errorf("leamas is already installed; refusing to replace an existing installation")
	}

	installed := false
	defer func() {
		if installed {
			_, _ = commandOutput(ctx, "", "sudo", "apt-get", "remove", "-y", "leamas")
		}
	}()
	if err := printCommand(ctx, out, "", "sudo", "apt-get", "install", "-y", packageInputPath(c.PackagePath)); err != nil {
		return err
	}
	installed = true

	pathOutput, err := commandOutputWithPath(ctx, "", "/usr/bin:/usr/sbin:/bin:/sbin", "sh", "-c", "test \"$(command -v leamas)\" = \"/usr/bin/leamas\"")
	if err != nil {
		return fmt.Errorf("installed command is not /usr/bin/leamas: %w", err)
	}
	_, _ = out.Write(pathOutput)
	status, err := commandOutput(ctx, "", "dpkg-query", "-W", "-f=${Status}\n", "leamas")
	if err != nil || strings.TrimSpace(string(status)) != "install ok installed" {
		return fmt.Errorf("package status is not install ok installed: %q", strings.TrimSpace(string(status)))
	}
	architecture, err := commandOutput(ctx, "", "dpkg-query", "-W", "-f=${Architecture}\n", "leamas")
	if err != nil || strings.TrimSpace(string(architecture)) != ExpectedArchitecture {
		return fmt.Errorf("installed package architecture is not amd64: %q", strings.TrimSpace(string(architecture)))
	}

	versionOutput, err := commandOutput(ctx, "", "/usr/bin/leamas", "version")
	if err != nil {
		return fmt.Errorf("installed leamas version failed: %w", err)
	}
	if err := verifyVersionOutput(string(versionOutput), c.Version, c.Commit); err != nil {
		return err
	}
	if err := verifyStaticAMD64("/usr/bin/leamas"); err != nil {
		return fmt.Errorf("installed binary is not static amd64: %w", err)
	}
	if err := printCommand(ctx, out, "", "/usr/bin/leamas", "doctor"); err != nil {
		return fmt.Errorf("installed non-mutating command failed: %w", err)
	}
	if err := printCommand(ctx, out, "", "sudo", "apt-get", "remove", "-y", "leamas"); err != nil {
		return err
	}
	installed = false
	if _, err := os.Stat("/usr/bin/leamas"); !os.IsNotExist(err) {
		return fmt.Errorf("/usr/bin/leamas remains after removal")
	}
	return nil
}

func verifyVersionOutput(output, expectedVersion, expectedCommit string) error {
	values := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(line, ": ")
		if ok {
			values[key] = value
		}
	}
	if values["version"] != expectedVersion {
		return fmt.Errorf("installed binary version mismatch: got %q, want %q", values["version"], expectedVersion)
	}
	if expectedCommit != "" && values["commit"] != expectedCommit {
		return fmt.Errorf("installed binary commit mismatch: got %q, want %q", values["commit"], expectedCommit)
	}
	if values["build_time"] == "" || values["build_time"] == "unknown" {
		return fmt.Errorf("installed binary has no build-time stamp")
	}
	return nil
}
