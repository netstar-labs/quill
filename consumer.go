package quill

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Protect says whether a consumer's unread segments are kept by retention (decision 57).
type Protect int

const (
	// ProtectAsStored keeps the flag the consumer's file holds; a new consumer is unprotected.
	ProtectAsStored Protect = iota
	// ProtectOn makes retention keep the consumer's unread segments, and records the flag.
	ProtectOn
	// ProtectOff stops retention keeping them, and records the flag.
	ProtectOff
)

// ConsumerOptions configures a consumer handle.
type ConsumerOptions struct {
	Protect Protect
	// Rewind opens a consumer whose offset is above Head + 1 (a log restored to an older state)
	// instead of failing with ErrAhead, so that Commit can rewind it. Such a handle reads
	// nothing and has no lag until it is rewound or the head catches up.
	Rewind bool
}

// Consumer is a named offset: the next sequence to read, stored in consumers/{name} and
// written only by the process holding consumers/{name}.lock (decisions 9, 36, 41).
type Consumer struct {
	store *consumerStore
	src   source
	name  string
	lock  fsx.File

	mu     sync.Mutex
	state  consumerState
	closed bool
}

// consumerStore is the consumers/ directory of one log, as seen by a Log or a Reader.
type consumerStore struct {
	fs   fsx.FS
	dir  string
	id   logID
	perm fs.FileMode
	// src is the Log or Reader the store belongs to; consumers read through it, and first
	// takes its view. head is the log's current head, for the offset range check.
	src  source
	head func() (uint64, error)
}

// first is the first retained sequence, where a new consumer starts.
func (st *consumerStore) first() (uint64, error) {
	v, release, err := st.src.view()
	if err != nil {
		return 0, err
	}
	release()
	return v.first, nil
}

func (st *consumerStore) path(name string) string {
	return filepath.Join(st.dir, consumersDir, name)
}

// open takes the consumer's lock, creates its file if absent at the first retained
// sequence, checks the log id and the offset, and applies the protect option.
func (st *consumerStore) open(name string, opts ConsumerOptions) (*Consumer, error) {
	if err := validConsumerName(name); err != nil {
		return nil, err
	}
	lock, err := takeLock(st.fs, st.path(name)+lockSuffix, st.perm)
	if err != nil {
		return nil, err
	}
	c := &Consumer{store: st, src: st.src, name: name, lock: lock}
	if err := c.init(opts); err != nil {
		_ = lock.Close()
		return nil, err
	}
	return c, nil
}

// protect makes the consumer exist, protected, for the writer's Open (decision 67). A file
// that is already protected, parses, is this log's and is not ahead of the head needs no
// write and so no lock: the consumer's process may be running and holds the lock for its
// whole life, and the writer must still be able to restart. Every other case (a missing
// file, an unprotected one, one that open would refuse) goes through open, which takes the
// lock and writes the flag or returns the error it always did (ErrCorrupt, ErrWrongLog,
// ErrAhead, or ErrLocked naming the consumer's lock).
func (st *consumerStore) protect(name string) error {
	if data, err := readState(st.fs, st.path(name)); err == nil {
		if state, err := decodeConsumer(data); err == nil && state.ID == st.id && state.Protect {
			head, err := st.head()
			if err != nil {
				return err
			}
			if state.Next <= head+1 {
				return nil
			}
		}
	}
	c, err := st.open(name, ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		return err
	}
	return c.Close()
}

// init loads or creates the consumer's file and applies the options. It leaves the lock to
// its caller, which releases it on any error.
func (c *Consumer) init(opts ConsumerOptions) error {
	st, name := c.store, c.name
	data, err := readState(st.fs, st.path(name))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		first, err := st.first()
		if err != nil {
			return err
		}
		c.state = consumerState{ID: st.id, Next: first, Protect: opts.Protect == ProtectOn}
		return c.write(true)
	case err != nil:
		return err
	}
	state, err := decodeConsumer(data)
	if err != nil {
		return fmt.Errorf("%w: consumer %q: %v", ErrCorrupt, name, err)
	}
	if state.ID != st.id {
		return fmt.Errorf("%w: consumer %q is from log %s, this is %s", ErrWrongLog, name, state.ID, st.id)
	}
	head, err := st.head()
	if err != nil {
		return err
	}
	if state.Next > head+1 && !opts.Rewind {
		return fmt.Errorf("%w: consumer %q is at %d, the head is %d", ErrAhead, name, state.Next, head)
	}
	c.state = state
	want := state.Protect
	switch opts.Protect {
	case ProtectOn:
		want = true
	case ProtectOff:
		want = false
	}
	if want != state.Protect {
		c.state.Protect = want
		return c.write(false)
	}
	return nil
}

// write persists the state: temp, fsync, rename; syncDir when the file is new, so a
// protected consumer exists durably from its creation.
func (c *Consumer) write(syncDir bool) error {
	if err := writeState(c.store.fs, c.store.path(c.name), encodeConsumer(c.state), c.store.perm); err != nil {
		return err
	}
	if syncDir {
		return c.store.fs.SyncDir(filepath.Join(c.store.dir, consumersDir))
	}
	return nil
}

// Name returns the consumer's name.
func (c *Consumer) Name() string { return c.name }

// Offset returns the committed offset: the next sequence to read.
func (c *Consumer) Offset() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state.Next
}

