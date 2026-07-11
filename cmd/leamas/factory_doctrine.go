package main

import (
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/factory/doctrinecompiler"
	"github.com/s1onique/leamas/internal/version"
)

// init wires the compiler version source so the verifier can run
// the real compatibility check against the current build identity.
func init() {
	ver := version.Get()
	doctrinecompiler.SetCompilerVersionSource(func() string { return ver.Version })
}

// doctrineSubcommands enumerates the supported `leamas factory doctrine`
// subcommands.
var doctrineSubcommands = []string{"plan", "compile", "verify", "explain"}

// handleFactoryDoctrine dispatches the four doctrine subcommands.
func handleFactoryDoctrine() {
	if len(os.Args) < 4 {
		printFactoryDoctrineUsage()
		os.Exit(1)
	}
	sub := os.Args[3]
	switch sub {
	case "plan":
		runDoctrinePlan(os.Args[4:])
	case "compile":
		runDoctrineCompile(os.Args[4:])
	case "verify":
		runDoctrineVerify(os.Args[4:])
	case "explain":
		runDoctrineExplain(os.Args[4:])
	case "--help", "-h":
		printFactoryDoctrineUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown doctrine subcommand: %s\n", sub)
		printFactoryDoctrineUsage()
		os.Exit(1)
	}
}

// resolveProfile loads the (pack, profile) tuple from the explicit
// CLI flag or, when absent, from the committed selector. The boolean
// fallback arg toggles selector fallback behaviour.
//
// plan and compile require an explicit --profile; verify and explain
// fall back to the selector.
func resolveProfile(explicit string, target string, fallback bool) (string, doctrinecompiler.ProfileId, error) {
	if explicit != "" {
		return "", doctrinecompiler.ProfileId(explicit), nil
	}
	if !fallback {
		return "", "", fmt.Errorf("--profile is required for this command")
	}
	return doctrinecompiler.ResolveSelection("", "", target, true)
}

// runDoctrinePlan implements `leamas factory doctrine plan`.
func runDoctrinePlan(args []string) {
	opts, rest, err := doctrinecompiler.ParseFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine plan: %v\n", err)
		os.Exit(2)
	}
	rejectExtraPositional(rest)
	if opts.Profile == "" {
		fmt.Fprintln(os.Stderr, "doctrine plan: --profile is required")
		os.Exit(2)
	}
	target, err := doctrinecompiler.ResolveTarget(opts.Target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine plan: %v\n", err)
		os.Exit(1)
	}
	pack, err := doctrinecompiler.LoadCorePack()
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine plan: %v\n", err)
		os.Exit(1)
	}
	profile, err := pack.MustProfile(opts.Profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine plan: %v\n", err)
		os.Exit(1)
	}
	if err := doctrinecompiler.CheckCompilerCompatibility(pack.CompilerVersion, version.Get().Version); err != nil {
		fmt.Fprintf(os.Stderr, "doctrine plan: %v\n", err)
		os.Exit(1)
	}
	planOpts := doctrinecompiler.PlannerOptions{
		Providers:       doctrinecompiler.CoreContentProviders(),
		CompilerVersion: version.Get().Version,
	}
	plan, err := doctrinecompiler.PlanWithOptions(pack, profile, target.Root, planOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine plan: %v\n", err)
		os.Exit(1)
	}
	doctrinecompiler.PrintPlan(os.Stdout, plan)
	// If the plan contains rejects, fail closed so callers can
	// distinguish safe plans from unsafe ones.
	for _, a := range plan.Actions {
		if a.Class == doctrinecompiler.ActionReject {
			os.Exit(1)
		}
	}
}

