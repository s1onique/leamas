package execution

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestRetainedOutputClassificationContract(t *testing.T) {
	const wantCode = "execution_retained_output_pipe"
	if CodeExecutionRetainedOutputPipe != wantCode {
		t.Fatalf("retained-output code=%q want=%q",
			CodeExecutionRetainedOutputPipe, wantCode)
	}
	err := ErrRetainedOutputPipe(exec.ErrWaitDelay)
	if err.Code != CodeExecutionRetainedOutputPipe {
		t.Fatalf("error code=%q", err.Code)
	}
	if !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("retained-output error does not wrap exec.ErrWaitDelay: %v", err)
	}
}

func TestRetainedOutputResultWireContract(t *testing.T) {
	result := Result{
		ExitCode:         0,
		OutputIncomplete: true,
		OutputTruncated:  false,
		Error:            ErrRetainedOutputPipe(exec.ErrWaitDelay),
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if !strings.Contains(string(encoded), `"OutputIncomplete":true`) {
		t.Fatalf("wire result omits OutputIncomplete: %s", encoded)
	}
	if strings.Contains(string(encoded), `"OutputTruncated":true`) {
		t.Fatalf("retained output overloaded OutputTruncated: %s", encoded)
	}

	var roundTrip Result
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !roundTrip.OutputIncomplete || roundTrip.ExitCode != 0 {
		t.Fatalf("round-trip result=%+v", roundTrip)
	}

	var legacy Result
	if err := json.Unmarshal([]byte(`{"ExitCode":0,"OutputTruncated":false}`),
		&legacy); err != nil {
		t.Fatalf("unmarshal legacy result: %v", err)
	}
	if legacy.OutputIncomplete {
		t.Fatal("legacy payload unexpectedly reports incomplete output")
	}
}
