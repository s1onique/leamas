// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var v2TagNamePattern = regexp.MustCompile(`^act/[a-z0-9][a-z0-9._/-]{0,199}$`)

type v2TagInput struct {
	ActID           string
	TagName         string
	ClosureCommit   string
	ClosureTree     string
	FreezeCommit    string
	FreezeTree      string
	SubjectCommit   string
	SubjectTree     string
	PlanBlobOID     string
	PlanSHA256      string
	ManifestBlobOID string
	ManifestSHA256  string
	ReportBlobOID   string
	ReportSHA256    string
	EvidenceSHA256  string
	RunnerRevision  string
	RunnerBinarySHA string
}

type v2TagObject struct {
	Name  string
	ActID string
	Bytes []byte
	OID   string
}

func canonicalV2TagName(actID string) string {
	return "act/" + strings.ToLower(actID)
}

func createV2TagObject(ctx context.Context, git gitClient, repoRoot string,
	format ObjectFormat, input v2TagInput) (v2TagObject, error) {
	if input.TagName == "" {
		input.TagName = canonicalV2TagName(input.ActID)
	}
	if !v2TagNamePattern.MatchString(input.TagName) || input.TagName != canonicalV2TagName(input.ActID) {
		return v2TagObject{}, fmt.Errorf("invalid canonical tag name %q", input.TagName)
	}
	if err := validateV2TagInputOIDs(input, format); err != nil {
		return v2TagObject{}, err
	}
	closureEpoch, err := commitEpoch(ctx, git, repoRoot, input.ClosureCommit)
	if err != nil {
		return v2TagObject{}, err
	}
	bytes := renderV2TagBytes(input, closureEpoch+1)
	result := git.RunWithStdin(ctx, repoRoot, string(bytes), "mktag")
	if result.Err != nil || result.ExitCode != 0 {
		return v2TagObject{}, fmt.Errorf("git mktag failed: %s", gitFailureDetail(result))
	}
	oid := strings.TrimSpace(string(result.Stdout))
	if err := ValidateOIDWithFormat("tag object", oid, format); err != nil {
		return v2TagObject{}, err
	}
	return v2TagObject{Name: input.TagName, ActID: input.ActID, Bytes: bytes, OID: oid}, nil
}

func validateV2TagInputOIDs(input v2TagInput, format ObjectFormat) error {
	fields := []struct{ name, value string }{
		{"closure commit", input.ClosureCommit}, {"closure tree", input.ClosureTree},
		{"freeze commit", input.FreezeCommit}, {"freeze tree", input.FreezeTree},
		{"subject commit", input.SubjectCommit}, {"subject tree", input.SubjectTree},
		{"plan blob", input.PlanBlobOID}, {"manifest blob", input.ManifestBlobOID},
		{"report blob", input.ReportBlobOID}, {"runner revision", input.RunnerRevision},
	}
	for _, field := range fields {
		if err := ValidateOIDWithFormat(field.name, field.value, format); err != nil {
			return err
		}
	}
	for name, digest := range map[string]string{
		"plan SHA-256": input.PlanSHA256, "manifest SHA-256": input.ManifestSHA256,
		"report SHA-256": input.ReportSHA256, "evidence SHA-256": input.EvidenceSHA256,
		"runner binary SHA-256": input.RunnerBinarySHA,
	} {
		if err := validateSHA256(name, digest); err != nil {
			return err
		}
	}
	return nil
}

func renderV2TagBytes(input v2TagInput, epoch int64) []byte {
	var tag bytes.Buffer
	fmt.Fprintf(&tag, "object %s\n", input.ClosureCommit)
	fmt.Fprintln(&tag, "type commit")
	fmt.Fprintf(&tag, "tag %s\n", input.TagName)
	fmt.Fprintf(&tag, "tagger %s <%s> %d +0000\n\n", v2ClosureName, v2ClosureEmail, epoch)
	fmt.Fprintf(&tag, "Closure Protocol v2: %s\n\n", input.ActID)
	fmt.Fprintf(&tag, "F %s\nF_TREE %s\n", input.FreezeCommit, input.FreezeTree)
	fmt.Fprintf(&tag, "S %s\nS_TREE %s\n", input.SubjectCommit, input.SubjectTree)
	fmt.Fprintf(&tag, "C %s\nC_TREE %s\n", input.ClosureCommit, input.ClosureTree)
	fmt.Fprintf(&tag, "PLAN_BLOB %s\nPLAN_SHA256 %s\n", input.PlanBlobOID, input.PlanSHA256)
	fmt.Fprintf(&tag, "MANIFEST_BLOB %s\nMANIFEST_SHA256 %s\n", input.ManifestBlobOID, input.ManifestSHA256)
	fmt.Fprintf(&tag, "REPORT_BLOB %s\nREPORT_SHA256 %s\n", input.ReportBlobOID, input.ReportSHA256)
	fmt.Fprintf(&tag, "EVIDENCE_INDEX_SHA256 %s\n", input.EvidenceSHA256)
	fmt.Fprintf(&tag, "RUNNER_VCS_REVISION %s\nRUNNER_BINARY_SHA256 %s\n",
		input.RunnerRevision, input.RunnerBinarySHA)
	return tag.Bytes()
}

func commitEpoch(ctx context.Context, git gitClient, repoRoot, commit string) (int64, error) {
	result := git.Run(ctx, repoRoot, "show", "-s", "--format=%ct", commit)
	if result.Err != nil || result.ExitCode != 0 {
		return 0, fmt.Errorf("read commit epoch: %s", gitFailureDetail(result))
	}
	epoch, err := strconv.ParseInt(strings.TrimSpace(string(result.Stdout)), 10, 64)
	if err != nil || epoch == int64(^uint64(0)>>1) {
		return 0, fmt.Errorf("parse commit epoch")
	}
	return epoch, nil
}
