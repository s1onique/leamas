package closure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

type commandExecutor interface {
	Execute(context.Context, *execution.Request) *execution.Result
}

type checkExecutionRequest struct {
	RepositoryRoot    string
	EvidenceDirectory string
	SubjectTreeOID    string
	Checks            []PlanCheck
	Now               func() time.Time
}

type boundedCommandExecutor struct{}

func (boundedCommandExecutor) Execute(ctx context.Context, request *execution.Request) *execution.Result {
	budget := execution.DefaultBudget().WithTimeout(request.Timeout).WithMaxConcurrent(1).WithMaxStarts(1)
	executor, err := execution.NewExecutor(budget, nil)
	if err != nil {
		return execution.NewErrorResult(execution.ErrInvalidRequest(fmt.Sprintf("create bounded executor: %v", err)))
	}
	defer executor.Close()
	return executor.Execute(ctx, request)
}

func executeChecks(ctx context.Context, request checkExecutionRequest, executor commandExecutor) ([]CheckResult, []EvidenceRecord, error) {
	if request.Now == nil {
		request.Now = time.Now
	}
	results := make([]CheckResult, 0, len(request.Checks))
	evidence := make([]EvidenceRecord, 0, len(request.Checks)*2)
	priorFailure := false
	for _, check := range request.Checks {
		if check.Mode != CheckModeRun {
			continue
		}
		if priorFailure {
			results = append(results, notRunResult(check, request.SubjectTreeOID))
			continue
		}
		result, records, err := executeOneCheck(ctx, request, check, executor)
		if err != nil {
			return nil, nil, err
		}
		results = append(results, result)
		evidence = append(evidence, records...)
		if result.Status == CheckStatusFail {
			priorFailure = true
		}
	}
	return results, evidence, nil
}

func executeOneCheck(ctx context.Context, request checkExecutionRequest, check PlanCheck, executor commandExecutor) (CheckResult, []EvidenceRecord, error) {
	started := request.Now().UTC()
	executionResult := executor.Execute(ctx, &execution.Request{
		Name:      "closure check " + check.ID,
		Args:      append([]string(nil), check.Argv...),
		Dir:       filepath.Join(request.RepositoryRoot, filepath.FromSlash(check.WorkingDirectory)),
		Env:       commandEnvironment(check.Environment),
		Timeout:   time.Duration(check.TimeoutSeconds) * time.Second,
		OutputCap: execution.DefaultMaxOutputBytes,
	})
	finished := request.Now().UTC()

	stdoutRecord, err := writeDetachedOutput(request.EvidenceDirectory, check.ID+".stdout", executionResult.Stdout)
	if err != nil {
		return CheckResult{}, nil, err
	}
	stderrRecord, err := writeDetachedOutput(request.EvidenceDirectory, check.ID+".stderr", executionResult.Stderr)
	if err != nil {
		return CheckResult{}, nil, err
	}
	status := CheckStatusPass
	if executionResult.Failed() || executionResult.OutputTruncated || executionResult.OutputIncomplete {
		status = CheckStatusFail
	}
	cleanupStatus := CleanupPass
	errorCode := ""
	if executionResult.Error != nil {
		errorCode = executionResult.Error.Code
		if errorCode == execution.CodeExecutionProcessTreeCleanupFailed {
			cleanupStatus = CleanupFailed
		}
	}
	exitCode := executionResult.ExitCode
	return CheckResult{
		CheckID:               check.ID,
		SubjectTreeOID:        request.SubjectTreeOID,
		Argv:                  append([]string(nil), check.Argv...),
		WorkingDirectory:      check.WorkingDirectory,
		OverriddenEnvironment: sortedEnvironmentNames(check.Environment),
		StartedAtUTC:          started.Format(time.RFC3339Nano),
		FinishedAtUTC:         finished.Format(time.RFC3339Nano),
		DurationMS:            executionResult.Duration.Milliseconds(),
		ExitCode:              &exitCode,
		Status:                status,
		StdoutSHA256:          stdoutRecord.SHA256,
		StdoutByteCount:       stdoutRecord.ByteCount,
		StderrSHA256:          stderrRecord.SHA256,
		StderrByteCount:       stderrRecord.ByteCount,
		OutputTruncated:       executionResult.OutputTruncated,
		OutputIncomplete:      executionResult.OutputIncomplete,
		OutputBytesObserved:   executionResult.OutputBytesObserved,
		CleanupStatus:         cleanupStatus,
		ExecutionErrorCode:    errorCode,
	}, []EvidenceRecord{stdoutRecord, stderrRecord}, nil
}

func notRunResult(check PlanCheck, subjectTree string) CheckResult {
	return CheckResult{
		CheckID:               check.ID,
		SubjectTreeOID:        subjectTree,
		Argv:                  append([]string(nil), check.Argv...),
		WorkingDirectory:      check.WorkingDirectory,
		OverriddenEnvironment: sortedEnvironmentNames(check.Environment),
		Status:                CheckStatusNotRun,
		CleanupStatus:         CleanupNotRequired,
	}
}

func commandEnvironment(overrides map[string]string) []string {
	names := sortedEnvironmentNames(overrides)
	environment := make([]string, 0, len(names)+3)
	for _, name := range names {
		environment = append(environment, name+"="+overrides[name])
	}
	environment = append(environment,
		execution.EnvRootID+"=",
		execution.EnvParentPID+"=",
		execution.EnvGeneration+"=",
	)
	return environment
}

func sortedEnvironmentNames(environment map[string]string) []string {
	names := make([]string, 0, len(environment))
	for name := range environment {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeDetachedOutput(directory, logicalName string, data []byte) (EvidenceRecord, error) {
	return writeDetachedBytes(directory, logicalName, "text/plain; charset=utf-8", data)
}

func writeDetachedBytes(directory, logicalName, mediaType string, data []byte) (EvidenceRecord, error) {
	path := filepath.Join(directory, logicalName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return EvidenceRecord{}, fmt.Errorf("create detached evidence %s: %w", logicalName, err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return EvidenceRecord{}, fmt.Errorf("write detached evidence %s: %w", logicalName, err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return EvidenceRecord{}, fmt.Errorf("sync detached evidence %s: %w", logicalName, err)
	}
	if err := file.Close(); err != nil {
		return EvidenceRecord{}, fmt.Errorf("close detached evidence %s: %w", logicalName, err)
	}
	return EvidenceRecord{
		LogicalName:  logicalName,
		MediaType:    mediaType,
		SHA256:       SHA256Hex(data),
		ByteCount:    int64(len(data)),
		Availability: "detached",
	}, nil
}
