// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ValidateV2BinaryIdentity verifies that an identity names the resolved
// executable file whose bytes produced the declared digest. VCSRevision is
// deliberately SHA-1-only: it identifies the Leamas source repository, not
// the target repository's potentially different object format.
func ValidateV2BinaryIdentity(identity V2BinaryIdentity) error {
	if identity.Path == "" || strings.TrimSpace(identity.Path) != identity.Path {
		return invalidV2BinaryIdentity("path", "binary path is required without surrounding whitespace")
	}
	if !filepath.IsAbs(identity.Path) || filepath.Clean(identity.Path) != identity.Path {
		return invalidV2BinaryIdentity("path", "binary path must be absolute and clean")
	}
	resolved, err := filepath.EvalSymlinks(identity.Path)
	if err != nil {
		return invalidV2BinaryIdentity("path", fmt.Sprintf("resolve binary path: %v", err))
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return invalidV2BinaryIdentity("path", fmt.Sprintf("make binary path absolute: %v", err))
	}
	resolved = filepath.Clean(resolved)
	if resolved != identity.Path {
		return invalidV2BinaryIdentity("path", "binary path must already be symlink-resolved")
	}
	info, err := os.Stat(identity.Path)
	if err != nil {
		return invalidV2BinaryIdentity("path", fmt.Sprintf("stat binary: %v", err))
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return invalidV2BinaryIdentity("path", "binary path must name an executable regular file")
	}
	if !sha256Pattern.MatchString(identity.SHA256) {
		return invalidV2BinaryIdentity("sha256", "binary SHA-256 must be 64 lowercase hexadecimal characters")
	}
	actual, err := sha256File(identity.Path)
	if err != nil {
		return invalidV2BinaryIdentity("sha256", fmt.Sprintf("hash binary: %v", err))
	}
	if actual != identity.SHA256 {
		return invalidV2BinaryIdentity("sha256", fmt.Sprintf("binary SHA-256 %s does not match supplied %s", actual, identity.SHA256))
	}
	if !sha1Pattern.MatchString(identity.VCSRevision) {
		return invalidV2BinaryIdentity("vcs_revision", "binary VCS revision must be a full 40-character lowercase Git OID")
	}
	if identity.LeamasVersion == "" || strings.TrimSpace(identity.LeamasVersion) != identity.LeamasVersion || containsControl(identity.LeamasVersion) {
		return invalidV2BinaryIdentity("leamas_version", "Leamas version must be nonempty and contain no surrounding whitespace or controls")
	}
	return nil
}

func invalidV2BinaryIdentity(property, message string) error {
	return NewV2ErrorWith(V2CodeBinaryIdentityInvalid, message,
		"leamas_binary_identity."+property, "")
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func validateV2ManifestObjectIdentities(b V2ManifestBuild) error {
	fields := []struct {
		name  string
		value string
	}{
		{name: "subject_commit", value: b.SubjectCommit},
		{name: "subject_tree", value: b.SubjectTree},
		{name: "freeze_commit", value: b.FreezeCommit},
		{name: "freeze_tree", value: b.FreezeTree},
		{name: "execution_tree", value: b.ExecutionTree},
		{name: "plan_blob", value: b.PlanBlob},
		{name: "caller_head", value: b.CallerHead},
	}
	width := 0
	for _, field := range fields {
		if !oidPattern.MatchString(field.value) {
			return invalidV2ManifestIdentity(field.name,
				fmt.Sprintf("%s must be a full lowercase Git OID", field.name))
		}
		if width == 0 {
			width = len(field.value)
		}
		if len(field.value) != width {
			return invalidV2ManifestIdentity(field.name,
				"target-repository object identities must use one Git object format")
		}
	}
	if b.ExecutionTree != b.SubjectTree {
		return invalidV2ManifestIdentity("execution_tree",
			fmt.Sprintf("execution_tree %s must equal subject_tree %s", b.ExecutionTree, b.SubjectTree))
	}
	return nil
}

func invalidV2ManifestIdentity(property, message string) error {
	return NewV2ErrorWith(V2CodeManifestIdentityInvalid, message, property, "")
}
