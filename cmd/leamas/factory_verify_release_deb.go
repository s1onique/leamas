package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/s1onique/leamas/internal/factory/releasedeb"
)

type repeatedStrings []string

func (values *repeatedStrings) String() string {
	return fmt.Sprint([]string(*values))
}

func (values *repeatedStrings) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type releaseDebOptions struct {
	packagePath string
	binaryPath  string
	version     string
	arch        string
	commit      string
	licenseFile string
	license     string
	goos        string
	goarch      string
	nfpmVersion string
	outputPath  string
	repository  string
	remote      string
	tag         string
	assets      repeatedStrings
}

func handleFactoryVerifyReleaseDeb() {
	if err := runReleaseDebVerifier(context.Background(), os.Args[4:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "release-deb verification FAILED: %v\n", err)
		os.Exit(1)
	}
}

func runReleaseDebVerifier(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("mode is required: preflight, inspect, verify, install-smoke, checksum, or publication")
	}
	mode := args[0]
	options, err := parseReleaseDebOptions(mode, args[1:])
	if err != nil {
		return err
	}
	config := releasedeb.Config{
		PackagePath:   options.packagePath,
		ReleaseBinary: options.binaryPath,
		Version:       options.version,
		Architecture:  options.arch,
		Commit:        options.commit,
		LicenseFile:   options.licenseFile,
		License:       options.license,
	}

	switch mode {
	case "preflight":
		if options.nfpmVersion != releasedeb.ExpectedNFPMVersion {
			return fmt.Errorf("nFPM must be pinned to %s (got %q)", releasedeb.ExpectedNFPMVersion, options.nfpmVersion)
		}
		return releasedeb.ValidateBuildInputs(options.version, options.goos, options.goarch,
			options.licenseFile, options.license)
	case "inspect":
		return config.Inspect(ctx, out)
	case "verify":
		return config.Verify(ctx, out)
	case "install-smoke":
		return config.InstallSmoke(ctx, out)
	case "checksum":
		if err := releasedeb.WriteChecksum(options.packagePath, options.outputPath); err != nil {
			return err
		}
		return releasedeb.VerifyChecksum(ctx, options.packagePath, options.outputPath, out)
	case "publication":
		return releasedeb.CheckPublication(ctx, releasedeb.PublicationConfig{
			Repository: options.repository,
			Remote:     options.remote,
			Tag:        options.tag,
			Assets:     options.assets,
		})
	default:
		return fmt.Errorf("unknown release-deb mode %q", mode)
	}
}

func parseReleaseDebOptions(mode string, args []string) (releaseDebOptions, error) {
	var options releaseDebOptions
	flags := flag.NewFlagSet("release-deb "+mode, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&options.packagePath, "package", "", "Debian package path")
	flags.StringVar(&options.binaryPath, "binary", "", "canonical release binary path")
	flags.StringVar(&options.version, "version", "", "strict stable SemVer")
	flags.StringVar(&options.arch, "arch", releasedeb.ExpectedArchitecture, "Debian architecture")
	flags.StringVar(&options.commit, "commit", "", "expected binary commit stamp")
	flags.StringVar(&options.licenseFile, "license-file", "", "repository license file")
	flags.StringVar(&options.license, "license", "", "SPDX license identifier")
	flags.StringVar(&options.goos, "goos", "", "release GOOS")
	flags.StringVar(&options.goarch, "goarch", "", "release GOARCH")
	flags.StringVar(&options.nfpmVersion, "nfpm-version", "", "pinned nFPM version")
	flags.StringVar(&options.outputPath, "output", "", "SHA256SUMS path")
	flags.StringVar(&options.repository, "repo", ".", "Git repository")
	flags.StringVar(&options.remote, "remote", "origin", "Git remote")
	flags.StringVar(&options.tag, "tag", "", "release tag")
	flags.Var(&options.assets, "asset", "release asset path; repeat for each asset")
	if err := flags.Parse(args); err != nil {
		return options, fmt.Errorf("parse release-deb %s flags: %w", mode, err)
	}
	if flags.NArg() != 0 {
		return options, fmt.Errorf("unexpected release-deb %s arguments: %v", mode, flags.Args())
	}
	return options, nil
}
