package store

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// LockFileName is the name of the ledger lock file.
//
// The lock is the reader/writer exclusion mechanism that guarantees a
// reader never observes a partial multi-record publication. It is a
// regular file inside the authorized ledger storage: no new database, no
// daemon, no external service.
const LockFileName = ".lock"

// ErrLockBusy is returned when the exclusive lock cannot be acquired.
var ErrLockBusy = errors.New("ledger lock is busy")

// Lock is an advisory reader/writer lock over the ledger storage.
//
// A writer holds the lock in EXCLUSIVE mode for the whole publication. A
// reader holds it in SHARED mode for the whole read.
//
// The exclusion is enforced ENTIRELY by the operating-system advisory file
// lock. There is NO process-wide reentrancy flag: every acquisition opens
// its own file description and takes its own advisory lock. Because an
// advisory file lock is held per open file description, two acquisitions in
// the SAME process conflict exactly like two acquisitions in different
// processes.
//
// The consequences are deliberate:
//
//   - two goroutines of one process that both try to write are mutually
//     exclusive;
//   - two processes that both try to write are mutually exclusive;
//   - a reader never enters while a writer of the SAME process publishes;
//   - a reader never enters while a writer of ANOTHER process publishes.
//
// A caller that must perform a nested operation under the lock it already
// holds passes that ownership EXPLICITLY (see Transaction.Ownership). The
// lock never infers reentrancy from a process-wide flag.
type Lock struct {
	// path is the lock file path.
	path string
	// root is the ledger root the lock guards.
	root string
	// exclusive reports whether the lock is the exclusive writer mode.
	exclusive bool
	// file is the open file description that carries the advisory lock.
	file *os.File
	// released guards against a double release.
	released bool
}

// lockPath returns the path of the lock file for a ledger root.
func lockPath(root string) string {
	return filepath.Join(root, LockFileName)
}

// AcquireRead acquires the shared lock.
//
// The shared lock coordinates with the exclusive writer lock of this
// package. The acquisition BLOCKS until the exclusive lock is released, so
// a reader never enters during a publication.
//
// It never skips the file lock: a reader inside a process that holds the
// exclusive lock is a programming error, and blocking is the safe outcome.
func AcquireRead(root string) (*Lock, error) {
	f, err := openLockFile(root)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &Lock{path: lockPath(root), root: root, file: f}, nil
}

// AcquireWrite acquires the exclusive lock.
//
// The exclusive lock is held for the whole publication. A second writer,
// whether a goroutine of this process or a separate process, cannot
// acquire it while the first one publishes.
//
// The acquisition BLOCKS until the exclusive lock is released. It is never
// reentrant: a nested acquisition under an already-held exclusive lock is
// a caller error, and the caller must pass the explicit transaction
// ownership instead (see Transaction.Ownership).
func AcquireWrite(root string) (*Lock, error) {
	f, err := openLockFile(root)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &Lock{path: lockPath(root), root: root, exclusive: true, file: f}, nil
}

// openLockFile opens the lock file of a ledger root.
func openLockFile(root string) (*os.File, error) {
	path := lockPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
}

// Release releases the lock.
func (l *Lock) Release() {
	if l == nil || l.released {
		return
	}
	l.released = true
	if l.file != nil {
		_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
		_ = l.file.Close()
		l.file = nil
	}
}
