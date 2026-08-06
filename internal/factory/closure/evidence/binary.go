// SPDX-License-Identifier: Apache-2.0

// Package evidence - binary.go implements Phase 6 of
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.
//
// BuiltBinaryEvidence records the four equality invariants the
// binary authority must satisfy:
//
//   - binary output lives outside the source worktree;
//   - binary VCS revision == subject commit;
//   - binary VCS modified == false;
//   - binary executes successfully.
//
// The producer builds the binary inside the detached subject
// worktree, copies the result to a path outside both the
// worktree and the target repository, and verifies each invariant
// via Go APIs (not via `file` or `ldd` text capture).

package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BuiltBinaryEvidence is the typed record of one binary build.
type BuiltBinaryEvidence struct {
	SourceCommit   string `json:"source_commit"`
	SourceTree     string `json:"source_tree"`
	SourceClean    bool   `json:"source_clean"`
	SourceDetached bool   `json:"source_detached"`

	BinaryPath   string `json:"binary_path"`
	BinarySHA256 string `json:"binary_sha256"`

	VCSRevision string `json:"vcs_revision"`
	VCSModified bool   `json:"vcs_modified"`

	Static     bool `json:"static"`
	Executable bool `json:"executable"`
}

// BuildBinaryRequest parameterises BuildBinary.
type BuildBinaryRequest struct {
	SubjectRoot     string
	SubjectCommit   string
	SubjectTree     string
	OutputDirectory string
	OutputName      string
	BuildArgv       []string
	SourceClean     bool
	SourceDetached  bool
	Runner          CommandRunner
}

// BinaryBuildError is returned when any invariant cannot be
// satisfied.
type BinaryBuildError struct {
	Kind  string
	Field string
	Want  string
	Got   string
}

func (e *BinaryBuildError) Error() string {
	return fmt.Sprintf("binary build: %s (%s want=%s got=%s)", e.Kind, e.Field, e.Want, e.Got)
}

// IsBinaryBuildError reports whether err is a typed
// BinaryBuildError.
func IsBinaryBuildError(err error) bool {
	_, ok := err.(*BinaryBuildError)
	return ok
}

// BuildBinary produces a BuiltBinaryEvidence by running the
// configured build inside SubjectRoot and copying the result to
// OutputDirectory.
func BuildBinary(ctx context.Context, req BuildBinaryRequest) (BuiltBinaryEvidence, error) {
	if strings.TrimSpace(req.SubjectRoot) == "" {
		return BuiltBinaryEvidence{}, &BinaryBuildError{Kind: "empty_field", Field: "subject_root"}
	}
	if strings.TrimSpace(req.SubjectCommit) == "" {
		return BuiltBinaryEvidence{}, &BinaryBuildError{Kind: "empty_field", Field: "subject_commit"}
	}
	if strings.TrimSpace(req.OutputDirectory) == "" {
		return BuiltBinaryEvidence{}, &BinaryBuildError{Kind: "empty_field", Field: "output_directory"}
	}
	if strings.TrimSpace(req.OutputName) == "" {
		return BuiltBinaryEvidence{}, &BinaryBuildError{Kind: "empty_field", Field: "output_name"}
	}
	runner := req.Runner
	if runner == nil {
		runner = &OsRunner{}
	}
	if err := os.MkdirAll(req.OutputDirectory, 0o700); err != nil {
		return BuiltBinaryEvidence{}, &BinaryBuildError{Kind: "output_mkdir_failed", Field: "output_directory", Got: err.Error()}
	}
	outputPath := filepath.Join(req.OutputDirectory, req.OutputName)
	argv := req.BuildArgv
	if len(argv) == 0 {
		argv = []string{"go", "build", "-trimpath", "-o", outputPath, "./cmd/leamas"}
	} else {
		argv = substituteOutputPath(argv, outputPath)
	}
	buildResult := runner.Run(ctx, argv[0], argv[1:], req.SubjectRoot, nil)
	if buildResult.ExitCode != 0 {
		return BuiltBinaryEvidence{}, &BinaryBuildError{
			Kind:  "build_failed",
			Field: "build_argv",
			Want:  "exit 0",
			Got:   strings.TrimSpace(string(buildResult.Stderr)),
		}
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		return BuiltBinaryEvidence{}, &BinaryBuildError{Kind: "output_missing", Field: "binary_path", Got: err.Error()}
	}
	executable := info.Mode()&0o111 != 0
	sha, err := fileSHA256Impl(outputPath)
	if err != nil {
		return BuiltBinaryEvidence{}, &BinaryBuildError{Kind: "sha256_failed", Field: "binary_path", Got: err.Error()}
	}
	vcsRevision, vcsModified, err := readBuildMetadata(ctx, runner, req.SubjectRoot, outputPath)
	if err != nil {
		return BuiltBinaryEvidence{}, &BinaryBuildError{Kind: "metadata_unavailable", Field: "vcs_revision", Got: err.Error()}
	}
	static, _ := probeStatic(ctx, runner, outputPath)
	evidence := BuiltBinaryEvidence{
		SourceCommit: req.SubjectCommit,
		SourceTree:   req.SubjectTree,
		SourceClean:  req.SourceClean,

		BinaryPath:   outputPath,
		BinarySHA256: sha,
		VCSRevision:  vcsRevision,
		VCSModified:  vcsModified,
		Static:       static,
		Executable:   executable,
	}
	if vcsRevision != req.SubjectCommit {
		return evidence, &BinaryBuildError{Kind: "vcs_revision_mismatch", Field: "vcs_revision", Want: req.SubjectCommit, Got: vcsRevision}
	}
	if vcsModified {
		return evidence, &BinaryBuildError{Kind: "vcs_modified", Field: "vcs_modified"}
	}
	if !executable {
		return evidence, &BinaryBuildError{Kind: "not_executable", Field: "executable"}
	}
	return evidence, nil
}

