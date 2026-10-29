package fsx

import (
	"errors"
	"io"
	"io/fs"
	"path"
	"slices"
	"time"
)

// simFile is an open handle.
type simFile struct {
	s        *Sim
	name     string
	ino      *inode
	writable bool
	dead     bool
	closed   bool
	gen      uint64
}

func (f *simFile) check(op Op) error {
	if f.closed {
		return &fs.PathError{Op: string(op), Path: f.name, Err: fs.ErrClosed}
	}
	if f.dead || f.gen != f.s.generation {
		return ErrDead
	}
	return f.s.inject(op, f.name)
}

func (f *simFile) ReadAt(p []byte, off int64) (int, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	if err := f.check(OpRead); err != nil {
		return 0, err
	}
	if off < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: fs.ErrInvalid}
	}
	if off >= int64(len(f.ino.volatile)) {
		return 0, io.EOF
	}
	n := copy(p, f.ino.volatile[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (f *simFile) WriteAt(p []byte, off int64) (int, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	err := f.check(OpWrite)
	var short *ShortWrite
	if errors.As(err, &short) {
		p, err = p[:min(short.N, len(p))], io.ErrShortWrite
	} else if err != nil {
		return 0, err
	}
	if !f.writable || off < 0 {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: fs.ErrInvalid}
	}
	end := off + int64(len(p))
	if end > int64(len(f.ino.volatile)) {
		f.ino.volatile = append(f.ino.volatile, make([]byte, end-int64(len(f.ino.volatile)))...)
		f.ino.sizeDirt = true
	}
	copy(f.ino.volatile[off:end], p)
	for pg := int(off / pageSize); pg <= int((end-1)/pageSize) && len(p) > 0; pg++ {
		f.ino.dirty[pg] = true
	}
	f.ino.mtime = f.s.now()
	return len(p), err
}

func (f *simFile) Sync() error {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	if err := f.check(OpSync); err != nil {
		return err
	}
	f.ino.durable = slices.Clone(f.ino.volatile)
	f.ino.dirty = map[int]bool{}
	f.ino.sizeDirt = false
	return nil
}

func (f *simFile) Truncate(size int64) error {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	if err := f.check(OpTruncate); err != nil {
		return err
	}
	if !f.writable || size < 0 {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: fs.ErrInvalid}
	}
	if size < int64(len(f.ino.volatile)) {
		f.ino.volatile = f.ino.volatile[:size:size]
	} else if size > int64(len(f.ino.volatile)) {
		f.ino.volatile = append(f.ino.volatile, make([]byte, size-int64(len(f.ino.volatile)))...)
	}
	f.ino.sizeDirt = true
	f.ino.mtime = f.s.now()
	return nil
}

func (f *simFile) Stat() (fs.FileInfo, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	if err := f.check(OpLstat); err != nil {
		return nil, err
	}
	return entry{file: f.ino}.info(path.Base(f.name)), nil
}

func (f *simFile) Chmod(mode fs.FileMode) error {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	if f.closed || f.dead {
		return ErrDead
	}
	f.ino.mode = mode
	return nil
}

func (f *simFile) TryLock() error {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	if f.closed || f.dead {
		return ErrDead
	}
	if holder, ok := f.s.locks[f.ino]; ok && holder != f {
		return ErrLocked
	}
	f.s.locks[f.ino] = f
	return nil
}

func (f *simFile) Close() error {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	if f.closed {
		return &fs.PathError{Op: "close", Path: f.name, Err: fs.ErrClosed}
	}
	f.closed = true
	if f.s.locks[f.ino] == f {
		delete(f.s.locks, f.ino)
	}
	return nil
}

func (f *simFile) Name() string { return f.name }

// fileInfo is the simulator's fs.FileInfo. Sys returns the file identity, which a test can
// use to tell a replaced file from the original.
type fileInfo struct {
	name  string
	size  int64
	mode  fs.FileMode
	mtime time.Time
	id    uint64
}

func (i fileInfo) Name() string       { return i.name }
func (i fileInfo) Size() int64        { return i.size }
func (i fileInfo) Mode() fs.FileMode  { return i.mode }
func (i fileInfo) ModTime() time.Time { return i.mtime }
func (i fileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i fileInfo) Sys() any           { return i.id }

// Durable returns the durable image of a file, or nil with an error if no durable directory
// entry reaches it. Tests use it to assert what survives without crashing.
func (s *Sim) Durable(name string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = path.Clean("/" + name)
	dir, base := path.Split(name)
	d := s.root
	for _, part := range splitPath(dir) {
		e, ok := d.durable[part]
		if !ok || e.dir == nil {
			return nil, &fs.PathError{Op: "durable", Path: name, Err: fs.ErrNotExist}
		}
		d = e.dir
	}
	e, ok := d.durable[base]
	if !ok || e.file == nil {
		return nil, &fs.PathError{Op: "durable", Path: name, Err: fs.ErrNotExist}
	}
	return slices.Clone(e.file.durable), nil
}
