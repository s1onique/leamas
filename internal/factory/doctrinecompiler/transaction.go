package doctrinecompiler

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// transaction is the transactional apply record for a single Compile.
//
// The transaction records three kinds of pre-state:
//
//   - filePre: per-path existence, mode, and bytes (regular files only).
//   - dirPre:  parent directories that existed before the apply and
//     therefore must not be removed by rollback.
//   - dirsCreated: directories that the apply created and that rollback
//     may remove (only when empty).
//
// It also records, in apply order, every mutation that succeeded
// (mutationJournal). Rollback walks the journal in reverse.
//
// The transaction is intentionally narrow: it only knows about regular
// files and the lock file. Symlinks, special files, and the target root
// are not within its scope.
type transaction struct {
	root        string
	filePre     map[string]filePreState
	dirPre      map[string]struct{}
	dirsCreated []string
	mutationLog []mutationRecord
	fs          fsOps
}

// filePreState captures the pre-apply state of a single regular file.
type filePreState struct {
	existed bool
	mode    os.FileMode
	bytes   []byte
}

// mutationRecord describes a single mutation the apply phase performed.
type mutationRecord struct {
	kind  mutationKind
	abs   string
	mode  os.FileMode
	bytes []byte
}

// mutationKind enumerates the operations the transaction can record.
type mutationKind int

const (
	mutCreate mutationKind = iota
	mutReplace
	mutRemove
)

// fsOps abstracts the filesystem calls used by the rollback path.
// Only the operations the rollback actually invokes appear here:
// Lstat/ReadDir/Remove/MkdirAll/WriteFile. ReadFile is intentionally
// absent because the rollback never reads from disk; the snapshot
// phase uses os.ReadFile directly during beginTransaction.
//
// Tests can replace individual functions to inject deterministic
// failures during rollback and prove that the failures are surfaced
// to the caller via errors.Join rather than silently discarded.
type fsOps struct {
	Lstat     func(string) (os.FileInfo, error)
	ReadDir   func(string) ([]os.DirEntry, error)
	Remove    func(string) error
	MkdirAll  func(string, os.FileMode) error
	WriteFile func(path string, data []byte, mode os.FileMode) (string, error)
}

// runtimeFsOps is the production filesystem interface used by rollback.
// Only the operations the rollback actually invokes are wired; see
// the fsOps type comment for the rationale.
var runtimeFsOps = fsOps{
	Lstat:     os.Lstat,
	ReadDir:   os.ReadDir,
	Remove:    os.Remove,
	MkdirAll:  os.MkdirAll,
	WriteFile: writeAtomicFile,
}

// beginTransaction snapshots the pre-apply state for the set of paths
// the apply is about to touch. The apply must not run until this
// snapshot is complete.
func beginTransaction(root string, absPaths []string) (*transaction, error) {
	t := &transaction{
		root:    root,
		filePre: make(map[string]filePreState, len(absPaths)),
		dirPre:  make(map[string]struct{}),
		fs:      runtimeFsOps,
	}
	for _, abs := range absPaths {
		snap, err := snapshotFile(abs)
		if err != nil {
			return nil, err
		}
		t.filePre[abs] = snap
		if err := markPreDirs(t, abs); err != nil {
			return nil, err
		}
	}
	return t, nil
}

// snapshotFile captures the existence, mode, and bytes of a path.
func snapshotFile(abs string) (filePreState, error) {
	lst, err := os.Lstat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return filePreState{existed: false}, nil
		}
		return filePreState{}, err
	}
	m := lst.Mode()
	if m&os.ModeSymlink != 0 {
		return filePreState{existed: true, mode: m}, nil
	}
	if !m.IsRegular() {
		return filePreState{existed: true, mode: m}, nil
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return filePreState{}, err
	}
	return filePreState{existed: true, mode: m.Perm(), bytes: data}, nil
}

// markPreDirs records every existing directory ancestor of abs.
func markPreDirs(t *transaction, abs string) error {
	cur := filepath.Dir(abs)
	for cur != "" && cur != "." && cur != string(filepath.Separator) {
		rel, err := filepath.Rel(t.root, cur)
		if err != nil || strings.HasPrefix(rel, "..") {
			break
		}
		if rel == "" {
			break
		}
		lst, err := os.Lstat(cur)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if lst.IsDir() {
			t.dirPre[cur] = struct{}{}
		}
		cur = filepath.Dir(cur)
	}
	return nil
}

// recordCreate appends a create mutation to the journal.
func (t *transaction) recordCreate(abs string, mode os.FileMode, data []byte) {
	t.mutationLog = append(t.mutationLog, mutationRecord{
		kind:  mutCreate,
		abs:   abs,
		mode:  mode,
		bytes: append([]byte(nil), data...),
	})
}

// recordReplace appends a replace mutation to the journal.
func (t *transaction) recordReplace(abs string, mode os.FileMode, data []byte) {
	t.mutationLog = append(t.mutationLog, mutationRecord{
		kind:  mutReplace,
		abs:   abs,
		mode:  mode,
		bytes: append([]byte(nil), data...),
	})
}

