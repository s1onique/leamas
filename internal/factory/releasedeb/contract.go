package releasedeb

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/version"
)

const (
	ExpectedArchitecture  = "amd64"
	ExpectedGOOS          = "linux"
	ExpectedLicense       = "Apache-2.0"
	ExpectedNFPMVersion   = "v2.47.0"
	OfficialLicenseSHA256 = "cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30"
)

// Config is the observable input contract for Debian release checks.
type Config struct {
	PackagePath   string
	ReleaseBinary string
	Version       string
	Architecture  string
	Commit        string
	LicenseFile   string
	License       string
}

func (c Config) expectedPackageVersion() string {
	return c.Version + "-1"
}

// ValidateBuildInputs rejects anything outside the first-release contract
// before release-build or nFPM can run.
func ValidateBuildInputs(versionValue, goos, goarch, licenseFile, license string) error {
	if versionValue == "" || versionValue == "dev" || versionValue == "unknown" ||
		!version.IsValidSemVer(versionValue) {
		return fmt.Errorf("VERSION must be a strict stable SemVer (got %q)", versionValue)
	}
	parts, ok := version.ParseSemVer(versionValue)
	if !ok || len(parts.Pre) != 0 || parts.Build != "" {
		return fmt.Errorf("VERSION must be a strict stable SemVer (got %q)", versionValue)
	}
	if goos != ExpectedGOOS {
		return fmt.Errorf("GOOS must be linux (got %q)", goos)
	}
	if goarch != ExpectedArchitecture {
		return fmt.Errorf("GOARCH must be amd64 (got %q)", goarch)
	}
	if license != ExpectedLicense {
		return fmt.Errorf("license metadata must be %s (got %q)", ExpectedLicense, license)
	}
	if licenseFile == "" {
		return errors.New("license file does not exist: empty path")
	}
	info, err := os.Stat(licenseFile)
	if err != nil {
		return fmt.Errorf("license file does not exist: %s: %w", licenseFile, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("license file is not a regular file: %s", licenseFile)
	}
	if err := verifyOfficialLicense(licenseFile); err != nil {
		return err
	}
	return nil
}

func verifyOfficialLicense(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read license file %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != OfficialLicenseSHA256 {
		return fmt.Errorf("license file %s is not the committed Apache-2.0 text (sha256 %s)", path, actual)
	}
	return nil
}

func requireRegularExecutable(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist: %s", label, path)
		}
		return fmt.Errorf("stat %s %s: %w", label, path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file: %s", label, path)
	}
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("%s is not executable: %s", label, path)
	}
	return nil
}

func packageInputPath(path string) string {
	if filepath.IsAbs(path) || strings.HasPrefix(path, "./") {
		return path
	}
	return "./" + path
}

func checksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
