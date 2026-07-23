package main

import (
	"context"
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
