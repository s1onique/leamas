// SPDX-License-Identifier: Apache-2.0

package closure

// closure_exact_subject_binary_identity.go implements the
// production binary-identity reader for BuildExactSubjectBinary.
//
// CANONICAL AUTHORITY:
//   exactBinaryReadIdentity invokes the produced binary's
//   own `version --json` surface and decodes the canonical
//   identity fields. This is the SAME surface the existing
//   production release artefacts expose; the B1 authority
//   MUST NOT introduce a second version scheme.
//
// AUXILIARY DIAGNOSTICS:
//   exactBinaryReadNativeBuildInfo decodes
//   `go version -m -json <binary>`. Native values are
//   recorded as diagnostics only; their absence MUST NOT
//   fail the exact-S authority because cmd/go does not
//   stamp linked worktrees.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

// leamasVersionInfo is the typed wire form of `leamas version
// --json`. The fields mirror internal/version.Info but only
// the keys B1 needs are decoded.
type leamasVersionInfo struct {
	Version         string `json:"version"`
	DeclaredVersion string `json:"declared_version,omitempty"`
	Commit          string `json:"commit"`
	BuildTime       string `json:"build_time"`
	Dirty           string `json:"dirty,omitempty"`
}

// exactBinaryIdentity holds the parsed canonical identity of
// the produced binary.
type exactBinaryIdentity struct {
	Commit    string
	Modified  bool
	FromDirty bool // true when the dirty key was present
	FromJSON  bool // true when the JSON form decoded cleanly
}

// exactBinaryReadIdentity invokes `leamas version --json` via
// the bounded execution gateway, parses the canonical JSON
// wire form, and returns the typed identity.
//
// Required predicates:
//   - exit code == 0
//   - bounded stdout (no truncation)
//   - canonical JSON decoded
//   - Commit field non-empty
//
// The dirty marker maps as follows:
//   dirty key absent       -> Modified=false (clean build)
//   dirty:"true"           -> Modified=true (rejected)
//   dirty:"false"          -> Modified=false
//   anything else          -> Modified=true (rejected)
//
// The subprocess receives exactBinarySubjectEnv so any
// LEAMAS_EXEC_* re-entry markers inherited from the parent
// process are stripped: the produced binary's `main()`
// re-entry fuse would otherwise abort with a nested-execution
// error before the version surface could respond.
func exactBinaryReadIdentity(ctx context.Context, binaryPath string) (exactBinaryIdentity, *execution.Result, error) {
	ex, err := execution.NewExecutor(exactBinaryIdentityBudget(), nil)
	if err != nil {
		return exactBinaryIdentity{}, nil, fmt.Errorf("create identity executor: %w", err)
	}
	defer ex.Close()
	result := ex.Execute(ctx, &execution.Request{
		Name:      "exact-binary identity introspection",
		Args:      []string{binaryPath, "version", "--json"},
		Env:       exactBinarySubjectEnv(),
		Timeout:   30 * time.Second,
		OutputCap: 64 * 1024,
	})
	if result.Error != nil {
		return exactBinaryIdentity{}, result, fmt.Errorf("identity subprocess error: %w", result.Error)
	}
	if result.ExitCode != 0 {
		return exactBinaryIdentity{}, result, fmt.Errorf("identity exit=%d (stderr=%s)",
			result.ExitCode, strings.TrimSpace(string(result.Stderr)))
	}
	if result.OutputTruncated || result.OutputIncomplete {
		return exactBinaryIdentity{}, result, errors.New("identity output truncated or incomplete")
	}
	var info leamasVersionInfo
	if err := json.Unmarshal(result.Stdout, &info); err != nil {
		return exactBinaryIdentity{}, result, fmt.Errorf("decode identity JSON: %w", err)
	}
	if info.Commit == "" {
		return exactBinaryIdentity{}, result, errors.New("identity JSON missing commit")
	}
	// The production version CLI only prints the dirty key
	// when it differs from the default "false" (see
	// cmd/leamas/version.go). Treat absent as Modified=false.
	modified := false
	fromDirty := false
	if info.Dirty != "" {
		fromDirty = true
		switch info.Dirty {
		case "true":
			modified = true
		case "false":
			modified = false
		default:
			return exactBinaryIdentity{}, result, fmt.Errorf("identity JSON has unknown dirty value %q", info.Dirty)
		}
	}
	return exactBinaryIdentity{
		Commit:    info.Commit,
		Modified:  modified,
		FromDirty: fromDirty,
		FromJSON:  true,
	}, result, nil
}

