package fsx

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"sync"
	"time"
)

// Sim is an in-memory file system that models what a power loss does to unsynced state
// (decision 61). Every file has a durable image and a volatile image; every directory has a
// durable entry table and a list of unsynced entry operations (create, mkdir, rename, unlink).
//
//   - A write changes the volatile image and marks the 4 KiB pages it touched, and the size
//     if it changed, as unsynced. Sync makes the file's volatile image durable, but not the
//     directory entry that names it.
//   - A create, mkdir, rename or unlink is an unsynced operation on its directory until SyncDir.
//     A rename over an existing name therefore yields the old or the new content until synced;
//     an unlink or a truncate may be undone.
//   - Crash decides, for each unsynced page, size and directory operation independently,
//     whether it persisted; a lost page of an extended file reads back as zeros. Persisted
//     directory operations are applied in the order they were issued. After a crash every
//     open File is dead and every volatile image is replaced by the durable one.
//
// Inject, when set, is consulted before every operation and may fail it: with ErrCrash the
// simulator crashes instead (as if power was lost just before the operation, using CrashKeep),
// and with a *ShortWrite a WriteAt writes that many bytes and returns io.ErrShortWrite.
//
// Paths are cleaned lexically; the root "/" always exists. Symbolic links do not exist in
// the simulator, so symlink behaviour is tested against OS.
type Sim struct {
	mu         sync.Mutex
	root       *simDir
	locks      map[*inode]*simFile
	ops        int    // operations performed, for the crash-at-every-operation driver
	generation uint64 // bumped by every crash; a File from an older generation is dead

	// Inject, if non-nil, is called before every operation with its kind and path and may
	// return an error to inject. It is called with the simulator's lock held: it must not
	// call back into the Sim.
	Inject func(op Op, name string) error
	// CrashKeep chooses, when Inject returns ErrCrash, which unsynced items persist: it is
	// called with the item index and the number of items. Nil keeps none.
	CrashKeep func(i, n int) bool
	// Clock stamps modification times; nil uses time.Now.
	Clock func() time.Time
}

// Op names an operation for Inject.
type Op string

const (
	OpOpen     Op = "open"
	OpCreate   Op = "create"
	OpRead     Op = "read"
	OpWrite    Op = "write"
	OpSync     Op = "sync"
	OpTruncate Op = "truncate"
	OpMkdir    Op = "mkdir"
	OpReadDir  Op = "readdir"
	OpLstat    Op = "lstat"
	OpRename   Op = "rename"
	OpRemove   Op = "remove"
	OpSyncDir  Op = "syncdir"
)

// ErrCrash, returned by Inject, makes the simulator crash before the operation.
var ErrCrash = errors.New("fsx: simulated crash")

// ErrDead is returned by every method of a File opened before a crash.
var ErrDead = errors.New("fsx: file handle from before a crash")

// ShortWrite, returned by Inject for OpWrite, makes the write persist N bytes and fail.
type ShortWrite struct{ N int }

func (s *ShortWrite) Error() string { return fmt.Sprintf("fsx: injected short write of %d bytes", s.N) }

const pageSize = 4096

type inode struct {
	durable  []byte
	volatile []byte
	dirty    map[int]bool // unsynced pages of the volatile image
	sizeDirt bool         // the size changed since the last Sync
	mode     fs.FileMode
	mtime    time.Time
	id       uint64 // identity, so a test can tell one file from another
}

type dirOp struct {
	kind    Op // OpCreate, OpMkdir, OpRename, OpRemove
	name    string
	oldname string
	file    *inode
	dir     *simDir
}

type entry struct {
	file *inode
	dir  *simDir
}

type simDir struct {
	durable  map[string]entry
	volatile map[string]entry
	ops      []dirOp
	mode     fs.FileMode
}

func newDir(mode fs.FileMode) *simDir {
	return &simDir{durable: map[string]entry{}, volatile: map[string]entry{}, mode: mode}
}

// NewSim returns an empty simulator with the root directory.
func NewSim() *Sim {
	return &Sim{root: newDir(0o700), locks: map[*inode]*simFile{}}
}

var _ FS = (*Sim)(nil)

