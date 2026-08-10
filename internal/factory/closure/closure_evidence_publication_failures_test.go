// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeEvidencePublicationFilesystem is a deterministic
// implementation of the evidencePublicationFilesystem seam.
// Tests toggle `fail` to inject failures at any I/O stage.
type fakeEvidencePublicationFilesystem struct {
	defaultFS evidencePublicationFilesystem
	fail      func(op string) error
}

func (f fakeEvidencePublicationFilesystem) err(op string) error {
	if f.fail == nil {
		return nil
	}
	return f.fail(op)
}

func (f fakeEvidencePublicationFilesystem) createTemp(root *os.Root, mode fs.FileMode) (string, *os.File, error) {
	if err := f.err("createTemp"); err != nil {
		return "", nil, err
	}
	return f.defaultFS.createTemp(root, mode)
}
func (f fakeEvidencePublicationFilesystem) writeAll(file *os.File, data []byte) error {
	if err := f.err("writeAll"); err != nil {
		return err
	}
	return f.defaultFS.writeAll(file, data)
}
func (f fakeEvidencePublicationFilesystem) syncFile(file *os.File) error {
	if err := f.err("syncFile"); err != nil {
		return err
	}
	return f.defaultFS.syncFile(file)
}
func (f fakeEvidencePublicationFilesystem) closeFile(file *os.File) error {
	if err := f.err("closeFile"); err != nil {
		return err
	}
	return f.defaultFS.closeFile(file)
}
func (f fakeEvidencePublicationFilesystem) link(root *os.Root, old, new string) error {
	if err := f.err("link:" + new); err != nil {
		return err
	}
	return f.defaultFS.link(root, old, new)
}
func (f fakeEvidencePublicationFilesystem) rename(root *os.Root, old, new string) error {
	if err := f.err("rename"); err != nil {
		return err
	}
	return f.defaultFS.rename(root, old, new)
}
func (f fakeEvidencePublicationFilesystem) unlink(root *os.Root, name string) error {
	if err := f.err("unlink"); err != nil {
		return err
	}
	return f.defaultFS.unlink(root, name)
}
func (f fakeEvidencePublicationFilesystem) readFile(root *os.Root, name string) ([]byte, error) {
	if err := f.err("readFile:" + name); err != nil {
		return nil, err
	}
	return f.defaultFS.readFile(root, name)
}
func (f fakeEvidencePublicationFilesystem) openDir(root *os.Root) (*os.File, error) {
	if err := f.err("openDir"); err != nil {
		return nil, err
	}
	return f.defaultFS.openDir(root)
}
func (f fakeEvidencePublicationFilesystem) syncDir(dir *os.File) error {
	if err := f.err("syncDir"); err != nil {
		return err
	}
	return f.defaultFS.syncDir(dir)
}
func (f fakeEvidencePublicationFilesystem) closeDir(dir *os.File) error {
	if err := f.err("closeDir"); err != nil {
		return err
	}
	return f.defaultFS.closeDir(dir)
}

// TestClosureEvidencePublicationFailureMatrix walks the
// deterministic I/O failure surface. Each row asserts:
//   - the final state matches the ACT contract;
//   - on a pre-link failure, no temp files remain AND
//     destination + sidecar are absent;
//   - on a post-link failure (e.g. sidecar link failure),
//     JSON may be visible and the state reflects the truth.
func TestClosureEvidencePublicationFailureMatrix(t *testing.T) {
	cases := []struct {
		name      string
		fail      func(op string) error
		wantState EvidencePublicationState
		wantJSON  bool
		wantSide  bool
	}{
		{name: "createTemp_failure", fail: func(op string) error {
			if op == "createTemp" {
				return errors.New("csprng")
			}
			return nil
		}, wantState: EvidencePublicationNotPublished, wantJSON: false, wantSide: false},
		{name: "writeAll_failure", fail: func(op string) error {
			if op == "writeAll" {
				return errors.New("eio")
			}
			return nil
		}, wantState: EvidencePublicationNotPublished, wantJSON: false, wantSide: false},
		{name: "syncFile_failure", fail: func(op string) error {
			if op == "syncFile" {
				return errors.New("eio")
			}
			return nil
		}, wantState: EvidencePublicationNotPublished, wantJSON: false, wantSide: false},
		{name: "closeFile_failure", fail: func(op string) error {
			if op == "closeFile" {
				return errors.New("eio")
			}
			return nil
		}, wantState: EvidencePublicationNotPublished, wantJSON: false, wantSide: false},
		{name: "json_link_failure", fail: func(op string) error {
			if op == "link:"+"evidence.json" {
				return errors.New("eexist")
			}
			return nil
		}, wantState: EvidencePublicationNotPublished, wantJSON: false, wantSide: false},
		{name: "sidecar_link_failure", fail: func(op string) error {
			if op == "link:evidence.json.sha256" {
				return errors.New("eexist")
			}
			return nil
		}, wantState: EvidencePublicationJSONVisible, wantJSON: true, wantSide: false},
		{name: "openDir_failure", fail: func(op string) error {
			if op == "openDir" {
				return errors.New("eio")
			}
			return nil
		}, wantState: EvidencePublicationPairVisibleDurabilityUnconfirmed, wantJSON: true, wantSide: true},
		{name: "syncDir_failure", fail: func(op string) error {
			if op == "syncDir" {
				return errors.New("eio")
			}
			return nil
		}, wantState: EvidencePublicationPairVisibleDurabilityUnconfirmed, wantJSON: true, wantSide: true},
		{name: "closeDir_failure", fail: func(op string) error {
			if op == "closeDir" {
				return errors.New("eio")
			}
			return nil
		}, wantState: EvidencePublicationPairVisibleDurabilityUnconfirmed, wantJSON: true, wantSide: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fx := newEvidencePublicationFixture(t)
			auth := prepareFromEvidencePublicationFixture(t, fx)
			auth.setPublicationFilesystemForTest(fakeEvidencePublicationFilesystem{defaultFS: defaultEvidencePublicationFilesystem{random: deterministicReader(t)}, fail: tc.fail})
			candidate := evidenceOnlyCandidate(t)
			res := auth.Publish(candidate)
			if res.State != tc.wantState {
				t.Fatalf("state = %s, want %s (err=%v)", res.State, tc.wantState, res.Err)
			}
			_, jsonErr := os.Lstat(fx.json)
			_, sideErr := os.Lstat(fx.sidecar)
			hasJSON := jsonErr == nil
			hasSide := sideErr == nil
			if hasJSON != tc.wantJSON {
				t.Fatalf("json presence = %v, want %v (err=%v)", hasJSON, tc.wantJSON, jsonErr)
			}
			if hasSide != tc.wantSide {
				t.Fatalf("sidecar presence = %v, want %v (err=%v)", hasSide, tc.wantSide, sideErr)
			}
			if tc.wantState == EvidencePublicationNotPublished {
				entries, _ := os.ReadDir(fx.outside)
				for _, e := range entries {
					if strings.HasPrefix(e.Name(), ".tmp-closure-") {
						t.Fatalf("temp residue after pre-link failure: %s", e.Name())
					}
				}
			}
		})
	}
}

