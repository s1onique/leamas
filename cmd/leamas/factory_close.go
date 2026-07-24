package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure"
	factoryoutput "github.com/s1onique/leamas/internal/factory/output"
)

func handleFactoryClose() {
	os.Exit(runFactoryClose(os.Args[3:], os.Stdout, os.Stderr))
}

func runFactoryClose(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "factory close: missing subcommand")
		printFactoryCloseUsage(stderr)
		return closeFailureCode("usage", "missing subcommand")
	}
	switch args[0] {
	case "plan":
		return runFactoryClosePlan(args[1:], stdout, stderr)
	case "run":
		return runFactoryCloseRun(args[1:], stdout, stderr)
	case "verify":
		return runFactoryCloseVerify(args[1:], stdout, stderr)
	case "render":
		return runFactoryCloseRender(args[1:], stdout, stderr)
	case "tag":
		return runFactoryCloseTag(args[1:], stdout, stderr)
	case "status":
		return runFactoryCloseStatus(args[1:], stdout, stderr)
	case "chain":
		return runFactoryCloseChain(args[1:], stdout, stderr)
	case "attest":
		return runFactoryCloseAttest(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "factory close: unknown subcommand %q\n", args[0])
		printFactoryCloseUsage(stderr)
		return closeFailureCode("usage", "unknown subcommand")
	}
}

func runFactoryClosePlan(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "validate" {
		fmt.Fprintln(stderr, "factory close plan: expected validate")
		return closeFailureCode("usage", "expected validate")
	}
	fs := newCloseFlagSet("factory close plan validate", stderr)
	var file string
	fs.StringVar(&file, "file", "", "closure plan JSON file")
	if err := parseCloseFlags(fs, args[1:]); err != nil || file == "" {
		return reportCloseFlagError(stderr, "factory close plan validate", err, "--file is required")
	}
	if _, _, err := closure.LoadPlan(file); err != nil {
		return reportCloseError(stderr, "factory close plan validate", err)
	}
	fmt.Fprintln(stdout, "VALID")
	return closeSuccessCode()
}

func runFactoryCloseRun(args []string, stdout, stderr io.Writer) int {
	fs := newCloseFlagSet("factory close run", stderr)
	var options closure.RunOptions
	fs.StringVar(&options.PlanPath, "plan", "", "frozen closure plan")
	fs.StringVar(&options.PlanFreeze, "plan-freeze", "", "pre-subject plan freeze as <commit>:<path>")
	fs.StringVar(&options.Subject, "subject", "", "subject commit")
	fs.StringVar(&options.EvidenceDirectory, "evidence-dir", "", "absolute detached evidence directory")
	fs.StringVar(&options.ManifestOutput, "manifest-out", "", "absolute detached manifest output")
	if err := parseCloseFlags(fs, args); err != nil || options.PlanPath == "" || options.Subject == "" || options.EvidenceDirectory == "" || options.ManifestOutput == "" || options.PlanFreeze == "" {
		return reportCloseFlagError(stderr, "factory close run", err, "--plan, --plan-freeze, --subject, --evidence-dir, and --manifest-out are required")
	}
	manifest, _, err := closure.RunClosure(context.Background(), options)
	if err != nil {
		return reportCloseError(stderr, "factory close run", err)
	}
	fmt.Fprintln(stdout, strings.ToUpper(manifest.Verdict))
	if manifest.Verdict != closure.VerdictPass {
		return closeFailureCode("verdict", "closure manifest verdict is fail")
	}
	return closeSuccessCode()
}

func runFactoryCloseVerify(args []string, stdout, stderr io.Writer) int {
	fs := newCloseFlagSet("factory close verify", stderr)
	var manifestPath string
	fs.StringVar(&manifestPath, "manifest", "", "closure manifest")
	if err := parseCloseFlags(fs, args); err != nil || manifestPath == "" {
		return reportCloseFlagError(stderr, "factory close verify", err, "--manifest is required")
	}
	manifest, _, err := closure.VerifyManifestFile(".", manifestPath)
	if err != nil {
		return reportCloseError(stderr, "factory close verify", err)
	}
	fmt.Fprintln(stdout, strings.ToUpper(manifest.Verdict))
	return closeSuccessCode()
}