func (s *Sim) now() time.Time {
	if s.Clock != nil {
		return s.Clock()
	}
	return time.Now()
}

// inject runs the Inject hook, crashing if it asks to. Called with the lock held.
func (s *Sim) inject(op Op, name string) error {
	s.ops++
	if s.Inject == nil {
		return nil
	}
	err := s.Inject(op, name)
	if errors.Is(err, ErrCrash) {
		s.crashLocked(s.CrashKeep)
	}
	return err
}

// Ops is the number of operations performed so far (including failed and injected ones).
func (s *Sim) Ops() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ops
}

// walk resolves the directory holding name and the base name, in the volatile namespace.
func (s *Sim) walk(name string) (*simDir, string, error) {
	name = path.Clean("/" + name)
	if name == "/" {
		return s.root, "", nil
	}
	dir, base := path.Split(name)
	d := s.root
	for _, part := range splitPath(dir) {
		e, ok := d.volatile[part]
		if !ok {
			return nil, "", &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
		if e.dir == nil {
			return nil, "", &fs.PathError{Op: "open", Path: name, Err: errNotDir}
		}
		d = e.dir
	}
	return d, base, nil
}

var errNotDir = errors.New("not a directory")

func splitPath(p string) []string {
	var parts []string
	for _, part := range splitOnSlash(path.Clean("/" + p)) {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func splitOnSlash(p string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(p); i++ {
		if i == len(p) || p[i] == '/' {
			out = append(out, p[start:i])
			start = i + 1
		}
	}
	return out
}

var nextID uint64

func (s *Sim) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	op := OpOpen
	if flag&O_CREATE != 0 {
		op = OpCreate
	}
	if err := s.inject(op, name); err != nil {
		return nil, err
	}
	d, base, err := s.walk(name)
	if err != nil {
		return nil, err
	}
	if base == "" {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	e, ok := d.volatile[base]
	switch {
	case ok && e.dir != nil:
		return nil, &fs.PathError{Op: "open", Path: name, Err: errIsDir}
	case ok && flag&(O_CREATE|O_EXCL) == O_CREATE|O_EXCL:
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrExist}
	case !ok && flag&O_CREATE == 0:
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	case !ok:
		nextID++
		e = entry{file: &inode{dirty: map[int]bool{}, mode: perm, mtime: s.now(), id: nextID}}
		d.volatile[base] = e
		d.ops = append(d.ops, dirOp{kind: OpCreate, name: base, file: e.file})
	}
	return &simFile{s: s, name: path.Clean("/" + name), ino: e.file, writable: flag&(O_WRONLY|O_RDWR) != 0, gen: s.generation}, nil
}

var errIsDir = errors.New("is a directory")

func (s *Sim) Mkdir(name string, perm fs.FileMode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.inject(OpMkdir, name); err != nil {
		return err
	}
	d, base, err := s.walk(name)
	if err != nil {
		return err
	}
	if base == "" {
		return &fs.PathError{Op: "mkdir", Path: name, Err: fs.ErrExist}
	}
	if _, ok := d.volatile[base]; ok {
		return &fs.PathError{Op: "mkdir", Path: name, Err: fs.ErrExist}
	}
	nd := newDir(perm)
	d.volatile[base] = entry{dir: nd}
	d.ops = append(d.ops, dirOp{kind: OpMkdir, name: base, dir: nd})
	return nil
}

func (s *Sim) ReadDir(name string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.inject(OpReadDir, name); err != nil {
		return nil, err
	}
	d, base, err := s.walk(name)
	if err != nil {
		return nil, err
	}
	if base != "" {
		e, ok := d.volatile[base]
		if !ok {
			return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
		}
		if e.dir == nil {
			return nil, &fs.PathError{Op: "readdir", Path: name, Err: errNotDir}
		}
		d = e.dir
	}
	names := make([]string, 0, len(d.volatile))
	for n := range d.volatile {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

func (s *Sim) Lstat(name string) (fs.FileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.inject(OpLstat, name); err != nil {
		return nil, err
	}
	d, base, err := s.walk(name)
	if err != nil {
		return nil, err
	}
	if base == "" {
		return fileInfo{name: "/", mode: fs.ModeDir | d.mode}, nil
	}
	e, ok := d.volatile[base]
	if !ok {
		return nil, &fs.PathError{Op: "lstat", Path: name, Err: fs.ErrNotExist}
	}
	return e.info(base), nil
}

func (e entry) info(name string) fileInfo {
	if e.dir != nil {
		return fileInfo{name: name, mode: fs.ModeDir | e.dir.mode}
	}
	return fileInfo{name: name, size: int64(len(e.file.volatile)), mode: e.file.mode, mtime: e.file.mtime, id: e.file.id}
}

func (s *Sim) Rename(oldpath, newpath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.inject(OpRename, oldpath); err != nil {
		return err
	}
	d, oldbase, err := s.walk(oldpath)
	if err != nil {
		return err
	}
	d2, newbase, err := s.walk(newpath)
	if err != nil {
		return err
	}
	if d != d2 || oldbase == "" || newbase == "" {
		return &fs.PathError{Op: "rename", Path: oldpath, Err: errors.New("fsx: rename across directories is not modelled")}
	}
	e, ok := d.volatile[oldbase]
	if !ok {
		return &fs.PathError{Op: "rename", Path: oldpath, Err: fs.ErrNotExist}
	}
	if e.dir != nil {
		return &fs.PathError{Op: "rename", Path: oldpath, Err: errIsDir}
	}
	if old, ok := d.volatile[newbase]; ok && old.dir != nil {
		return &fs.PathError{Op: "rename", Path: newpath, Err: errIsDir}
	}
	delete(d.volatile, oldbase)
	d.volatile[newbase] = e
	d.ops = append(d.ops, dirOp{kind: OpRename, name: newbase, oldname: oldbase, file: e.file})
	return nil
}

func (s *Sim) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.inject(OpRemove, name); err != nil {
		return err
	}
	d, base, err := s.walk(name)
	if err != nil {
		return err
	}
	e, ok := d.volatile[base]
	if !ok || base == "" {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrNotExist}
	}
	if e.dir != nil {
		return &fs.PathError{Op: "remove", Path: name, Err: errIsDir}
	}
	delete(d.volatile, base)
	d.ops = append(d.ops, dirOp{kind: OpRemove, name: base, file: e.file})
	return nil
}

func (s *Sim) SyncDir(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.inject(OpSyncDir, name); err != nil {
		return err
	}
	d, base, err := s.walk(name)
	if err != nil {
		return err
	}
	if base != "" {
		e, ok := d.volatile[base]
		if !ok {
			return &fs.PathError{Op: "syncdir", Path: name, Err: fs.ErrNotExist}
		}
		if e.dir == nil {
			return &fs.PathError{Op: "syncdir", Path: name, Err: errNotDir}
		}
		d = e.dir
	}
	d.durable = cloneEntries(d.volatile)
	d.ops = nil
	return nil
}

func cloneEntries(m map[string]entry) map[string]entry {
	out := make(map[string]entry, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Unsynced counts the items a crash would decide independently: unsynced pages and sizes
// of every file reachable in the volatile namespace, and unsynced directory operations.
func (s *Sim) Unsynced() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.unsyncedItems())
}

// item is one unsynced thing: a page or the size of a file, or a directory operation.
type item struct {
	ino  *inode
	page int // -1 for the size
	dir  *simDir
	op   int // index into dir.ops, when dir != nil
}

func (s *Sim) unsyncedItems() []item {
	var items []item
	seen := map[*inode]bool{}
	seenDir := map[*simDir]bool{}
	var walk func(d *simDir)
	walk = func(d *simDir) {
		if seenDir[d] {
			return
		}
		seenDir[d] = true
		for i := range d.ops {
			items = append(items, item{dir: d, op: i})
		}
		// Files reachable by any name, volatile or durable, or by an unsynced operation.
		var files []*inode
		var dirs []*simDir
		for _, e := range d.volatile {
			if e.file != nil {
				files = append(files, e.file)
			} else {
				dirs = append(dirs, e.dir)
			}
		}
		for _, e := range d.durable {
			if e.file != nil {
				files = append(files, e.file)
			} else {
				dirs = append(dirs, e.dir)
			}
		}
		for _, op := range d.ops {
			if op.file != nil {
				files = append(files, op.file)
			}
			if op.dir != nil {
				dirs = append(dirs, op.dir)
			}
		}
		for _, f := range files {
			if seen[f] {
				continue
			}
			seen[f] = true
			if f.sizeDirt {
				items = append(items, item{ino: f, page: -1})
			}
			pages := make([]int, 0, len(f.dirty))
			for p := range f.dirty {
				pages = append(pages, p)
			}
			slices.Sort(pages)
			for _, p := range pages {
				items = append(items, item{ino: f, page: p})
			}
		}
		for _, sub := range dirs {
			walk(sub)
		}
	}
	walk(s.root)
	return items
}

// Crash loses power: each unsynced item persists if keep(i, n) is true (nil keeps none),
// every open File dies, and the volatile state becomes the durable state. The items are
// in a fixed order for a given state, so a test can enumerate every subset.
func (s *Sim) Crash(keep func(i, n int) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.crashLocked(keep)
}

func (s *Sim) crashLocked(keep func(i, n int) bool) {
	items := s.unsyncedItems()
	n := len(items)
	kept := func(i int) bool { return keep != nil && keep(i, n) }

	// Files first: compute each inode's new durable image.
	type fileState struct {
		size    int64
		sizeSet bool
		pages   []int
	}
	files := map[*inode]*fileState{}
	dirs := map[*simDir][]bool{}
	for i, it := range items {
		if it.dir != nil {
			if dirs[it.dir] == nil {
				dirs[it.dir] = make([]bool, len(it.dir.ops))
			}
			dirs[it.dir][it.op] = kept(i)
			continue
		}
		st := files[it.ino]
		if st == nil {
			st = &fileState{}
			files[it.ino] = st
		}
		if !kept(i) {
			continue
		}
		if it.page < 0 {
			st.size, st.sizeSet = int64(len(it.ino.volatile)), true
		} else {
			st.pages = append(st.pages, it.page)
		}
	}
	for ino, st := range files {
		img := slices.Clone(ino.durable)
		if st.sizeSet {
			if int(st.size) < len(img) {
				img = img[:st.size]
			} else {
				img = append(img, make([]byte, int(st.size)-len(img))...)
			}
		}
		for _, p := range st.pages {
			lo, hi := p*pageSize, min((p+1)*pageSize, len(img), len(ino.volatile))
			if lo < hi {
				copy(img[lo:hi], ino.volatile[lo:hi])
			}
		}
		ino.durable = img
	}
	// Then directories: apply the persisted operations in order to the durable table.
	applied := map[*simDir]bool{}
	var apply func(d *simDir)
	apply = func(d *simDir) {
		if applied[d] {
			return
		}
		applied[d] = true
		keptOps := dirs[d]
		for i, op := range d.ops {
			if keptOps != nil && !keptOps[i] {
				continue
			}
			switch op.kind {
			case OpCreate:
				d.durable[op.name] = entry{file: op.file}
			case OpMkdir:
				d.durable[op.name] = entry{dir: op.dir}
			case OpRename:
				delete(d.durable, op.oldname)
				d.durable[op.name] = entry{file: op.file}
			case OpRemove:
				delete(d.durable, op.name)
			}
		}
		d.ops = nil
		for _, e := range d.volatile {
			if e.dir != nil {
				apply(e.dir)
			}
		}
		for _, e := range d.durable {
			if e.dir != nil {
				apply(e.dir)
			}
		}
	}
	apply(s.root)
	// Reboot: volatile = durable everywhere, nothing dirty, no locks, every handle dead.
	var reset func(d *simDir)
	reset = func(d *simDir) {
		d.volatile = cloneEntries(d.durable)
		for _, e := range d.volatile {
			if e.file != nil {
				e.file.volatile = slices.Clone(e.file.durable)
				e.file.dirty = map[int]bool{}
				e.file.sizeDirt = false
			} else {
				reset(e.dir)
			}
		}
	}
	reset(s.root)
	for _, f := range s.locks {
		f.dead = true
	}
	s.locks = map[*inode]*simFile{}
	s.generation++
}
