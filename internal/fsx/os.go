//go:build linux || darwin

package fsx

import (
	"io/fs"
	"os"
	"syscall"
)

// OS is the real file system.
type OS struct{}

var _ FS = OS{}

// osFlags maps the portable flags of this package to the os package's. The values are
// equal on every supported platform; the mapping keeps that an assertion, not a hope.
func osFlags(flag int) int {
	out := 0
	if flag&O_WRONLY != 0 {
		out |= os.O_WRONLY
	}
	if flag&O_RDWR != 0 {
		out |= os.O_RDWR
	}
	if flag&O_CREATE != 0 {
		out |= os.O_CREATE
	}
	if flag&O_EXCL != 0 {
		out |= os.O_EXCL
	}
	return out | syscall.O_NOFOLLOW
}

func (OS) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	f, err := os.OpenFile(name, osFlags(flag), perm)
	if err != nil {
		return nil, err
	}
	return osFile{f}, nil
}

func (OS) Mkdir(name string, perm fs.FileMode) error { return os.Mkdir(name, perm) }

func (OS) ReadDir(name string) ([]string, error) {
	ents, err := os.ReadDir(name) // sorted by name
	if err != nil {
		return nil, err
	}
	names := make([]string, len(ents))
	for i, e := range ents {
		names[i] = e.Name()
	}
	return names, nil
}

func (OS) Lstat(name string) (fs.FileInfo, error) { return os.Lstat(name) }
func (OS) Rename(oldpath, newpath string) error   { return os.Rename(oldpath, newpath) }
func (OS) Remove(name string) error               { return os.Remove(name) }

func (OS) SyncDir(name string) error {
	d, err := os.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	if err := d.Sync(); err != nil {
		_ = d.Close()
		return err
	}
	return d.Close()
}

type osFile struct{ *os.File }

func (f osFile) TryLock() error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
		return ErrLocked
	}
	return err
}