// recordRemove appends a remove mutation to the journal.
func (t *transaction) recordRemove(abs string) {
	t.mutationLog = append(t.mutationLog, mutationRecord{
		kind: mutRemove,
		abs:  abs,
	})
}

// noteDirCreated appends a directory the apply created.
func (t *transaction) noteDirCreated(abs string) {
	if _, ok := t.dirPre[abs]; ok {
		return
	}
	for _, d := range t.dirsCreated {
		if d == abs {
			return
		}
	}
	t.dirsCreated = append(t.dirsCreated, abs)
}

// rollback walks the mutation journal in reverse and restores the
// pre-apply state. It joins every step that fails (other than the
// expected fs.ErrNotExist cases) into the returned error.
func (t *transaction) rollback() error {
	var rerr error
	for i := len(t.mutationLog) - 1; i >= 0; i-- {
		m := t.mutationLog[i]
		switch m.kind {
		case mutCreate:
			if err := removeRegularIfExists(t.fs, m.abs); err != nil {
				rerr = joinErr(rerr, fmt.Errorf("rollback remove created %s: %w", m.abs, err))
			}
		case mutRemove:
			pre, ok := t.filePre[m.abs]
			if !ok {
				rerr = joinErr(rerr, fmt.Errorf("rollback recreate %s: no pre-state", m.abs))
				continue
			}
			if !pre.existed {
				continue
			}
			if err := t.fs.MkdirAll(filepath.Dir(m.abs), 0o755); err != nil {
				rerr = joinErr(rerr, fmt.Errorf("rollback mkdir parent for %s: %w", m.abs, err))
				continue
			}
			if _, err := t.fs.WriteFile(m.abs, pre.bytes, pre.mode); err != nil {
				rerr = joinErr(rerr, fmt.Errorf("rollback write %s: %w", m.abs, err))
				continue
			}
		case mutReplace:
			pre, ok := t.filePre[m.abs]
			if !ok {
				rerr = joinErr(rerr, fmt.Errorf("rollback replace %s: no pre-state", m.abs))
				continue
			}
			if !pre.existed {
				rerr = joinErr(rerr, fmt.Errorf("rollback replace %s: unexpected pre-state missing", m.abs))
				continue
			}
			if err := t.fs.MkdirAll(filepath.Dir(m.abs), 0o755); err != nil {
				rerr = joinErr(rerr, fmt.Errorf("rollback parent for %s: %w", m.abs, err))
				continue
			}
			if _, err := t.fs.WriteFile(m.abs, pre.bytes, pre.mode); err != nil {
				rerr = joinErr(rerr, fmt.Errorf("rollback write %s: %w", m.abs, err))
				continue
			}
		}
	}
	// Remove transaction-created directories deepest-first. Never
	// remove the target root. Never remove a directory that existed
	// before compilation. Only fs.ErrNotExist is ignorable; every
	// other error is joined into the rollback failure so the caller
	// never silently loses it.
	sortedDirs := append([]string(nil), t.dirsCreated...)
	sort.Slice(sortedDirs, func(i, j int) bool {
		return depth(sortedDirs[i]) > depth(sortedDirs[j])
	})
	for _, d := range sortedDirs {
		if d == t.root {
			continue
		}
		if _, ok := t.dirPre[d]; ok {
			continue
		}
		lst, err := t.fs.Lstat(d)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			rerr = joinErr(rerr, fmt.Errorf("rollback lstat %s: %w", d, err))
			continue
		}
		if !lst.IsDir() {
			continue
		}
		entries, err := t.fs.ReadDir(d)
		if err != nil {
			rerr = joinErr(rerr, fmt.Errorf("rollback readdir %s: %w", d, err))
			continue
		}
		if len(entries) != 0 {
			continue
		}
		if err := t.fs.Remove(d); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			rerr = joinErr(rerr, fmt.Errorf("rollback rmdir %s: %w", d, err))
		}
	}
	return rerr
}

// removeRegularIfExists removes a regular file using the supplied
// fsOps. Symlinks and missing files are not errors; the apply phase
// never creates them. fs.ErrNotExist is the only error class
// treated as ignorable here; every other error is returned to the
// caller so the rollback can join it.
func removeRegularIfExists(ops fsOps, abs string) error {
	lst, err := ops.Lstat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if lst.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to remove symlink: %s", abs)
	}
	if !lst.Mode().IsRegular() {
		return fmt.Errorf("refusing to remove non-regular: %s", abs)
	}
	if err := ops.Remove(abs); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// depth returns the path depth (number of separators + 1).
func depth(p string) int {
	if p == "" {
		return 0
	}
	return strings.Count(p, string(filepath.Separator)) + 1
}

// joinErr joins two errors, preserving both via errors.Join. Returns
// nil if both are nil.
func joinErr(a, b error) error {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return errors.Join(a, b)
}
