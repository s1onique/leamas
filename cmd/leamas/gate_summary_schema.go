package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/s1onique/leamas/internal/gatesummary/schema"
)

// gateSummarySchemaCLI wires the embedded Gate Summary schema
// registry to the CLI command surface. The command is deliberately
// narrow: it lists supported versions and prints the byte-exact
// embedded schema for a named version. It does not validate, parse,
// or normalize Gate Summary documents.
//
// The CLI surface:
//
//	leamas gate-summary schema list
//	leamas gate-summary schema show <version>
//
// Writers are injected so tests can capture stdout and stderr without
// mutating os.Stdout / os.Stderr.
//
// Write-failure contract: each command makes one exact-byte write
// attempt through the checked schema.WriteExact helper. A short write
// or writer error returns a non-zero exit code. A failing destination
// MAY observe a prefix of the bytes before reporting failure; the
// contract does not promise atomic output to hostile destinations.
type gateSummarySchemaCLI struct {
	stdout io.Writer
	stderr io.Writer
}

// newGateSummarySchemaCLI constructs a CLI bound to the given writers.
// It is the only constructor used by both the live main() path and
// the tests.
func newGateSummarySchemaCLI(stdout, stderr io.Writer) *gateSummarySchemaCLI {
	return &gateSummarySchemaCLI{stdout: stdout, stderr: stderr}
}

// runGateSummarySchema dispatches the `leamas gate-summary schema`
// subcommand. It returns the integer exit code the caller should
// propagate to the process.
func runGateSummarySchema(args []string) int {
	return newGateSummarySchemaCLI(os.Stdout, os.Stderr).Run(args)
}

// Run is the dispatch entry point. The slice is the argv remainder
// after `leamas gate-summary schema`.
func (c *gateSummarySchemaCLI) Run(args []string) int {
	// Help flags win at any position so the CLI matches the
	// surface convention: `leamas gate-summary schema --help`,
	// `leamas gate-summary schema show --help`, and
	// `leamas gate-summary schema help` all print the same text.
	for _, a := range args {
		if a == "--help" || a == "-h" {
			printGateSummarySchemaUsageTo(c.stdout)
			return 0
		}
	}
	if len(args) == 0 {
		printGateSummarySchemaUsageTo(c.stderr)
		return 1
	}
	switch args[0] {
	case "list":
		if len(args) > 1 {
			fmt.Fprintf(c.stderr, "gate-summary schema list: unexpected argument %q\n", args[1])
			return 1
		}
		return c.runList()
	case "show":
		return c.runShow(args[1:])
	case "help":
		printGateSummarySchemaUsageTo(c.stdout)
		return 0
	default:
		fmt.Fprintf(c.stderr, "gate-summary schema: unknown subcommand %q\n", args[0])
		printGateSummarySchemaUsageTo(c.stderr)
		return 1
	}
}

// runList prints the supported-version table to stdout. The full
// table is rendered into a local buffer first, then written through
// the checked schema.WriteExact helper so prefix-write failures are
// detected. The contract does not promise atomic output to hostile
// destinations: a failing writer may observe a prefix before reporting
// failure.
func (c *gateSummarySchemaCLI) runList() int {
	// Column width and padding lock the wire-format output. The header
	// uses the same column widths as the data rows so downstream
	// consumers can grep or awk the table.
	var buf bytes.Buffer
	buf.WriteString("VERSION  STATUS     SCHEMA_ID\n")
	for _, d := range schema.List() {
		fmt.Fprintf(&buf, "%-7s  %-9s  %s\n", string(d.Version), string(d.Status), d.SchemaID)
	}
	if err := schema.WriteExact(c.stdout, buf.Bytes()); err != nil {
		fmt.Fprintf(c.stderr, "gate-summary schema list: %v\n", err)
		return 1
	}
	return 0
}

// runShow prints the exact embedded schema for the requested version
// to stdout. It rejects mutable aliases, unknown versions, and wrong
// argument counts with a non-zero exit code and a diagnostic on stderr.
func (c *gateSummarySchemaCLI) runShow(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(c.stderr, "gate-summary schema show: missing version")
		printGateSummarySchemaShowUsageTo(c.stderr)
		return 1
	}
	if len(args) > 1 {
		fmt.Fprintf(c.stderr, "gate-summary schema show: unexpected extra arguments after %q\n", args[0])
		return 1
	}
	raw := args[0]
	if isMutableVersionAlias(raw) {
		fmt.Fprintf(c.stderr, "gate-summary schema show: %q is a mutable alias; specify an explicit version (v1 or v2)\n", raw)
		return 1
	}
	version := schema.Version(raw)
	if err := schema.Write(version, c.stdout); err != nil {
		if schema.IsUnknownVersion(err) {
			fmt.Fprintf(c.stderr, "gate-summary schema show: unknown version %q (allowed: v1, v2)\n", raw)
			return 1
		}
		// All other failures are operational write failures. Report
		// them on stderr and return a non-zero code. The destination
		// may have observed a prefix of the schema bytes before
		// reporting failure; the contract does not guarantee atomic
		// output to hostile destinations.
		fmt.Fprintf(c.stderr, "gate-summary schema show: %v\n", err)
		return 1
	}
	return 0
}

// isMutableVersionAlias reports whether the user requested a mutable
// alias that the contract explicitly rejects. The set is closed:
// "latest", "current", "stable", "default", and the empty string.
func isMutableVersionAlias(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "latest", "current", "stable", "default", "":
		return true
	}
	return false
}

// printGateSummarySchemaUsageTo prints the surface-level help text to
// the given writer. The text is identical for stdout and stderr; the
// caller chooses the stream.
func printGateSummarySchemaUsageTo(w io.Writer) {
	fmt.Fprintln(w, "Usage: leamas gate-summary schema <subcommand> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  list                List supported schema versions")
	fmt.Fprintln(w, "  show <version>      Print the exact embedded JSON Schema for <version>")
	fmt.Fprintln(w, "  --help, -h          Show this help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Allowed versions: v1, v2")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The emitted JSON Schema describes the Gate Summary wire format.")
	fmt.Fprintln(w, "Decode, normalization, lifecycle, diagnostic, and digest semantics remain")
	fmt.Fprintln(w, "defined by the executable Leamas Gate Summary contract.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  leamas gate-summary schema list")
	fmt.Fprintln(w, "  leamas gate-summary schema show v1")
	fmt.Fprintln(w, "  leamas gate-summary schema show v2 > gate-summary-v2.schema.json")
}

// printGateSummarySchemaShowUsageTo prints the help text for the
// `show` subcommand. It is separate from the surface help so error
// output can point users at the narrower context.
func printGateSummarySchemaShowUsageTo(w io.Writer) {
	fmt.Fprintln(w, "Usage: leamas gate-summary schema show <version>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Allowed versions: v1, v2")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The emitted JSON Schema describes the Gate Summary wire format.")
	fmt.Fprintln(w, "Decode, normalization, lifecycle, diagnostic, and digest semantics remain")
	fmt.Fprintln(w, "defined by the executable Leamas Gate Summary contract.")
}

// printGateSummarySchemaUsage prints the surface help to stderr. It
// is the standard entry point used by main() when the command is
// invoked without arguments.
func printGateSummarySchemaUsage() {
	printGateSummarySchemaUsageTo(os.Stderr)
}
