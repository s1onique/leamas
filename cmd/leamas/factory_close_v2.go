package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure"
)

func runFactoryCloseRunV2(args []string, stdout, stderr io.Writer) int {
	fs := newCloseFlagSet("factory close run", stderr)
	var options closure.RunV2Options
	fs.StringVar(&options.PlanPath, "plan", "", "frozen closure plan")
	fs.StringVar(&options.Subject, "subject", "", "subject commit")
	fs.BoolVar(&options.JSONOutput, "json", false, "output JSON format")
	if err := parseCloseFlags(fs, args); err != nil || options.PlanPath == "" || options.Subject == "" {
		return reportCloseFlagError(stderr, "factory close run", err,
			"--plan and --subject are required for v2")
	}

	result, err := closure.RunClosureV2(context.Background(), options)
	if err != nil {
		return reportCloseError(stderr, "factory close run", err)
	}

	if options.JSONOutput {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, strings.ToUpper(result.Verdict))
	}

	fmt.Fprintf(stderr, "Closure Protocol v2: ACT=%s F=%s S=%s C=%s Tag=%s\n",
		result.ActID, trunc8(result.FreezeCommit), trunc8(result.SubjectCommit),
		trunc8(result.ClosureCommit), result.TagName)

	if result.Verdict != closure.VerdictPass {
		return closeFailureCode("verdict", "closure manifest verdict is fail")
	}
	return closeSuccessCode()
}
