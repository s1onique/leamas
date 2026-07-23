package closure

import (
	"strings"
	"testing"
)

const fullCommitOID = "1111111111111111111111111111111111111111"
const fullTreeOID = "2222222222222222222222222222222222222222"

func canonicalPlanJSON() string {
	return `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-TEST01",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "policy_profile": "leamas-act-v1",
  "freeze": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "blob_oid": "0000000000000000000000000000000000000000000000000000000000000000"
  },
  "checks": [
    {
      "id": "focused-count-1",
      "mode": "run",
      "argv": ["go", "test", "-count=1", "./internal/factory/closure/...", "./cmd/leamas/..."],
      "working_directory": ".",
      "timeout_seconds": 600,
      "environment": {"CGO_ENABLED": "0"}
    },
    {
      "id": "focused-count-20",
      "mode": "run",
      "argv": ["go", "test", "-count=20", "./internal/factory/closure/...", "./cmd/leamas/..."],
      "working_directory": ".",
      "timeout_seconds": 600,
      "environment": {"CGO_ENABLED": "0"}
    },
    {
      "id": "focused-race-5",
      "mode": "run",
      "argv": ["go", "test", "-race", "-count=5", "./internal/factory/closure/...", "./cmd/leamas/..."],
      "working_directory": ".",
      "timeout_seconds": 600,
      "environment": {"CGO_ENABLED": "0"}
    },
    {
      "id": "vet",
      "mode": "run",
      "argv": ["go", "vet", "./internal/factory/closure/...", "./cmd/leamas/..."],
      "working_directory": ".",
      "timeout_seconds": 300,
      "environment": {"CGO_ENABLED": "0"}
    },
    {
      "id": "build",
      "mode": "run",
      "argv": ["go", "build", "-buildvcs=true", "-trimpath", "-o", "/tmp/leamas-closure-protocol-v1-self", "./cmd/leamas"],
      "working_directory": ".",
      "timeout_seconds": 600,
      "environment": {"CGO_ENABLED": "0"}
    },
    {
      "id": "gate-fast",
      "mode": "run",
      "argv": ["make", "gate-fast"],
      "working_directory": ".",
      "timeout_seconds": 600,
      "environment": {"CGO_ENABLED": "0"}
    },
    {
      "id": "diff-check",
      "mode": "run",
      "argv": ["git", "diff", "--check"],
      "working_directory": ".",
      "timeout_seconds": 60,
      "environment": {}
    },
    {
      "id": "dupcode",
      "mode": "exclude",
      "reason": "No dupcode-owned source or registration changed."
    }
  ],
  "artifacts": [
    {
      "id": "summary",
      "path": ".factory/gate-fast-summary.json",
      "required": true,
      "max_bytes": 1048576,
      "media_type": "application/json"
    }
  ],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`
}

func TestClosurePlanAcceptsCanonicalV1(t *testing.T) {
	plan, err := DecodePlan([]byte(canonicalPlanJSON()))
	if err != nil {
		t.Fatalf("DecodePlan() error = %v", err)
	}
	if plan.ContractVersion != ContractVersionV1 || len(plan.Checks) != 8 {
		t.Fatalf("decoded plan = %+v", plan)
	}
}

func TestClosurePlanRejectsUnknownField(t *testing.T) {
	assertPlanDecodeError(t, strings.Replace(canonicalPlanJSON(),
		`"act_id": "ACT-LEAMAS-TEST01",`,
		`"act_id": "ACT-LEAMAS-TEST01", "surprise": true,`, 1), "unknown field")
}

func TestClosurePlanRejectsDuplicateJSONKey(t *testing.T) {
	assertPlanDecodeError(t, strings.Replace(canonicalPlanJSON(),
		`"contract_version": 1,`,
		`"contract_version": 1, "contract_version": 1,`, 1), "duplicate")
}

func TestClosurePlanRejectsTrailingJSON(t *testing.T) {
	assertPlanDecodeError(t, canonicalPlanJSON()+` {}`, "trailing")
}

func TestClosurePlanRejectsUnsupportedVersion(t *testing.T) {
	assertPlanDecodeError(t, strings.Replace(canonicalPlanJSON(),
		`"contract_version": 1`, `"contract_version": 2`, 1), "unsupported")
}

func TestClosurePlanRejectsDuplicateCheckID(t *testing.T) {
	assertPlanDecodeError(t, strings.Replace(canonicalPlanJSON(),
		`"id": "dupcode"`, `"id": "diff-check"`, 1), "duplicate check")
}

func TestClosurePlanRejectsDuplicateArtifactID(t *testing.T) {
	raw := strings.Replace(canonicalPlanJSON(), `  ],
  "policy"`, `,
    {
      "id": "summary",
      "path": ".factory/other.json",
      "required": false,
      "max_bytes": 10,
      "media_type": "application/json"
    }
  ],
  "policy"`, 1)
	assertPlanDecodeError(t, raw, "duplicate artifact")
}

func TestClosurePlanRejectsShellString(t *testing.T) {
	raw := strings.Replace(canonicalPlanJSON(),
		`"argv": ["go", "test", "-count=1", "./internal/factory/closure/...", "./cmd/leamas/..."],`,
		`"command": "go test ./internal/factory/closure/...",`, 1)
	assertPlanDecodeError(t, raw, "unknown field")
}

func TestClosurePlanRejectsEscapingWorkingDirectory(t *testing.T) {
	raw := strings.Replace(canonicalPlanJSON(),
		`"working_directory": "."`, `"working_directory": "../outside"`, 1)
	assertPlanDecodeError(t, raw, "working_directory")
}

func TestClosurePlanRejectsMissingExclusionReason(t *testing.T) {
	raw := strings.Replace(canonicalPlanJSON(),
		`"reason": "No dupcode-owned source or registration changed."`, `"reason": ""`, 1)
	assertPlanDecodeError(t, raw, "reason")
}

func TestClosurePlanRejectsPlaceholderIdentity(t *testing.T) {
	raw := strings.Replace(canonicalPlanJSON(), fullCommitOID, "<commit>", 1)
	assertPlanDecodeError(t, raw, "placeholder")
}

func assertPlanDecodeError(t *testing.T, raw, contains string) {
	t.Helper()
	_, err := DecodePlan([]byte(raw))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(contains)) {
		t.Fatalf("DecodePlan() error = %v, want containing %q", err, contains)
	}
}
