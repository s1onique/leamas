//go:build unix || darwin || linux

package execution

import (
	"strings"
	"testing"
)

func TestParseLinuxProcState(t *testing.T) {
	tests := []struct {
		name    string
		stat    string
		want    byte
		wantErr bool
	}{
		{name: "sleeping", stat: "42 (helper child) S 1 2 3", want: 'S'},
		{name: "running with close paren", stat: "43 (odd) name)) R 1 2", want: 'R'},
		{name: "zombie", stat: "44 (holder) Z 1 2 3", want: 'Z'},
		{name: "missing close", stat: "44 holder Z 1", wantErr: true},
		{name: "missing state", stat: "44 (holder)", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseLinuxProcState(test.stat)
			if (err != nil) != test.wantErr {
				t.Fatalf("error=%v, wantErr=%v", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Fatalf("state=%q, want=%q", got, test.want)
			}
		})
	}
}

func TestValidateRetainedPipeTopology(t *testing.T) {
	fd1 := descriptorIdentity{Target: "pipe:[101]", Dev: 15, Ino: 101}
	fd2 := descriptorIdentity{Target: "pipe:[202]", Dev: 15, Ino: 202}
	set := descriptorSet{FD1: fd1, FD2: fd2}
	baselineParent := PIDRecord{Role: "parent", PID: 11, PGID: 11, Descriptors: &set}
	baselineChild := PIDRecord{
		Role: "child", PID: 12, PPID: 11, PGID: 11, Descriptors: &set,
	}
	baselineReady := descriptorReadyInfo{
		Role: "child", PID: 12, PPID: 11, PGID: 11, Descriptors: set,
	}
	cloneSet := func(value descriptorSet) *descriptorSet { return &value }

	tests := []struct {
		name   string
		mutate func(*PIDRecord, *PIDRecord, *descriptorReadyInfo)
		want   string
	}{
		{name: "valid"},
		{name: "missing parent descriptors", want: "missing", mutate: func(p, _ *PIDRecord, _ *descriptorReadyInfo) {
			p.Descriptors = nil
		}},
		{name: "different group", want: "pgid", mutate: func(_ *PIDRecord, c *PIDRecord, _ *descriptorReadyInfo) {
			c.PGID++
		}},
		{name: "different child fd", want: "fd1 mismatch", mutate: func(_ *PIDRecord, c *PIDRecord, r *descriptorReadyInfo) {
			changed := *c.Descriptors
			changed.FD1.Ino++
			c.Descriptors = &changed
			r.Descriptors = changed
		}},
		{name: "regular file target", want: "not a Linux pipe", mutate: func(p, c *PIDRecord, r *descriptorReadyInfo) {
			changed := *p.Descriptors
			changed.FD1.Target = "/tmp/output"
			p.Descriptors = &changed
			c.Descriptors = cloneSet(changed)
			r.Descriptors = changed
		}},
		{name: "target fstat mismatch", want: "fstat inode", mutate: func(p, c *PIDRecord, r *descriptorReadyInfo) {
			changed := *p.Descriptors
			changed.FD2.Ino++
			p.Descriptors = &changed
			c.Descriptors = cloneSet(changed)
			r.Descriptors = changed
		}},
		{name: "readiness descriptor mismatch", want: "readiness descriptors", mutate: func(_, _ *PIDRecord, r *descriptorReadyInfo) {
			r.Descriptors.FD2.Ino++
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parent := baselineParent
			child := baselineChild
			parent.Descriptors = cloneSet(set)
			child.Descriptors = cloneSet(set)
			ready := baselineReady
			if test.mutate != nil {
				test.mutate(&parent, &child, &ready)
			}
			err := validateRetainedPipeTopology(parent, child, ready)
			if test.want == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func TestParseDescriptorReadyRequiresDescriptorIdentity(t *testing.T) {
	valid := "role=child\npid=12\nppid=11\npgid=11\n" +
		"fd1_target=pipe:[101]\nfd1_dev=15\nfd1_ino=101\n" +
		"fd2_target=pipe:[202]\nfd2_dev=15\nfd2_ino=202\n"
	if _, err := parseDescriptorReadyContent(valid); err != nil {
		t.Fatalf("valid evidence rejected: %v", err)
	}
	if _, err := parseDescriptorReadyContent(strings.ReplaceAll(valid,
		"fd2_target=pipe:[202]\n", "")); err == nil {
		t.Fatal("missing fd2 target accepted")
	}
}
