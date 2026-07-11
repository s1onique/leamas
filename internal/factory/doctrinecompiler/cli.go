package doctrinecompiler

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Common CLI flags.
//
// These constants document the public command surface and are reused
// by the help printer and the dispatcher. They are intentionally
// narrow: the surface is fixed by the ACT.
const (
	FlagTarget  = "--target"
	FlagProfile = "--profile"
)

// CLIOptions is the parsed argument bag shared by every command.
type CLIOptions struct {
	Target  string
	Profile ProfileId
}

// ParseFlags walks args looking for the supported flags. It returns
// the parsed options, the remaining positional arguments, and an error
// for any unknown flag or missing required value.
func ParseFlags(args []string) (CLIOptions, []string, error) {
	opts := CLIOptions{}
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case FlagTarget:
			if i+1 >= len(args) {
				return opts, rest, fmt.Errorf("missing value for %s", FlagTarget)
			}
			opts.Target = args[i+1]
			i++
		case FlagProfile:
			if i+1 >= len(args) {
				return opts, rest, fmt.Errorf("missing value for %s", FlagProfile)
			}
			opts.Profile = ProfileId(args[i+1])
			i++
		default:
			if strings.HasPrefix(a, "--") {
				return opts, rest, fmt.Errorf("unknown flag: %s", a)
			}
			rest = append(rest, a)
		}
	}
	return opts, rest, nil
}

// CLIError prints an error message to stderr and returns a non-zero
// exit code via the supplied exit function.
func CLIError(w io.Writer, msg string, exit func(int)) {
	fmt.Fprintln(w, msg)
	if exit != nil {
		exit(1)
	}
}

// ResolveTarget validates the CLI target flag and creates a Resolver.
// Defaults to the current working directory when --target is omitted.
func ResolveTarget(target string) (*Resolver, error) {
	if target == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, newError("validate", "target",
				"could not determine working directory: "+err.Error())
		}
		target = cwd
	}
	return NewResolver(target)
}

// PrintPlan writes a deterministic text plan to w.
func PrintPlan(w io.Writer, plan ProjectionPlan) {
	_, _ = w.Write(FormatPlan(plan))
}

// PrintExplain writes a deterministic text explain report to w.
func PrintExplain(w io.Writer, rep ExplainReport) {
	_, _ = w.Write(FormatExplain(rep))
}

// PrintVerify writes the verify result to w. Failures are written to
// stderr to match the rest of the Leamas output contract.
func PrintVerify(w, errW io.Writer, result VerifyResult) {
	if result.OK {
		fmt.Fprintln(w, "doctrine verify: OK")
		return
	}
	fmt.Fprintln(w, "doctrine verify: FAIL")
	for _, f := range result.Findings {
		fmt.Fprintf(errW, "  %s: %s: %s\n", f.Path, f.Kind, f.Message)
	}
}
