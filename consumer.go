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
	ProtectOn
	ProtectOff
)

// ConsumerOptions configures a consumer handle.
type ConsumerOptions struct {
	Protect Protect
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
	// head and first are the log's current bounds, for Commit's range check.
	head  func() (uint64, error)
	first func() (uint64, error)
}

func (st *consumerStore) path(name string) string {
	return filepath.Join(st.dir, consumersDir, name)
}

// open takes the consumer's lock, creates its file if absent at the first retained
// sequence, checks the log id and the offset, and applies the protect option. internal
// allows the reserved leading underscore (the archiver).
func (st *consumerStore) open(src source, name string, opts ConsumerOptions, internal bool) (*Consumer, error) {
	if internal {
		if err := validName(strings.TrimPrefix(name, "_")); err != nil || !strings.HasPrefix(name, "_") {
			return nil, fmt.Errorf("%w: %q", ErrInvalidName, name)
		}
	} else if err := validName(name); err != nil {
		return nil, err
	}
	lock, err := takeLock(st.fs, st.path(name)+lockSuffix, st.perm)
	if err != nil {
		return nil, err
	}
	c := &Consumer{store: st, src: src, name: name, lock: lock}
	data, err := readState(st.fs, st.path(name))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		first, ferr := st.first()
		if ferr != nil {
			_ = lock.Close()
			return nil, ferr
		}
		c.state = consumerState{ID: st.id, Next: first, Protect: opts.Protect == ProtectOn}
		if err := c.write(true); err != nil {
			_ = lock.Close()
			return nil, err
		}
		return c, nil
	case err != nil:
		_ = lock.Close()
		return nil, err
	}
	state, err := decodeConsumer(data)
	if err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("%w: consumer %q: %v", ErrCorrupt, name, err)
	}
	if state.ID != st.id {
		_ = lock.Close()
		return nil, fmt.Errorf("%w: consumer %q is from log %s, this is %s", ErrWrongLog, name, state.ID, st.id)
	}
	head, err := st.head()
	if err != nil {
		_ = lock.Close()
		return nil, err
	}
	if state.Next > head+1 {
		_ = lock.Close()
		return nil, fmt.Errorf("%w: consumer %q is at %d, the head is %d", ErrAhead, name, state.Next, head)
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
		if err := c.write(false); err != nil {
			_ = lock.Close()
			return nil, err
		}
	}
	return c, nil
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

// Read, ReadWait and Scan read from the consumer's offset, without committing.
func (c *Consumer) Read(limit int) ([]Record, error) { return readFrom(c.src, c.Offset(), limit) }
func (c *Consumer) ReadWait(ctx context.Context, limit int) ([]Record, error) {
	return readWait(ctx, c.src, c.Offset(), limit)
}
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
	if err := validName(name); err != nil {
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
	return &consumerStore{fs: l.fs, dir: l.dir, id: l.id, perm: l.cfg.Perm,
		head: func() (uint64, error) { return l.Head(), nil },
		first: func() (uint64, error) {
			v, release, err := l.view()
			if err != nil {
				return 0, err
			}
			release()
			return v.first, nil
		}}
}

// Consumer takes a handle on a named consumer, creating its offset file at the first retained
// sequence if absent. It holds the consumer's lock until Close.
func (l *Log) Consumer(name string, opts ConsumerOptions) (*Consumer, error) {
	return l.consumerStore().open(l, name, opts, false)
}

// Consumers lists every consumer file.
func (l *Log) Consumers() ([]ConsumerInfo, error) { return l.consumerStore().list() }

// DeleteConsumer takes the consumer's lock and removes its offset file; the lock file is
// never deleted.
func (l *Log) DeleteConsumer(name string) error { return l.consumerStore().remove(name) }

// The Reader's consumer API.

func (r *Reader) consumerStore() *consumerStore {
	return &consumerStore{fs: r.fs, dir: r.dir, id: r.id, perm: 0o600,
		head: r.Head,
		first: func() (uint64, error) {
			v, release, err := r.view()
			if err != nil {
				return 0, err
			}
			release()
			return v.first, nil
		}}
}

// Consumer takes a handle on a named consumer through a Reader: an external consumer.
func (r *Reader) Consumer(name string, opts ConsumerOptions) (*Consumer, error) {
	return r.consumerStore().open(r, name, opts, false)
}

// Consumers lists every consumer file.
func (r *Reader) Consumers() ([]ConsumerInfo, error) { return r.consumerStore().list() }

// DeleteConsumer takes the consumer's lock and removes its offset file.
func (r *Reader) DeleteConsumer(name string) error { return r.consumerStore().remove(name) }
