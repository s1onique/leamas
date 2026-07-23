package main

import (
	"context"
	"fmt"
	"io"

	"github.com/s1onique/leamas/internal/factory/closure"
)

func runFactoryCloseTag(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "create" {
		fmt.Fprintln(stderr, "factory close tag: expected create")
		return closeFailureCode("usage", "expected create")
	}
	fs := newCloseFlagSet("factory close tag create", stderr)
	var options closure.TagOptions
	fs.StringVar(&options.ManifestPath, "manifest", "", "committed closure manifest")
	fs.StringVar(&options.ReportPath, "report", "", "committed generated report")
	fs.StringVar(&options.TagName, "tag", "", "new immutable annotated tag")
	fs.StringVar(&options.Target, "target", "", "closure commit")
	if err := parseCloseFlags(fs, args[1:]); err != nil || options.ManifestPath == "" || options.ReportPath == "" || options.TagName == "" || options.Target == "" {
		return reportCloseFlagError(stderr, "factory close tag create", err, "--manifest, --report, --tag, and --target are required")
	}
	if _, err := closure.CreateTag(context.Background(), options); err != nil {
		return reportCloseError(stderr, "factory close tag create", err)
	}
	fmt.Fprintln(stdout, options.TagName)
	return closeSuccessCode()
}

func runFactoryCloseStatus(args []string, stdout, stderr io.Writer) int {
	fs := newCloseFlagSet("factory close status", stderr)
	var options closure.StatusOptions
	fs.StringVar(&options.ManifestPath, "manifest", "", "committed closure manifest")
	fs.StringVar(&options.ReportPath, "report", "", "committed generated report")
	fs.StringVar(&options.TagName, "tag", "", "immutable annotated tag")
	fs.StringVar(&options.Remote, "remote", "", "configured Git remote")
	if err := parseCloseFlags(fs, args); err != nil || options.ManifestPath == "" || options.ReportPath == "" || options.TagName == "" {
		return reportCloseFlagError(stderr, "factory close status", err, "--manifest, --report, and --tag are required")
	}
	result, err := closure.Status(context.Background(), options)
	if err != nil {
		return reportCloseError(stderr, "factory close status", err)
	}
	fmt.Fprintln(stdout, result.State)
	if result.RemoteDiagnostic != "" {
		fmt.Fprintln(stderr, result.RemoteDiagnostic)
	}
	return closeSuccessCode()
}

func printFactoryCloseUsage(writer io.Writer) {
	fmt.Fprintln(writer, "Closure Protocol v1 commands:")
	fmt.Fprintln(writer, "  leamas factory close plan validate --file <plan.json>")
	fmt.Fprintln(writer, "  leamas factory close run --plan <plan.json> --subject <commit> --evidence-dir <absolute-dir> --manifest-out <manifest.json>")
	fmt.Fprintln(writer, "  leamas factory close verify --manifest <manifest.json>")
	fmt.Fprintln(writer, "  leamas factory close render --manifest <manifest.json> --output <report.md>")
	fmt.Fprintln(writer, "  leamas factory close tag create --manifest <manifest.json> --report <report.md> --tag <name> --target <commit>")
	fmt.Fprintln(writer, "  leamas factory close status --manifest <manifest.json> --report <report.md> --tag <name> [--remote origin]")
}