func runFactoryCloseRender(args []string, stdout, stderr io.Writer) int {
	fs := newCloseFlagSet("factory close render", stderr)
	var manifestPath, outputPath string
	fs.StringVar(&manifestPath, "manifest", "", "closure manifest")
	fs.StringVar(&outputPath, "output", "", "generated close report")
	if err := parseCloseFlags(fs, args); err != nil || manifestPath == "" || outputPath == "" {
		return reportCloseFlagError(stderr, "factory close render", err, "--manifest and --output are required")
	}
	if _, err := closure.RenderFile(".", manifestPath, outputPath); err != nil {
		return reportCloseError(stderr, "factory close render", err)
	}
	fmt.Fprintln(stdout, outputPath)
	return closeSuccessCode()
}

func newCloseFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func parseCloseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	return nil
}

func reportCloseFlagError(stderr io.Writer, command string, parseErr error, required string) int {
	if parseErr != nil {
		fmt.Fprintf(stderr, "%s: %v\n", command, parseErr)
	} else {
		fmt.Fprintf(stderr, "%s: %s\n", command, required)
	}
	return closeFailureCode("usage", required)
}

func reportCloseError(stderr io.Writer, command string, err error) int {
	fmt.Fprintf(stderr, "%s: %v\n", command, err)
	return closeFailureCode("failure", err.Error())
}

func closeSuccessCode() int {
	result := factoryoutput.NewResult("factory-close")
	result.SetOK()
	return result.ExitCode()
}

func closeFailureCode(code, message string) int {
	result := factoryoutput.NewResult("factory-close")
	result.AddFailure(code, message)
	return result.ExitCode()
}

func runFactoryCloseChain(args []string, stdout, stderr io.Writer) int {
	fs := newCloseFlagSet("factory close chain", stderr)
	var req closure.ChainValidationRequest
	var jsonFormat bool
	var manifestPath string
	fs.StringVar(&req.Freeze, "freeze", "", "freeze commit OID")
	fs.StringVar(&req.Subject, "subject", "", "subject commit OID")
	fs.StringVar(&req.Closure, "closure", "", "closure commit OID")
	fs.StringVar(&req.Tag, "tag", "", "tag name")
	fs.StringVar(&req.PlanPath, "plan-path", "", "plan path in repository")
	fs.StringVar(&manifestPath, "manifest", "", "manifest path")
	fs.BoolVar(&jsonFormat, "json", false, "output JSON format")
	if err := parseCloseFlags(fs, args); err != nil {
		return reportCloseFlagError(stderr, "factory close chain", err, "flag parsing failed")
	}
	// Require all chain fields
	if req.Freeze == "" {
		return reportCloseFlagError(stderr, "factory close chain", nil, "--freeze is required")
	}
	if req.Subject == "" {
		return reportCloseFlagError(stderr, "factory close chain", nil, "--subject is required")
	}
	if req.Closure == "" {
		return reportCloseFlagError(stderr, "factory close chain", nil, "--closure is required")
	}
	if req.PlanPath == "" {
		return reportCloseFlagError(stderr, "factory close chain", nil, "--plan-path is required")
	}
	if manifestPath == "" {
		return reportCloseFlagError(stderr, "factory close chain", nil, "--manifest is required")
	}
	if req.Tag == "" {
		return reportCloseFlagError(stderr, "factory close chain", nil, "--tag is required")
	}

	// Initialize repository authority
	git := closure.RealGit{}
	repoRoot, err := git.ShowToplevel(context.Background())
	if err != nil {
		return reportCloseError(stderr, "factory close chain", err)
	}

	// Load and validate manifest
	manifest, _, err := closure.LoadManifest(manifestPath)
	if err != nil {
		return reportCloseError(stderr, "factory close chain", err)
	}

	req.RepoRoot = repoRoot
	req.Git = git
	req.Manifest = &manifest

	result, err := closure.VerifyChain(context.Background(), req)
	if err != nil {
		return reportCloseError(stderr, "factory close chain", err)
	}
	result.Output(stdout, jsonFormat)
	if result.Verdict == "FAIL" {
		return closeFailureCode("chain", "chain validation failed")
	}
	return closeSuccessCode()
}