// exactBinaryNativeBuildInfo captures the auxiliary
// `go version -m -json <binary>` parse. The values are
// auxiliary diagnostics only; their absence MUST NOT fail
// the exact-S authority.
type exactBinaryNativeBuildInfo struct {
	Revision         string
	RevisionPresent  bool
	Modified         bool
	ModifiedPresent  bool
	RawOutput        []byte
	SubprocessResult *execution.Result
}

// exactBinaryReadNativeBuildInfo invokes
// `go version -m -json <binary>` via the bounded execution
// gateway and decodes the runtime/debug.BuildInfo JSON. The
// helper rejects duplicate / malformed vcs settings because
// the cmd/go output contract guarantees a single occurrence
// of each key with a recognised "true" / "false" payload.
//
// When cmd/go is unavailable (e.g. toolchain absent in the
// sandbox), the helper records the failure as a typed
// observation without producing native values; the caller
// treats absence as auxiliary-only.
func exactBinaryReadNativeBuildInfo(ctx context.Context, binaryPath string) (exactBinaryNativeBuildInfo, error) {
	ex, err := execution.NewExecutor(exactBinaryNativeBuildInfoBudget(), nil)
	if err != nil {
		return exactBinaryNativeBuildInfo{}, fmt.Errorf("create native buildinfo executor: %w", err)
	}
	defer ex.Close()
	result := ex.Execute(ctx, &execution.Request{
		Name:      "exact-binary native buildinfo",
		Args:      []string{"go", "version", "-m", "-json", binaryPath},
		Env:       exactBinarySubjectEnv(),
		Timeout:   30 * time.Second,
		OutputCap: 64 * 1024,
	})
	out := exactBinaryNativeBuildInfo{SubprocessResult: result}
	if result.Error != nil {
		// Toolchain absence is not a hard failure: cmd/go is
		// an auxiliary diagnostic surface, not the canonical
		// authority. Record the raw output and report the
		// observation as auxiliary-only.
		out.RawOutput = append(out.RawOutput, result.Stderr...)
		return out, fmt.Errorf("native buildinfo subprocess error: %w", result.Error)
	}
	if result.ExitCode != 0 {
		// Same: cmd/go refusing to stamp is not a hard
		// failure for the exact-S authority.
		out.RawOutput = append(out.RawOutput, result.Stderr...)
		return out, fmt.Errorf("native buildinfo exit=%d (stderr=%s)",
			result.ExitCode, strings.TrimSpace(string(result.Stderr)))
	}
	if result.OutputTruncated || result.OutputIncomplete {
		return out, errors.New("native buildinfo output truncated or incomplete")
	}
	out.RawOutput = append(out.RawOutput, result.Stdout...)
	var bi debug.BuildInfo
	if err := json.Unmarshal(result.Stdout, &bi); err != nil {
		return out, fmt.Errorf("decode native buildinfo JSON: %w", err)
	}
	// Parse vcs.revision / vcs.modified with strict
	// duplicate + value rejection.
	var (
		revisionSeen  bool
		modifiedSeen  bool
		revisionValue string
		modifiedValue bool
	)
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if revisionSeen {
				return out, errors.New("native buildinfo: duplicate vcs.revision")
			}
			revisionSeen = true
			revisionValue = s.Value
		case "vcs.modified":
			if modifiedSeen {
				return out, errors.New("native buildinfo: duplicate vcs.modified")
			}
			modifiedSeen = true
			switch s.Value {
			case "true":
				modifiedValue = true
			case "false":
				modifiedValue = false
			default:
				return out, fmt.Errorf("native buildinfo: invalid vcs.modified value %q", s.Value)
			}
		}
	}
	if revisionSeen {
		out.Revision = revisionValue
		out.RevisionPresent = true
	}
	if modifiedSeen {
		out.Modified = modifiedValue
		out.ModifiedPresent = true
	}
	return out, nil
}