// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"errors"
	"io/fs"
	"os"
	"testing"
)

type closeErrorPublicationFS struct{}

func (closeErrorPublicationFS) createTemp(root *os.Root, mode fs.FileMode) (string, *os.File, error) {
	name := ".verifier-output-close-error.tmp"
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	return name, file, err
}
func (closeErrorPublicationFS) writeAll(file *os.File, data []byte) error {
	_, err := file.Write(data)
	return err
}
func (closeErrorPublicationFS) syncFile(file *os.File) error  { return file.Sync() }
func (closeErrorPublicationFS) closeFile(file *os.File) error { return file.Close() }
func (closeErrorPublicationFS) rename(root *os.Root, oldname, newname string) error {
	return root.Rename(oldname, newname)
}
func (closeErrorPublicationFS) openParent(root *os.Root) (*os.File, error) { return root.Open(".") }
func (closeErrorPublicationFS) syncDirFile(file *os.File) (error, error) {
	_ = file.Close()
	return nil, errors.New("synthetic directory close failure")
}
func (closeErrorPublicationFS) remove(root *os.Root, name string) error { return root.Remove(name) }

func TestPublication_PostPublishCloseFailureIsObserverState(t *testing.T) {
	fx := newPublicationFixture(t)
	fx.authority.setPublicationFilesystemForTest(closeErrorPublicationFS{})
	res := fx.authority.Publish([]byte("payload"))
	if res.State != PublicationPublishedButPostPublishObservationFailed {
		t.Fatalf("state = %s, want post-publish observation failure", res.State)
	}
	if res.Err == nil {
		t.Fatal("post-publish close failure must be surfaced")
	}
	got, err := os.ReadFile(fx.candidate)
	if err != nil || string(got) != "payload" {
		t.Fatalf("published bytes lost: %q, %v", got, err)
	}
}

func TestPublication_CSPRNGFailureDoesNotOpenOrLeaveTemp(t *testing.T) {
	rootPath := t.TempDir()
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	fsys := defaultPublicationFilesystem{random: errorReader{}}
	_, file, err := fsys.createTemp(root, 0o644)
	if err == nil || file != nil {
		t.Fatalf("random failure = %v, file = %v", err, file)
	}
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("random failure left entries: %v", entries)
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("synthetic random source failure") }

func TestRepositoryWorktreeInventory_RejectsDirtyAndRelativeRoots(t *testing.T) {
	root := t.TempDir()
	for _, candidate := range []string{root + string(os.PathSeparator) + ".", "relative"} {
		if _, err := newRepositoryWorktreeInventoryFromCanonical([]string{candidate}); err == nil {
			t.Fatalf("accepted invalid root %q", candidate)
		}
	}
}
