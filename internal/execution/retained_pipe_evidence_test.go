//go:build unix || darwin || linux

package execution

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	retainedPipeMode       = "held-descriptor"
	retainedPipeWaitDelay  = 200 * time.Millisecond
	retainedPipeLowerSlack = 50 * time.Millisecond
	retainedPipeUpperSlack = 500 * time.Millisecond
	retainedPipeTimeout    = 10 * time.Second
)

type descriptorReadyInfo struct {
	Role        string
	PID         int
	PPID        int
	PGID        int
	Descriptors descriptorSet
}

type parentExitEvidence struct {
	PID      int
	UnixNano int64
}

type retainedPipeProbe struct {
	PID      int    `json:"pid"`
	Sequence uint64 `json:"sequence"`
	UnixNano int64  `json:"unix_nano"`
	Bytes    int    `json:"bytes"`
	Error    string `json:"error,omitempty"`
}

type retainedPipeHandoff struct {
	Parent    PIDRecord
	Child     PIDRecord
	Ready     descriptorReadyInfo
	Exit      parentExitEvidence
	PostProbe retainedPipeProbe
}

func waitForRetainedPipeHandoff(t *testing.T, verifier *processVerifier) retainedPipeHandoff {
	t.Helper()
	deadline := time.Now().Add(retainedPipeTimeout)
	readyPath, err := waitForSinglePath(filepath.Join(verifier.ReadyDir(),
		"*.descriptor-ready.ready"), deadline)
	if err != nil {
		t.Fatalf("descriptor readiness: %v", err)
	}
	readyBytes, err := os.ReadFile(readyPath)
	if err != nil {
		t.Fatalf("read descriptor readiness: %v", err)
	}
	ready, err := parseDescriptorReadyContent(string(readyBytes))
	if err != nil {
		t.Fatalf("parse descriptor readiness: %v", err)
	}

	parent, child, err := waitForRetainedPipeRecords(verifier, deadline)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateRetainedPipeTopology(parent, child, ready); err != nil {
		t.Fatalf("invalid retained-pipe topology: %v", err)
	}

	exitPath, err := waitForSinglePath(filepath.Join(verifier.ReadyDir(),
		"parent-exit-imminent.*"), deadline)
	if err != nil {
		t.Fatalf("parent exit evidence: %v", err)
	}
	exitBytes, err := os.ReadFile(exitPath)
	if err != nil {
		t.Fatalf("read parent exit evidence: %v", err)
	}
	exit, err := parseParentExitEvidence(string(exitBytes))
	if err != nil {
		t.Fatalf("parse parent exit evidence: %v", err)
	}
	if exit.PID != parent.PID {
		t.Fatalf("exit evidence pid=%d, parent pid=%d", exit.PID, parent.PID)
	}

	if err := waitForProcessAbsent(verifier, parent.PID, deadline); err != nil {
		t.Fatal(err)
	}
	// A probe timestamped after this observation is necessarily
	// post-parent-exit, not merely after the pre-exit handoff timestamp.
	parentAbsentUnixNano := time.Now().UnixNano()
	if err := requireNonZombieProcess(child.PID); err != nil {
		t.Fatalf("descriptor holder is not live: %v", err)
	}
	probePath := filepath.Join(verifier.ReadyDir(),
		fmt.Sprintf("%d.retained-pipe-probes.jsonl", child.PID))
	probe, err := waitForSuccessfulProbeAfter(probePath, parentAbsentUnixNano, deadline)
	if err != nil {
		t.Fatalf("post-parent-exit pipe probe: %v", err)
	}
	return retainedPipeHandoff{
		Parent: parent, Child: child, Ready: ready, Exit: exit, PostProbe: probe,
	}
}

func waitForRetainedPipeRecords(verifier *processVerifier,
	deadline time.Time,
) (PIDRecord, PIDRecord, error) {
	for time.Now().Before(deadline) {
		records, err := verifier.parseManifest()
		if err == nil {
			var parent, child PIDRecord
			for _, record := range records {
				switch record.Role {
				case "parent":
					parent = record
				case "child":
					child = record
				}
			}
			if parent.PID != 0 && child.PID != 0 {
				return parent, child, nil
			}
		}
		time.Sleep(readinessPollInterval)
	}
	return PIDRecord{}, PIDRecord{}, fmt.Errorf("parent/child manifest deadline exceeded")
}

func validateRetainedPipeTopology(parent, child PIDRecord,
	ready descriptorReadyInfo,
) error {
	if parent.Descriptors == nil || child.Descriptors == nil {
		return fmt.Errorf("missing manifest descriptor identity")
	}
	if parent.PGID == 0 || parent.PGID != parent.PID ||
		parent.PGID != child.PGID {
		return fmt.Errorf("parent pid=%d pgid=%d child pgid=%d",
			parent.PID, parent.PGID, child.PGID)
	}
	if child.PPID != parent.PID {
		return fmt.Errorf("child ppid=%d parent pid=%d", child.PPID, parent.PID)
	}
	if ready.Role != "child" || ready.PID != child.PID ||
		ready.PPID != parent.PID || ready.PGID != parent.PGID {
		return fmt.Errorf("readiness identity=%+v parent=%d child=%d pgid=%d",
			ready, parent.PID, child.PID, parent.PGID)
	}
	if ready.Descriptors != *child.Descriptors {
		return fmt.Errorf("readiness descriptors=%+v manifest descriptors=%+v",
			ready.Descriptors, *child.Descriptors)
	}
	pairs := []struct {
		name          string
		parent, child descriptorIdentity
	}{
		{name: "fd1", parent: parent.Descriptors.FD1, child: child.Descriptors.FD1},
		{name: "fd2", parent: parent.Descriptors.FD2, child: child.Descriptors.FD2},
	}
	for _, pair := range pairs {
		if pair.parent != pair.child {
			return fmt.Errorf("%s mismatch: parent=%+v child=%+v",
				pair.name, pair.parent, pair.child)
		}
		if err := validatePipeIdentity(pair.parent); err != nil {
			return fmt.Errorf("%s: %w", pair.name, err)
		}
	}
	return nil
}

