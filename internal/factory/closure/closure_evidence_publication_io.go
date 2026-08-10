// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// evidencePublicationFilesystem abstracts the I/O surface used
// by the B3 authority. All operations are rooted; the
// production wiring uses `*os.Root` directly.
type evidencePublicationFilesystem interface {
	createTemp(root *os.Root, mode fs.FileMode) (relName string, file *os.File, err error)
	writeAll(file *os.File, data []byte) error
	syncFile(file *os.File) error
	closeFile(file *os.File) error
	// link creates a new name at `newname` for the existing
	// rooted file `oldname`. On POSIX this is `linkat`. The
	// call must refuse to replace an existing non-directory
	// entry at `newname`; the production implementation uses
	// `os.Root.Link`, which delegates to `link(2)` and
	// therefore enforces no-replace at the kernel boundary.
	link(root *os.Root, oldname, newname string) error
	rename(root *os.Root, oldname, newname string) error
	unlink(root *os.Root, name string) error
	readFile(root *os.Root, name string) ([]byte, error)
	openDir(root *os.Root) (*os.File, error)
	syncDir(dir *os.File) error
	closeDir(dir *os.File) error
}

// defaultEvidencePublicationFilesystem is the production I/O surface.
type defaultEvidencePublicationFilesystem struct {
	random io.Reader
}

// randomTempName returns a 32-character hex basename.
func randomTempName(r io.Reader) (string, error) {
	if r == nil {
		r = rand.Reader
	}
	var raw [16]byte
	if _, err := io.ReadFull(r, raw[:]); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	return ".tmp-closure-" + hex.EncodeToString(raw[:]), nil
}

func (f defaultEvidencePublicationFilesystem) createTemp(root *os.Root, mode fs.FileMode) (string, *os.File, error) {
	name, err := randomTempName(f.random)
	if err != nil {
		return "", nil, err
	}
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return "", nil, err
	}
	return name, file, nil
}

func (defaultEvidencePublicationFilesystem) writeAll(file *os.File, data []byte) error {
	_, err := file.Write(data)
	return err
}

func (defaultEvidencePublicationFilesystem) syncFile(file *os.File) error  { return file.Sync() }
func (defaultEvidencePublicationFilesystem) closeFile(file *os.File) error { return file.Close() }
func (defaultEvidencePublicationFilesystem) link(root *os.Root, old, new string) error {
	return root.Link(old, new)
}
func (defaultEvidencePublicationFilesystem) rename(root *os.Root, old, new string) error {
	return root.Rename(old, new)
}
func (defaultEvidencePublicationFilesystem) unlink(root *os.Root, name string) error {
	return root.Remove(name)
}
func (defaultEvidencePublicationFilesystem) readFile(root *os.Root, name string) ([]byte, error) {
	return root.ReadFile(name)
}
func (defaultEvidencePublicationFilesystem) openDir(root *os.Root) (*os.File, error) {
	return root.Open(".")
}
func (defaultEvidencePublicationFilesystem) syncDir(dir *os.File) error  { return dir.Sync() }
func (defaultEvidencePublicationFilesystem) closeDir(dir *os.File) error { return dir.Close() }