// evidenceRaceNoReplaceFS simulates a competing writer that
// appears between staging and link.
type evidenceRaceNoReplaceFS struct {
	defaultFS  evidencePublicationFilesystem
	raceOnLink string
	existing   []byte
}

func (r *evidenceRaceNoReplaceFS) createTemp(root *os.Root, mode fs.FileMode) (string, *os.File, error) {
	return r.defaultFS.createTemp(root, mode)
}
func (r *evidenceRaceNoReplaceFS) writeAll(f *os.File, b []byte) error {
	return r.defaultFS.writeAll(f, b)
}
func (r *evidenceRaceNoReplaceFS) syncFile(f *os.File) error  { return r.defaultFS.syncFile(f) }
func (r *evidenceRaceNoReplaceFS) closeFile(f *os.File) error { return r.defaultFS.closeFile(f) }
func (r *evidenceRaceNoReplaceFS) link(root *os.Root, old, new string) error {
	if filepath.Base(new) == filepath.Base(r.raceOnLink) {
		if err := os.WriteFile(r.raceOnLink, r.existing, 0o644); err != nil {
			return err
		}
	}
	return r.defaultFS.link(root, old, new)
}
func (r *evidenceRaceNoReplaceFS) rename(root *os.Root, old, new string) error {
	return r.defaultFS.rename(root, old, new)
}
func (r *evidenceRaceNoReplaceFS) unlink(root *os.Root, name string) error {
	return r.defaultFS.unlink(root, name)
}
func (r *evidenceRaceNoReplaceFS) readFile(root *os.Root, name string) ([]byte, error) {
	return r.defaultFS.readFile(root, name)
}
func (r *evidenceRaceNoReplaceFS) openDir(root *os.Root) (*os.File, error) {
	return r.defaultFS.openDir(root)
}
func (r *evidenceRaceNoReplaceFS) syncDir(f *os.File) error  { return r.defaultFS.syncDir(f) }
func (r *evidenceRaceNoReplaceFS) closeDir(f *os.File) error { return r.defaultFS.closeDir(f) }

// TestClosureEvidencePublicationNoReplaceRace is the
// race-safety proof. After staging but before link, a
// competing writer pre-creates the destination; the publisher
// must refuse without replacing the existing bytes.
func TestClosureEvidencePublicationNoReplaceRace(t *testing.T) {
	fx := newEvidencePublicationFixture(t)
	auth := prepareFromEvidencePublicationFixture(t, fx)
	preexisting := []byte("PREEXISTING")
	if err := os.WriteFile(fx.json, preexisting, 0o644); err != nil {
		t.Fatal(err)
	}
	raceFS := &evidenceRaceNoReplaceFS{
		defaultFS:  defaultEvidencePublicationFilesystem{random: deterministicReader(t)},
		raceOnLink: fx.json,
		existing:   preexisting,
	}
	auth.setPublicationFilesystemForTest(raceFS)
	candidate := evidenceOnlyCandidate(t)
	res := auth.Publish(candidate)
	if res.State != EvidencePublicationNotPublished {
		t.Fatalf("state = %s, want not_published (err=%v)", res.State, res.Err)
	}
	got, err := os.ReadFile(fx.json)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(preexisting) {
		t.Fatalf("existing bytes replaced: got %q, want %q", got, preexisting)
	}
}