// Protected reports whether retention keeps the consumer's unread segments.
func (c *Consumer) Protected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state.Protect
}

// Commit stores next as the offset: the idiom is Commit(lastRead + 1). It accepts values
// from the first retained sequence to Head + 1 and refuses others with ErrOffsetRange. Commit
// once per batch: each commit is a file write and fsync. It never changes the protect flag.
func (c *Consumer) Commit(next uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	head, err := c.store.head()
	if err != nil {
		return err
	}
	first, err := c.store.first()
	if err != nil {
		return err
	}
	if next < first || next > head+1 {
		return fmt.Errorf("%w: %d is outside [%d, %d]", ErrOffsetRange, next, first, head+1)
	}
	prev := c.state.Next
	c.state.Next = next
	if err := c.write(false); err != nil {
		c.state.Next = prev
		return err
	}
	return nil
}

// Lag returns Head + 1 − offset: the records not yet read.
func (c *Consumer) Lag() (uint64, error) {
	head, err := c.store.head()
	if err != nil {
		return 0, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Next > head+1 {
		return 0, nil
	}
	return head + 1 - c.state.Next, nil
}

// Read reads from the consumer's offset, without committing; ReadWait and Scan follow.
func (c *Consumer) Read(limit int) ([]Record, error) { return readFrom(c.src, c.Offset(), limit) }

// ReadWait is Read, blocking until a record is available, ctx is done or the handle closes.
func (c *Consumer) ReadWait(ctx context.Context, limit int) ([]Record, error) {
	return readWait(ctx, c.src, c.Offset(), limit)
}

// Scan is the zero-copy Read from the consumer's offset, as Log.Scan, without committing.
func (c *Consumer) Scan(limit int, fn func(Record) bool) error {
	return scanFrom(c.src, c.Offset(), limit, fn)
}

// Close releases the consumer's lock. The offset file stays.
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	c.closed = true
	return c.lock.Close()
}

// ConsumerInfo describes one consumer file, as listed by Consumers.
type ConsumerInfo struct {
	Name      string
	Offset    uint64
	Protected bool
	Err       error // the file could not be read or parsed; such a consumer protects everything
}

// list reads every consumer file. A lock file or a temp file is not a consumer.
func (st *consumerStore) list() ([]ConsumerInfo, error) {
	names, err := st.fs.ReadDir(filepath.Join(st.dir, consumersDir))
	if err != nil {
		return nil, err
	}
	var out []ConsumerInfo
	for _, n := range names {
		if strings.HasSuffix(n, lockSuffix) || strings.HasPrefix(n, tempPrefix) {
			continue
		}
		info := ConsumerInfo{Name: n}
		data, err := readState(st.fs, st.path(n))
		if err == nil {
			var state consumerState
			if state, err = decodeConsumer(data); err == nil {
				if state.ID != st.id {
					err = fmt.Errorf("%w: %s", ErrWrongLog, state.ID)
				}
				info.Offset, info.Protected = state.Next, state.Protect
			}
		}
		info.Err = err
		out = append(out, info)
	}
	return out, nil
}

// remove takes the consumer's lock and deletes its offset file; the lock file stays.
func (st *consumerStore) remove(name string) error {
	if err := validConsumerName(name); err != nil {
		return err
	}
	lock, err := takeLock(st.fs, st.path(name)+lockSuffix, st.perm)
	if err != nil {
		return err
	}
	defer lock.Close()
	return st.fs.Remove(st.path(name))
}

// consumerMtime returns a consumer file's modification time, for ConsumerTTL.
func (st *consumerStore) mtime(name string) (time.Time, error) {
	info, err := st.fs.Lstat(st.path(name))
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

const lockSuffix = ".lock"

// The Log's consumer API.

func (l *Log) consumerStore() *consumerStore {
	return &consumerStore{fs: l.fs, dir: l.dir, id: l.id, perm: l.cfg.Perm, src: l,
		head: func() (uint64, error) { return l.Head(), nil }}
}

// Consumer takes a handle on a named consumer, creating its offset file at the first retained
// sequence if absent. It holds the consumer's lock until Close.
func (l *Log) Consumer(name string, opts ConsumerOptions) (*Consumer, error) {
	return l.consumerStore().open(name, opts)
}

// Consumers lists every consumer file.
func (l *Log) Consumers() ([]ConsumerInfo, error) { return l.consumerStore().list() }

// DeleteConsumer takes the consumer's lock and removes its offset file; the lock file is
// never deleted.
func (l *Log) DeleteConsumer(name string) error { return l.consumerStore().remove(name) }

// The Reader's consumer API.

func (r *Reader) consumerStore() *consumerStore {
	return &consumerStore{fs: r.fs, dir: r.dir, id: r.id, perm: 0o600, src: r, head: r.Head}
}

// Consumer takes a handle on a named consumer through a Reader: an external consumer.
func (r *Reader) Consumer(name string, opts ConsumerOptions) (*Consumer, error) {
	return r.consumerStore().open(name, opts)
}

// Consumers lists every consumer file.
func (r *Reader) Consumers() ([]ConsumerInfo, error) { return r.consumerStore().list() }

// DeleteConsumer takes the consumer's lock and removes its offset file.
func (r *Reader) DeleteConsumer(name string) error { return r.consumerStore().remove(name) }
