package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/s1onique/leamas/internal/version"
)

type runnerIdentityProvider interface {
	Identity() (RunnerIdentity, error)
}

type currentRunnerIdentity struct{}

func (currentRunnerIdentity) Identity() (RunnerIdentity, error) {
	executable, err := os.Executable()
	if err != nil {
		return RunnerIdentity{}, fmt.Errorf("locate runner executable: %w", err)
	}
	binaryHash, err := hashRunnerBinary(executable)
	if err != nil {
		return RunnerIdentity{}, err
	}
	info := version.Get()
	revision, modified := vcsBuildIdentity()
	if revision == "" && oidPattern.MatchString(info.Commit) {
		revision = info.Commit
	}
	if err := validateOID("runner vcs revision", revision); err != nil {
		return RunnerIdentity{}, fmt.Errorf("runner has no full VCS revision: %w", err)
	}
	return RunnerIdentity{
		LeamasVersion: info.Version,
		BinarySHA256:  binaryHash,
		VCSRevision:   revision,
		VCSModified:   modified,
	}, nil
}

func vcsBuildIdentity() (string, bool) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	var revision string
	modified := false
	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	return revision, modified
}

func hashRunnerBinary(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open runner binary: %w", err)
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() {
		return "", fmt.Errorf("runner binary is not a regular file")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash runner binary: %w", err)
	}
	after, err := file.Stat()
	if err != nil || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || !os.SameFile(before, after) {
		return "", fmt.Errorf("runner binary changed during hashing")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
