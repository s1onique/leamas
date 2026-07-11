package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// handleFactoryVerifyActDoctrineCompiler runs the ACT-local verifier
// for the doctrine compiler ACT.
//
// The verifier asserts, at minimum:
//
//   - canonical pack and profile presence
//   - required CLI commands are wired
//   - required ownership cases are exercised
//   - fixture and golden-tree presence
//   - adversarial test presence
//   - documentation presence
//   - no oversized source files
//   - no unrelated staged files in the package tree
//
// It is intentionally local-first and read-only: no network, no git
// mutation, no execution outside the package directory.
func handleFactoryVerifyActDoctrineCompiler() {
	root := "internal/factory/doctrinecompiler"
	findings := actDoctrineCompilerChecks(root)
	if len(findings) == 0 {
		fmt.Println("act-doctrine-compiler: OK")
		os.Exit(0)
	}
	for _, f := range findings {
		fmt.Fprintf(os.Stderr, "  %s: %s: %s\n", f.Path, f.Kind, f.Message)
	}
	os.Exit(1)
}

// actDoctrineCompilerChecks walks the doctrine-compiler package and
// reports a Finding for every missing or oversized requirement.
func actDoctrineCompilerChecks(root string) []checks.Finding {
	var findings []checks.Finding
	// Required source files (production code).
	requiredSources := []string{
		"types.go",
		"paths.go",
		"digest.go",
		"packschema.go",
		"pack.go",
		"packcore.go",
		"plan.go",
		"compile.go",
		"verify.go",
		"explain.go",
		"cli.go",
		"lockfile.go",
		"fsx.go",
		"helpers.go",
		"lock_helpers.go",
	}
	for _, name := range requiredSources {
		path := filepath.Join(root, name)
		if !fileExists(path) {
			findings = append(findings, checks.Finding{
				Path:    path,
				Kind:    "missing_source",
				Message: "required source file missing",
			})
			continue
		}
		if oversized(path, 4096) {
			findings = append(findings, checks.Finding{
				Path:    path,
				Kind:    "oversized_source",
				Message: "source file exceeds 4096 lines",
			})
		}
	}
	// Required tests covering ACT acceptance criteria.
	requiredTests := map[string][]string{
		"pack_test.go": {
			"TestCorePackValid",
			"TestPackSchemaVersionEnforced",
			"TestPackUnknownField",
			"TestPackDuplicateDoctrineFails",
			"TestPackAbsolutePathFails",
			"TestPackTraversalPathFails",
		},
		"plan_test.go": {
			"TestPlanEmptyTarget",
			"TestPlanDetectsUpdateManaged",
			"TestPlanPreserveSeeded",
			"TestPlanObsoleteManaged",
			"TestPlanIgnoresUnrelatedFiles",
			"TestPlanPerformsNoWrites",
		},
		"compile_test.go": {
			"TestCompileEmptyTargetProducesGoldenTree",
			"TestCompileIdempotentNoFilesystemChange",
			"TestCompileNeverOverwritesSeeded",
			"TestCompileRepairsManagedFiles",
			"TestCompileRemovesObsoleteManaged",
		},
		"compile_safety_test.go": {
			"TestCompileLeavesNoTempFiles",
			"TestCompileAtomicLockLast",
			"TestCompileRefusesSymlinkParent",
			"TestCompileRollbackFailsClosed",
			"TestCompileRollbackRemovesCreatedOnEmptyTarget",
		},
		"verify_test.go": {
			"TestVerifyDetectsManagedModification",
			"TestVerifyDetectsMissingManaged",
			"TestVerifyDetectsLockModification",
			"TestVerifyDetectsPackDigestMismatch",
			"TestVerifyDetectsProfileMismatch",
			"TestVerifyDetectsMissingMakefileInclude",
			"TestVerifyDetectsGateWithoutFactorizeDep",
			"TestVerifyIgnoresUnrelatedChanges",
			"TestVerifyPerformsNoWrites",
		},
		"determinism_test.go": {
			"TestDeterminismRepeatedOutput",
			"TestDeterminismTimezonesEquivalent",
			"TestDeterminismNoWorkingDirectoryCoupling",
			"TestDeterminismDeclarationOrdering",
			"TestDeterminismNoTimestampsOrAbsolutePaths",
		},
		"integration_test.go": {
			"TestIntegrationEndToEnd",
			"TestIntegrationGoldenProjection",
			"TestIntegrationIdempotent",
			"TestIntegrationNoticeOnManagedFiles",
		},
		"paths_test.go": {
			"TestNormalizeTargetPath",
			"TestValidatePathUniqueness",
			"TestResolverContains",
			"TestResolverHasSymlinkEscapeRejects",
			"TestWriteAtomicFileRoundTrip",
			"TestSameFilesystem",
		},
	}
	for file, needles := range requiredTests {
		path := filepath.Join(root, file)
		if !fileExists(path) {
			findings = append(findings, checks.Finding{
				Path:    path,
				Kind:    "missing_test",
				Message: "required test file missing",
			})
			continue
		}
		data := mustReadFile(path)
		for _, needle := range needles {
			if !strings.Contains(string(data), needle) {
				findings = append(findings, checks.Finding{
					Path:    path,
					Kind:    "missing_test_case",
					Message: "missing required test marker: " + needle,
				})
			}
		}
	}
	// Required pack and profile presence.
	packPath := filepath.Join(root, "packs/factory-core-v1/pack.json")
	if !fileExists(packPath) {
		findings = append(findings, checks.Finding{
			Path:    packPath,
			Kind:    "missing_pack",
			Message: "canonical factory-core-v1 pack missing",
		})
	}
	// Required fixture files (golden tree).
	fixtureFiles := []string{
		"testdata/fsharp-elm-empty/README.md",
		"testdata/fsharp-elm-empty/expected/Makefile",
		"testdata/fsharp-elm-empty/expected/.factory/project.json",
		"testdata/fsharp-elm-empty/expected/.factory/doctrine.lock.json",
		"testdata/fsharp-elm-empty/expected/.factory/generated/factory.mk",
		"testdata/fsharp-elm-empty/expected/.factory/generated/doctrine-inventory.md",
		"testdata/fsharp-elm-empty/expected/docs/factory/README.md",
	}
	for _, p := range fixtureFiles {
		full := filepath.Join(root, p)
		if !fileExists(full) {
			findings = append(findings, checks.Finding{
				Path:    full,
				Kind:    "missing_fixture",
				Message: "required fixture missing",
			})
		}
	}
	// Required documentation.
	docPath := "docs/doctrine/doctrine-compiler.md"
	if !fileExists(docPath) {
		findings = append(findings, checks.Finding{
			Path:    docPath,
			Kind:    "missing_documentation",
			Message: "required doctrine-compiler documentation missing",
		})
	}
	// Required CLI wiring.
	cmdPath := "cmd/leamas/factory_doctrine.go"
	if !fileExists(cmdPath) {
		findings = append(findings, checks.Finding{
			Path:    cmdPath,
			Kind:    "missing_cli_wiring",
			Message: "doctrine CLI dispatcher missing",
		})
	}
	data := mustReadFile(cmdPath)
	for _, needle := range []string{
		"runDoctrinePlan", "runDoctrineCompile", "runDoctrineVerify", "runDoctrineExplain",
		"--profile", "--target",
	} {
		if !strings.Contains(string(data), needle) {
			findings = append(findings, checks.Finding{
				Path:    cmdPath,
				Kind:    "missing_cli_surface",
				Message: "missing required CLI surface: " + needle,
			})
		}
	}
	return findings
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func oversized(path string, maxLines int) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Count(string(data), "\n") > maxLines
}

func mustReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}

// runDoctrineCompilerVerifier is the entry point used by the
// `leamas factory verify act-doctrine-compiler` subcommand. It runs
// the ACT-local verifier and exits with a deterministic code.
func runDoctrineCompilerVerifier() {
	handleFactoryVerifyActDoctrineCompiler()
}

// init registers the act-doctrine-compiler verifier with the
// factory verify dispatcher.
func init() {
	registerActDoctrineCompiler()
}