// runDoctrineCompile implements `leamas factory doctrine compile`.
func runDoctrineCompile(args []string) {
	opts, rest, err := doctrinecompiler.ParseFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine compile: %v\n", err)
		os.Exit(2)
	}
	rejectExtraPositional(rest)
	if opts.Profile == "" {
		fmt.Fprintln(os.Stderr, "doctrine compile: --profile is required")
		os.Exit(2)
	}
	target, err := doctrinecompiler.ResolveTarget(opts.Target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine compile: %v\n", err)
		os.Exit(1)
	}
	pack, err := doctrinecompiler.LoadCorePack()
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine compile: %v\n", err)
		os.Exit(1)
	}
	profile, err := pack.MustProfile(opts.Profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine compile: %v\n", err)
		os.Exit(1)
	}
	if err := doctrinecompiler.CheckCompilerCompatibility(pack.CompilerVersion, version.Get().Version); err != nil {
		fmt.Fprintf(os.Stderr, "doctrine compile: %v\n", err)
		os.Exit(1)
	}
	ver := version.Get()
	copts := doctrinecompiler.CompilerOptions{
		CompilerVersion: ver.Version,
		CompilerCommit:  ver.Commit,
	}
	if _, err := doctrinecompiler.Compile(pack, profile, target.Root, copts); err != nil {
		fmt.Fprintf(os.Stderr, "doctrine compile: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "doctrine compile: OK")
}

// runDoctrineVerify implements `leamas factory doctrine verify`.
//
// When --profile is absent, the verifier loads the committed selector
// from .factory/project.json. This makes the generated make factorize
// target self-contained.
func runDoctrineVerify(args []string) {
	opts, rest, err := doctrinecompiler.ParseFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine verify: %v\n", err)
		os.Exit(2)
	}
	rejectExtraPositional(rest)
	target, err := doctrinecompiler.ResolveTarget(opts.Target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine verify: %v\n", err)
		os.Exit(1)
	}
	_, profileID, err := resolveProfile(string(opts.Profile), target.Root, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine verify: %v\n", err)
		os.Exit(1)
	}
	pack, err := doctrinecompiler.LoadCorePack()
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine verify: %v\n", err)
		os.Exit(1)
	}
	profile, err := pack.MustProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine verify: %v\n", err)
		os.Exit(1)
	}
	result, err := doctrinecompiler.Verify(pack, profile, target.Root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine verify: %v\n", err)
		os.Exit(1)
	}
	doctrinecompiler.PrintVerify(os.Stdout, os.Stderr, result)
	if !result.OK {
		os.Exit(1)
	}
}

// runDoctrineExplain implements `leamas factory doctrine explain`.
//
// The profile is inferred from the committed selector unless the
// caller supplies --profile.
func runDoctrineExplain(args []string) {
	opts, rest, err := doctrinecompiler.ParseFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine explain: %v\n", err)
		os.Exit(2)
	}
	rejectExtraPositional(rest)
	target, err := doctrinecompiler.ResolveTarget(opts.Target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine explain: %v\n", err)
		os.Exit(1)
	}
	_, profileID, err := resolveProfile(string(opts.Profile), target.Root, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine explain: %v\n", err)
		os.Exit(1)
	}
	pack, err := doctrinecompiler.LoadCorePack()
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine explain: %v\n", err)
		os.Exit(1)
	}
	profile, err := pack.MustProfile(profileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine explain: %v\n", err)
		os.Exit(1)
	}
	ver := version.Get()
	rep, err := doctrinecompiler.Explain(pack, profile, target.Root,
		ver.Version, ver.Commit, ver.Commit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctrine explain: %v\n", err)
		os.Exit(1)
	}
	doctrinecompiler.PrintExplain(os.Stdout, rep)
}

// rejectExtraPositional returns an error if any positional argument
// was passed beyond the supported flags.
func rejectExtraPositional(rest []string) {
	if len(rest) > 0 {
		fmt.Fprintf(os.Stderr, "unexpected positional arguments: %v\n", rest)
		os.Exit(2)
	}
}

// printFactoryDoctrineUsage prints the doctrine subcommand help.
func printFactoryDoctrineUsage() {
	fmt.Println("Factory doctrine commands:")
	fmt.Println("  leamas factory doctrine plan     --profile <id> [--target <path>]")
	fmt.Println("  leamas factory doctrine compile  --profile <id> [--target <path>]")
	fmt.Println("  leamas factory doctrine verify   [--profile <id>] [--target <path>]")
	fmt.Println("  leamas factory doctrine explain  [--profile <id>] [--target <path>]")
	fmt.Println("")
	fmt.Println("verify and explain infer the profile from .factory/project.json when --profile is omitted.")
}