func validatePipeIdentity(identity descriptorIdentity) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	if !strings.HasPrefix(identity.Target, "pipe:[") ||
		!strings.HasSuffix(identity.Target, "]") {
		return fmt.Errorf("target %q is not a Linux pipe", identity.Target)
	}
	text := strings.TrimSuffix(strings.TrimPrefix(identity.Target, "pipe:["), "]")
	inode, err := strconv.ParseUint(text, 10, 64)
	if err != nil || inode == 0 || inode != identity.Ino {
		return fmt.Errorf("target inode=%q fstat inode=%d", text, identity.Ino)
	}
	if identity.Dev == 0 {
		return fmt.Errorf("fstat device is zero")
	}
	return nil
}

func parseDescriptorReadyContent(content string) (descriptorReadyInfo, error) {
	values, err := parseKeyValueEvidence(content)
	if err != nil {
		return descriptorReadyInfo{}, err
	}
	integer := func(key string) (int, error) {
		value, err := strconv.Atoi(values[key])
		if err != nil || value <= 0 {
			return 0, fmt.Errorf("invalid %s=%q", key, values[key])
		}
		return value, nil
	}
	unsigned := func(key string) (uint64, error) {
		value, err := strconv.ParseUint(values[key], 10, 64)
		if err != nil || value == 0 {
			return 0, fmt.Errorf("invalid %s=%q", key, values[key])
		}
		return value, nil
	}
	var info descriptorReadyInfo
	info.Role = values["role"]
	if info.Role == "" {
		return info, fmt.Errorf("missing role")
	}
	for key, target := range map[string]*int{
		"pid": &info.PID, "ppid": &info.PPID, "pgid": &info.PGID,
	} {
		*target, err = integer(key)
		if err != nil {
			return info, err
		}
	}
	info.Descriptors.FD1.Target = values["fd1_target"]
	info.Descriptors.FD2.Target = values["fd2_target"]
	for key, target := range map[string]*uint64{
		"fd1_dev": &info.Descriptors.FD1.Dev,
		"fd1_ino": &info.Descriptors.FD1.Ino,
		"fd2_dev": &info.Descriptors.FD2.Dev,
		"fd2_ino": &info.Descriptors.FD2.Ino,
	} {
		*target, err = unsigned(key)
		if err != nil {
			return info, err
		}
	}
	if info.Descriptors.FD1.Target == "" || info.Descriptors.FD2.Target == "" {
		return info, fmt.Errorf("missing descriptor target")
	}
	return info, nil
}

func parseParentExitEvidence(content string) (parentExitEvidence, error) {
	values, err := parseKeyValueEvidence(content)
	if err != nil {
		return parentExitEvidence{}, err
	}
	pid, err := strconv.Atoi(values["pid"])
	if err != nil || pid <= 0 {
		return parentExitEvidence{}, fmt.Errorf("invalid pid=%q", values["pid"])
	}
	nano, err := strconv.ParseInt(values["parent_exit_imminent_unix_nano"], 10, 64)
	if err != nil || nano <= 0 {
		return parentExitEvidence{}, fmt.Errorf("invalid exit timestamp")
	}
	return parentExitEvidence{PID: pid, UnixNano: nano}, nil
}

func parseKeyValueEvidence(content string) (map[string]string, error) {
	values := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" || value == "" {
			return nil, fmt.Errorf("malformed evidence line %q", line)
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("duplicate evidence key %q", key)
		}
		values[key] = value
	}
	return values, nil
}

func waitForSuccessfulProbeAfter(path string, unixNano int64,
	deadline time.Time,
) (retainedPipeProbe, error) {
	for time.Now().Before(deadline) {
		probes, _ := readRetainedPipeProbes(path)
		for _, probe := range probes {
			if probe.Error == "" && probe.Bytes > 0 && probe.UnixNano >= unixNano {
				return probe, nil
			}
		}
		time.Sleep(readinessPollInterval)
	}
	return retainedPipeProbe{}, fmt.Errorf("no successful probe after %d", unixNano)
}

func waitForProbeError(path string, deadline time.Time) (retainedPipeProbe, error) {
	for time.Now().Before(deadline) {
		probes, _ := readRetainedPipeProbes(path)
		for _, probe := range probes {
			if probe.Error != "" {
				return probe, nil
			}
		}
		time.Sleep(readinessPollInterval)
	}
	return retainedPipeProbe{}, fmt.Errorf("final pipe write error not recorded")
}

func readRetainedPipeProbes(path string) ([]retainedPipeProbe, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var probes []retainedPipeProbe
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var probe retainedPipeProbe
		if err := json.Unmarshal(scanner.Bytes(), &probe); err != nil {
			return probes, err
		}
		probes = append(probes, probe)
	}
	return probes, scanner.Err()
}
