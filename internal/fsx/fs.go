// Package fsx is the file-system seam every file operation in quill goes through: an interface,
// a real implementation (OS) and a crash simulator (Sim) that models what a power loss does to
// unsynced writes, so recovery can be tested against outcomes a SIGKILL cannot produce.
//
// The interface is deliberately small. Paths are plain strings; a File is positional (ReadAt,
// WriteAt) because quill never appends blindly: a flush writes at the durable offset it knows.
// Every open refuses to follow a symbolic link at the final path component.
package fsx

import (
	"errors"
	"io"
	"io/fs"
)

// FS is the set of file operations quill uses.
type FS interface {
	// OpenFile opens name with os.OpenFile semantics, never following a symbolic link at the
	// final component (O_NOFOLLOW). flag is a combination of O_RDONLY, O_RDWR, O_WRONLY,
	// O_CREATE and O_EXCL; perm applies to a created file.
	OpenFile(name string, flag int, perm fs.FileMode) (File, error)
	// Mkdir creates one directory; the parent must exist.
	Mkdir(name string, perm fs.FileMode) error
	// ReadDir lists the names in a directory, sorted.
	ReadDir(name string) ([]string, error)
	// Lstat describes name without following a final symbolic link.
	Lstat(name string) (fs.FileInfo, error)
	// Rename moves oldpath to newpath, replacing newpath if it exists. Both must be in the
	// same directory: quill only ever renames a temp file over its neighbour.
	Rename(oldpath, newpath string) error
	// Remove unlinks a file (never a directory).
	Remove(name string) error
	// SyncDir fsyncs a directory, making its entries (creates, renames, unlinks) durable.
	SyncDir(name string) error
}

// File is an open file. Reads and writes are positional.
type File interface {
	io.ReaderAt
	io.WriterAt
	// Sync fsyncs the file's data and size. On darwin this is F_FULLFSYNC.
	Sync() error
	// Truncate sets the file's size; it may extend with zeros.
	Truncate(size int64) error
	// Stat describes the open file (fstat), which cannot be a symbolic link.
	Stat() (fs.FileInfo, error)
	// Chmod sets the file's mode exactly, whatever the process umask.
	Chmod(mode fs.FileMode) error
	// TryLock takes an exclusive, non-blocking advisory lock on the file, which another
	// process's TryLock on the same file then fails with ErrLocked. The lock is released by
	// Close or by the process ending.
	TryLock() error
	Close() error
	// Name is the path the file was opened with.
	Name() string
}

// ErrLocked is returned by TryLock when another holder has the lock.
var ErrLocked = errors.New("fsx: file is locked by another holder")

// Flags, with the values of the os package.
const (
	O_RDONLY = 0x0
	O_WRONLY = 0x1
	O_RDWR   = 0x2
	O_CREATE = 0x40
	O_EXCL   = 0x80
)