func runFactoryCloseAttest(args []string, stdout, stderr io.Writer) int {
	fs := newCloseFlagSet("factory close attest", stderr)
	var manifestPath, outputPath string
	var closureCommit string
	var jsonFormat bool
	fs.StringVar(&manifestPath, "manifest", "", "closure manifest")
	fs.StringVar(&outputPath, "output", "", "output attestation file")
	fs.StringVar(&closureCommit, "closure", "", "closure commit OID (distinct from subject)")
	fs.BoolVar(&jsonFormat, "json", false, "output JSON format")
	if err := parseCloseFlags(fs, args); err != nil || manifestPath == "" || outputPath == "" {
		return reportCloseFlagError(stderr, "factory close attest", err, "--manifest and --output are required")
	}

	// Load manifest with strict decoding
	manifest, _, err := closure.LoadManifest(manifestPath)
	if err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}

	// Require tag in manifest
	if manifest.Tag == "" {
		fmt.Fprintln(stderr, "factory close attest: manifest must include tag field")
		return closeFailureCode("manifest", "tag field required in manifest")
	}

	// Require pass verdict
	if manifest.Verdict != closure.VerdictPass {
		fmt.Fprintln(stderr, "factory close attest: manifest verdict must be pass")
		return closeFailureCode("verdict", "manifest verdict is fail")
	}

	// Build chain request for validation
	var realGit closure.RealGit
	repoRoot, err := realGit.ShowToplevel(context.Background())
	if err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}

	// Closure commit is distinct from subject (S != C for normal closure)
	if closureCommit == "" {
		fmt.Fprintln(stderr, "factory close attest: --closure is required (must be distinct from subject)")
		return closeFailureCode("usage", "--closure required and must differ from subject")
	}
	if closureCommit == manifest.Subject.CommitOID {
		fmt.Fprintln(stderr, "factory close attest: closure must differ from subject commit")
		return closeFailureCode("usage", "closure must differ from subject")
	}

	chainReq := closure.ChainValidationRequest{
		RepoRoot: repoRoot,
		Git:      realGit,
		Freeze:   manifest.PlanFreeze.FreezeCommit,
		Subject:  manifest.Subject.CommitOID,
		Closure:  closureCommit,
		PlanPath: manifest.Plan.Path,
		Tag:      manifest.Tag,
		Manifest: &manifest,
	}

	// Validate chain
	chainResult, err := closure.VerifyChain(context.Background(), chainReq)
	if err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}
	if chainResult.Verdict != "PASS" {
		for _, e := range chainResult.Errors {
			fmt.Fprintf(stderr, "factory close attest: %s\n", e)
		}
		return closeFailureCode("chain", "chain validation failed")
	}

	// Generate attestation with explicit closure commit
	attestReq := closure.AttestationRequest{
		RepoRoot:      repoRoot,
		Git:           realGit,
		Manifest:      manifest,
		ChainResult:   chainResult,
		ClosureCommit: closureCommit,
	}
	attest, err := closure.GenerateAttestation(context.Background(), attestReq)
	if err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}

	// Validate attestation
	if err := closure.ValidateAttestation(attest); err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}

	// Marshal attestation
	data, err := json.MarshalIndent(attest, "", "  ")
	if err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}

	// Robust atomic write using os.CreateTemp
	tmpFile, err := os.CreateTemp("", "attest-*.json")
	if err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}
	if err := tmpFile.Close(); err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return reportCloseError(stderr, "factory close attest", err)
	}

	fmt.Fprintln(stdout, outputPath)
	return closeSuccessCode()
}
