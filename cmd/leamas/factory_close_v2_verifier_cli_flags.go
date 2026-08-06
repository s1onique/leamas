// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_cli_flags.go isolates the flag
// parser (parseV2VerifierFlags, scalar flag name set, and
// duplicate detection) from the command dispatcher in
// factory_close_v2_verifier_cli.go. Splitting along the
// flag-handling boundary keeps each file under the LLM-
// friendliness 400-line threshold.

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// v2VerifierScalarFlagNames is the closed set of scalar
// flags the duplicate-detection pass walks over. Each name
// is the canonical CLI flag without the leading "--".
// The parser rejects repeated occurrences of any name in
// the list (including repeats with the same value).
var v2VerifierScalarFlagNames = []string{
	"repository",
	"protocol-version",
	"plan-contract-version",
	"subject",
	"freeze",
	"closure",
	"plan-path",
	"manifest-path",
	"working-manifest-assertion",
	"expected-tag",
	"output",
}

// detectDuplicateV2VerifierFlags scans args for repeated
// occurrences of any scalar flag and returns a typed
// *V2VerifierError with code V2VerifierDuplicateCLIFlag
// naming the first duplicate. The check runs BEFORE
// flag.Parse so the duplicate surfaces before any Git
// observation.
func detectDuplicateV2VerifierFlags(args []string) error {
	seen := make(map[string]int, len(args))
	for _, a := range args {
		name, isFlag := stripV2VerifierFlagName(a)
		if !isFlag {
			continue
		}
		if !isTrackedV2VerifierFlag(name) {
			continue
		}
		seen[name]++
		if seen[name] > 1 {
			return closure.NewV2VerifierError(closure.V2VerifierDiagnostic{
				Code:         closure.V2VerifierDuplicateCLIFlag,
				Message:      fmt.Sprintf("duplicate flag: --%s", name),
				PropertyName: "flag",
			})
		}
	}
	return nil
}

// stripV2VerifierFlagName returns the canonical flag
// name and true when arg is a --flag or --flag=value
// token. The function tolerates "-flag" as a synonym for
// "--flag" to match Go's flag package conventions; a
// positional or non-flag argument returns ("", false).
func stripV2VerifierFlagName(arg string) (string, bool) {
	if arg == "" {
		return "", false
	}
	if !strings.HasPrefix(arg, "-") {
		return "", false
	}
	trimmed := strings.TrimLeft(arg, "-")
	if trimmed == "" {
		return "", false
	}
	if eq := strings.IndexByte(trimmed, '='); eq >= 0 {
		trimmed = trimmed[:eq]
	}
	return trimmed, true
}

// isTrackedV2VerifierFlag reports whether name is in
// the closed set of scalar flags the duplicate detector
// tracks. The detector is intentionally narrow: a stray
// unknown flag will be caught by flag.Parse below.
func isTrackedV2VerifierFlag(name string) bool {
	for _, n := range v2VerifierScalarFlagNames {
		if n == name {
			return true
		}
	}
	return false
}

// parseV2VerifierFlags parses the command arguments. The
// function returns a fully populated request or a typed
// usage error.
func parseV2VerifierFlags(name string, stderr io.Writer, args []string) (v2VerifierParsedInput, error) {
	if err := detectDuplicateV2VerifierFlags(args); err != nil {
		return v2VerifierParsedInput{}, err
	}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)

	var in v2VerifierParsedInput
	fs.StringVar(&in.Request.RepositoryRoot, "repository", "", "Git repository root (absolute path)")
	fs.StringVar((*string)(&in.Request.ClosureProtocolVersion), "protocol-version", string(closure.ClosureProtocolV2),
		"closure protocol version (only 2 is accepted)")
	fs.IntVar((*int)(&in.Request.PlanContractVersion), "plan-contract-version", int(closure.PlanContractV1),
		"plan contract version (only 1 is accepted)")
	fs.StringVar(&in.Request.SubjectCommit, "subject", "", "subject S commit OID")
	fs.StringVar(&in.Request.FreezeCommit, "freeze", "", "freeze F commit OID")
	fs.StringVar(&in.Request.ClosureCommit, "closure", "", "closure C commit OID")
	fs.StringVar(&in.Request.PlanPath, "plan-path", "", "repository-relative plan path P")
	fs.StringVar(&in.Request.ManifestPath, "manifest-path", "", "repository-relative manifest path M")
	fs.StringVar(&in.WorkingManifestPath, "working-manifest-assertion", "",
		"optional path whose bytes must match C:M (non-authoritative)")
	fs.StringVar(&in.ExpectedTag, "expected-tag", "",
		"optional annotated-tag name that must target C")
	fs.StringVar(&in.OutputPath, "output", "",
		"optional path for the structured text summary; MUST live outside --repository")
	fs.BoolVar(&in.JSONOutput, "json", false, "emit exactly one JSON document on stdout")
	fs.BoolVar(&in.CaptureCallerState, "capture-caller-state", false,
		"capture pre/post caller state (test surface)")
	fs.Bool("help", false, "print help and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return v2VerifierParsedInput{}, flag.ErrHelp
		}
		return v2VerifierParsedInput{}, fmt.Errorf("%s: %w", name, err)
	}
	if fs.NArg() != 0 {
		return v2VerifierParsedInput{}, fmt.Errorf("%s: unexpected arguments: %v", name, fs.Args())
	}

	helpFlag := fs.Lookup("help")
	if helpFlag != nil && helpFlag.Value.String() == "true" {
		return v2VerifierParsedInput{}, flag.ErrHelp
	}

	if string(in.Request.ClosureProtocolVersion) != string(closure.ClosureProtocolV2) {
		return v2VerifierParsedInput{}, fmt.Errorf("--protocol-version must be %q, got %q",
			closure.ClosureProtocolV2, in.Request.ClosureProtocolVersion)
	}
	if int(in.Request.PlanContractVersion) != int(closure.PlanContractV1) {
		return v2VerifierParsedInput{}, fmt.Errorf("--plan-contract-version must be %d, got %d",
			int(closure.PlanContractV1), int(in.Request.PlanContractVersion))
	}

	required := []struct {
		name, value string
	}{
		{"--repository", in.Request.RepositoryRoot},
		{"--subject", in.Request.SubjectCommit},
		{"--freeze", in.Request.FreezeCommit},
		{"--closure", in.Request.ClosureCommit},
		{"--plan-path", in.Request.PlanPath},
		{"--manifest-path", in.Request.ManifestPath},
	}
	for _, field := range required {
		if field.value == "" {
			return v2VerifierParsedInput{}, fmt.Errorf("%s is required", field.name)
		}
	}

	if err := closure.ValidateDetachedVerifierOutputPath(in.Request.RepositoryRoot, in.OutputPath); err != nil {
		return v2VerifierParsedInput{}, err
	}

	if in.WorkingManifestPath != "" {
		bytes, err := os.ReadFile(in.WorkingManifestPath)
		if err != nil {
			return v2VerifierParsedInput{}, fmt.Errorf("--working-manifest-assertion: %w", err)
		}
		in.Request.OptionalManifestAssertion = bytes
	}
	in.Request.ExpectedTagName = in.ExpectedTag
	return in, nil
}