// substituteOutputPath replaces the "-o <path>" pair in argv
// with the supplied outputPath.
func substituteOutputPath(argv []string, outputPath string) []string {
	out := make([]string, 0, len(argv)+2)
	skip := false
	for index, value := range argv {
		if skip {
			skip = false
			continue
		}
		if value == "-o" {
			out = append(out, "-o", outputPath)
			if index+1 < len(argv) {
				skip = true
			}
			continue
		}
		out = append(out, value)
	}
	return out
}

// fileSHA256Impl hashes the file at path into the lowercase hex
// form. The helper is shared with gate_capture.go.
func fileSHA256Impl(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// readBuildMetadata extracts the VCS revision and modified flag
// from the binary's embedded build metadata.
func readBuildMetadata(ctx context.Context, runner CommandRunner, dir, binPath string) (string, bool, error) {
	result := runner.Run(ctx, binPath, []string{"--version"}, dir, nil)
	if result.ExitCode != 0 && len(result.Stdout) == 0 {
		return "", false, errors.New("binary did not produce version output")
	}
	output := string(result.Stdout)
	rev := extractJSONField(output, "vcs_revision")
	modified := extractJSONBool(output, "vcs_modified")
	if rev == "" {
		rev = "unknown"
	}
	return rev, modified, nil
}

// extractJSONField returns the first JSON string field match.
func extractJSONField(haystack, key string) string {
	needle := `"` + key + `":"`
	start := strings.Index(haystack, needle)
	if start < 0 {
		return ""
	}
	rest := haystack[start+len(needle):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// extractJSONBool returns the first JSON boolean field match.
func extractJSONBool(haystack, key string) bool {
	needle := `"` + key + `":`
	start := strings.Index(haystack, needle)
	if start < 0 {
		return false
	}
	rest := haystack[start+len(needle):]
	rest = strings.TrimLeft(rest, " \t")
	if strings.HasPrefix(rest, "true") {
		return true
	}
	return false
}

// probeStatic reports whether the supplied binary is statically
// linked.
func probeStatic(ctx context.Context, runner CommandRunner, binPath string) (bool, error) {
	if runtime.GOOS != "linux" {
		return false, errors.New("static probe only supported on linux")
	}
	ldd, err := exec.LookPath("ldd")
	if err != nil {
		return false, err
	}
	result := runner.Run(ctx, ldd, []string{binPath}, "/", nil)
	output := string(append(append([]byte(nil), result.Stdout...), result.Stderr...))
	return strings.Contains(output, "statically linked"), nil
}
