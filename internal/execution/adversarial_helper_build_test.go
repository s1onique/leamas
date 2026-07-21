//go:build unix || darwin || linux

package execution

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

const helperPackagePath = "internal/execution/testdata/testhelper"

type helperSourceSnapshot struct {
	Files  []string
	Digest string
}

type helperBuildIdentity struct {
	Path         string
	SourceDigest string
	GoVersion    string
}

type helperPackageMetadata struct {
	Dir      string
	GoFiles  []string
	CgoFiles []string
}

func loadHelperSourceSnapshot(sourceDir string) (helperSourceSnapshot, error) {
	metadata, err := listHelperPackage(sourceDir)
	if err != nil {
		return helperSourceSnapshot{}, err
	}
	files := append([]string{}, metadata.GoFiles...)
	files = append(files, metadata.CgoFiles...)
	sort.Strings(files)
	files = compactStrings(files)
	if len(files) == 0 {
		return helperSourceSnapshot{}, fmt.Errorf("helper package has no buildable Go sources")
	}

	hash := sha256.New()
	for _, name := range files {
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			return helperSourceSnapshot{}, fmt.Errorf("unsafe helper source path %q", name)
		}
		path := filepath.Join(metadata.Dir, filepath.FromSlash(name))
		data, err := os.ReadFile(path)
		if err != nil {
			return helperSourceSnapshot{}, fmt.Errorf("read helper source %s: %w", name, err)
		}
		fmt.Fprintf(hash, "%d:%s:%d:", len(name), name, len(data))
		_, _ = hash.Write(data)
	}
	return helperSourceSnapshot{
		Files:  files,
		Digest: hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func listHelperPackage(sourceDir string) (helperPackageMetadata, error) {
	cmd := exec.Command("go", "list", "-json", ".")
	cmd.Dir = sourceDir
	cmd.Env = helperBuildEnvironment()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return helperPackageMetadata{}, fmt.Errorf("go list helper package: %w: %s",
			err, bytes.TrimSpace(output))
	}
	var metadata helperPackageMetadata
	if err := json.Unmarshal(output, &metadata); err != nil {
		return helperPackageMetadata{}, fmt.Errorf("decode go list output: %w", err)
	}
	return metadata, nil
}

func buildContentAddressedHelper(sourceDir, outputDir string) (helperBuildIdentity, bool, error) {
	snapshot, err := loadHelperSourceSnapshot(sourceDir)
	if err != nil {
		return helperBuildIdentity{}, false, err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return helperBuildIdentity{}, false, fmt.Errorf("create helper output directory: %w", err)
	}
	outputPath := filepath.Join(outputDir, "testhelper-"+snapshot.Digest)
	want := helperBuildIdentity{Path: outputPath, SourceDigest: snapshot.Digest}
	if got, err := readHelperIdentity(outputPath); err == nil &&
		got.SourceDigest == snapshot.Digest {
		want.GoVersion = got.GoVersion
		return want, false, nil
	}

	temporary, err := os.CreateTemp(outputDir, ".testhelper-building-")
	if err != nil {
		return helperBuildIdentity{}, false, fmt.Errorf("create temporary helper output: %w", err)
	}
	temporaryPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return helperBuildIdentity{}, false, fmt.Errorf("close temporary helper output: %w", err)
	}
	defer os.Remove(temporaryPath)

	linkValue := "-X=main.helperSourceDigest=" + snapshot.Digest
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags", linkValue,
		"-o", temporaryPath, ".")
	cmd.Dir = sourceDir
	cmd.Env = helperBuildEnvironment()
	if output, err := cmd.CombinedOutput(); err != nil {
		return helperBuildIdentity{}, false, fmt.Errorf("build test helper: %w: %s",
			err, bytes.TrimSpace(output))
	}
	got, err := readHelperIdentity(temporaryPath)
	if err != nil {
		return helperBuildIdentity{}, false, fmt.Errorf("verify built helper identity: %w", err)
	}
	if got.SourceDigest != snapshot.Digest {
		return helperBuildIdentity{}, false, fmt.Errorf(
			"built helper source digest=%q, pre-build digest=%q",
			got.SourceDigest, snapshot.Digest)
	}
	if err := os.Rename(temporaryPath, outputPath); err != nil {
		return helperBuildIdentity{}, false, fmt.Errorf("publish test helper: %w", err)
	}
	want.GoVersion = got.GoVersion
	return want, true, nil
}

func readHelperIdentity(path string) (helperBuildIdentity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "identity")
	output, err := cmd.Output()
	if err != nil {
		return helperBuildIdentity{}, fmt.Errorf("run %s identity: %w", path, err)
	}
	values := make(map[string]string, 2)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" || value == "" {
			return helperBuildIdentity{}, fmt.Errorf("malformed identity line %q", line)
		}
		if _, duplicate := values[key]; duplicate {
			return helperBuildIdentity{}, fmt.Errorf("duplicate identity key %q", key)
		}
		values[key] = value
	}
	if len(values) != 2 || values["helper_source_digest"] == "" ||
		values["helper_build_go_version"] == "" {
		return helperBuildIdentity{}, fmt.Errorf("incomplete helper identity %q", output)
	}
	return helperBuildIdentity{
		Path:         path,
		SourceDigest: values["helper_source_digest"],
		GoVersion:    values["helper_build_go_version"],
	}, nil
}

func helperBuildEnvironment() []string {
	env := os.Environ()
	env = updateEnv(env, "CGO_ENABLED", "0")
	env = updateEnv(env, "GO111MODULE", "off")
	return env
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

func TestMain(m *testing.M) {
	code := m.Run()
	if helperOutputDir != "" {
		_ = os.RemoveAll(helperOutputDir)
	}
	os.Exit(code)
}
