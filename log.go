package quill

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Config configures a Log. The zero value of every field but Dir is the default below.
type Config struct {
	// Dir is the log's directory, created if missing. Its base name is the topic name.
	Dir string
	// Retention: whole sealed segments older than this are deleted. 0 means 24 h; negative
	// disables expiry. Set it longer than the worst delay of any unprotected consumer.
	Retention time.Duration
	// PruneInterval is the sweep period: min(Retention, 1 m), at least 1 s; one minute when
	// Retention is negative.
	PruneInterval time.Duration
	// FlushBytes is the buffer size at which an append flushes inline (64 KiB).
	FlushBytes int
	// FlushInterval is the background flush period (250 ms).
	FlushInterval time.Duration
	// SyncAppends makes every Append and AppendBatch return only once its records are durable,
	// as if it called Sync on its last sequence. An error then means unknown, not lost.
	SyncAppends bool
	// MaxBufferBytes applies only while flushes are failing: then an append is accepted only
	// into an empty buffer or within this limit (16 × FlushBytes).
	MaxBufferBytes int
	// SegmentBytes is the rotation size (8 MiB), at most 1 GiB, and under the 1.5 GiB segment file
	// limit once the largest flush is added: max(FlushBytes, MaxBufferBytes) + max(MaxBatchBytes,
	// MaxPayloadBytes+34). While rotation keeps failing, an append that would pass that sum is
	// refused with the rotation error.
	SegmentBytes int64
	// SegmentMaxAge rotates a quiet active segment so retention can delete it (Retention).
	// Left at zero it takes Retention's value, so it is never when Retention is negative; an
	// explicit positive value still rotates, whatever Retention is, and a negative one is never.
	SegmentMaxAge time.Duration
	// MaxBytes caps this log's segment bytes (0: off). Below twice SegmentBytes is refused.
	MaxBytes int64
	// MinFreeBytes is a free-space floor on the log's filesystem (0: off).
	MinFreeBytes int64
	// MaxPayloadBytes is the largest accepted record (64 MiB − 32, the ceiling).
	MaxPayloadBytes int
	// MaxBatchBytes is the largest encoded AppendBatch (16 MiB).
	MaxBatchBytes int
	// MaxReadBytes is the payload bytes one Read returns, at least one record (16 MiB).
	MaxReadBytes int
	// ProtectedConsumers names consumers the writer creates, protected, when it opens the log. A
	// file already protected is left alone, so a running consumer does not block a restart.
	ProtectedConsumers []string
	// Logger receives retention deletions, stale consumers, bulk-expiry warnings and failures.
	// Its handler, Enabled included, runs synchronously with the log's lock held, on the
	// goroutine that made the record (an Append, Flush, Roll, Prune or Close call, or the
	// log's flusher). It must therefore neither call the Log, which deadlocks it, nor block,
	// which stalls every Append, Read, Sync and the flusher: hand the record to another
	// goroutine and return.
	Logger *slog.Logger
	// Perm is the file mode (0600); directories get the matching execute bits (0700).
	Perm fs.FileMode
	// MaxRecords keeps roughly the newest N records (0: off).
	MaxRecords int64
	// ConsumerTTL: a protected consumer behind for this long stops protecting segments (0: off).
	ConsumerTTL time.Duration
	// NoMaintenance disables the background prune pass and age rotation: for offline tools
	// that open a log briefly (decision 68). Flushing still runs. Open does not write
	// quill.policy under it, so a tool never replaces the writer's settings (decision 99).
	NoMaintenance bool

	fs     fsx.FS                           // the seam; nil is the real file system
	clock  func() time.Time                 // nil is time.Now
	statfs func(dir string) (uint64, error) // nil is freeBytes
}

// Defaults and ceilings.
const (
	DefaultRetention     = 24 * time.Hour
	DefaultFlushBytes    = 64 << 10
	DefaultFlushInterval = 250 * time.Millisecond
	DefaultSegmentBytes  = 8 << 20
	MaxSegmentBytes      = 1 << 30
	DefaultMaxBatchBytes = 16 << 20
	DefaultMaxReadBytes  = 16 << 20
	// MaxPayloadBytes is the default and the ceiling of Config.MaxPayloadBytes.
	MaxPayloadBytes = maxPayloadBytes
	DefaultPerm     = fs.FileMode(0o600)
)

// withDefaults returns the config with every zero field set to its default.
func (c Config) withDefaults() Config {
	if c.Retention == 0 {
		c.Retention = DefaultRetention
	}
	if c.PruneInterval == 0 {
		if c.Retention < 0 {
			c.PruneInterval = time.Minute
		} else {
			c.PruneInterval = max(min(c.Retention, time.Minute), time.Second)
		}
	}
	if c.FlushBytes == 0 {
		c.FlushBytes = DefaultFlushBytes
	}
	if c.FlushInterval == 0 {
		c.FlushInterval = DefaultFlushInterval
	}
	if c.MaxBufferBytes == 0 {
		c.MaxBufferBytes = 16 * c.FlushBytes
	}
	if c.SegmentBytes == 0 {
		c.SegmentBytes = DefaultSegmentBytes
	}
	if c.SegmentMaxAge == 0 {
		c.SegmentMaxAge = c.Retention
	}
	if c.MaxPayloadBytes == 0 {
		c.MaxPayloadBytes = MaxPayloadBytes
	}
	if c.MaxBatchBytes == 0 {
		c.MaxBatchBytes = DefaultMaxBatchBytes
	}
	if c.MaxReadBytes == 0 {
		c.MaxReadBytes = DefaultMaxReadBytes
	}
	if c.Perm == 0 {
		c.Perm = DefaultPerm
	}
	if c.fs == nil {
		c.fs = fsx.OS{}
	}
	if c.clock == nil {
		c.clock = time.Now
	}
	if c.statfs == nil {
		c.statfs = freeBytes
	}
	return c
}

// validate checks a config with defaults applied.
func (c Config) validate() error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrConfig, fmt.Sprintf(format, args...))
	}
	switch {
	case c.Dir == "":
		return bad("Dir is required")
	case c.PruneInterval < time.Second:
		return bad("PruneInterval %v below 1s", c.PruneInterval)
	case c.FlushBytes < 0, c.FlushInterval < 0, c.MaxBufferBytes < 0, c.SegmentBytes < 0,
		c.MaxBytes < 0, c.MinFreeBytes < 0, c.MaxPayloadBytes < 0,
		c.MaxBatchBytes < 0, c.MaxReadBytes < 0, c.MaxRecords < 0, c.ConsumerTTL < 0:
		return bad("a size, count or duration is negative") // Retention and SegmentMaxAge may be: negative means never
	case c.FlushInterval < time.Millisecond:
		return bad("FlushInterval %v below 1ms", c.FlushInterval)
	case c.SegmentBytes > MaxSegmentBytes:
		return bad("SegmentBytes %d above %d", c.SegmentBytes, MaxSegmentBytes)
	case c.MaxPayloadBytes > MaxPayloadBytes:
		return bad("MaxPayloadBytes %d above the ceiling %d", c.MaxPayloadBytes, MaxPayloadBytes)
	case c.maxActiveBytes() >= maxSegmentBytes:
		return bad("SegmentBytes %d plus one flush (up to %d bytes) reaches the %d-byte segment file limit",
			c.SegmentBytes, c.maxActiveBytes()-c.SegmentBytes, int64(maxSegmentBytes))
	case c.MaxBytes != 0 && c.MaxBytes < 2*c.SegmentBytes:
		return bad("MaxBytes %d below twice SegmentBytes %d", c.MaxBytes, c.SegmentBytes)
	case c.Perm&0o600 != 0o600:
		return bad("Perm %v must include owner read and write", c.Perm)
	case c.Perm&^fs.ModePerm != 0:
		return bad("Perm %v holds mode bits other than permissions", c.Perm)
	}
	for _, name := range c.ProtectedConsumers {
		if err := validName(name); err != nil {
			return bad("ProtectedConsumers: %v", err)
		}
	}
	return nil
}

// maxActiveBytes is the most the active segment can hold: SegmentBytes, which a flush reaches
// and then rotates, plus one flush: the buffer (at most the larger of FlushBytes and
// MaxBufferBytes) and the one append that overshoots it (at most MaxBatchBytes, or a single
// record's MaxPayloadBytes plus 34; the accounting of decision 69). The ceiling an append is
// refused at while rotation keeps failing, and what validate keeps under the file limit.
func (c Config) maxActiveBytes() int64 {
	return c.SegmentBytes + int64(max(c.FlushBytes, c.MaxBufferBytes)+max(c.MaxBatchBytes, c.MaxPayloadBytes+34))
}

// dirPerm is the directory mode for a file mode: execute wherever read is granted.
func (c Config) dirPerm() fs.FileMode {
	return c.Perm | (c.Perm&0o444)>>2
}

// validName checks a consumer name for the public API: [a-z0-9_-], 1 to 200 characters, no
// leading underscore (reserved).
func validName(name string) error {
	if name == "" || len(name) > 200 {
		return fmt.Errorf("%w: %q: 1 to 200 characters", ErrInvalidName, name)
	}
	if name[0] == '_' {
		return fmt.Errorf("%w: %q: a leading underscore is reserved", ErrInvalidName, name)
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return fmt.Errorf("%w: %q: only [a-z0-9_-]", ErrInvalidName, name)
		}
	}
	return nil
}

// Log is one topic in one directory, held by its one writer.
type Log struct {
	cfg Config
	fs  fsx.FS
	dir string
	// tailPath is the marker's path, joined once in Open: publishLocked writes it on every flush.
	tailPath string
	id       logID
	lock     fsx.File

	mu sync.Mutex // the log's lock: everything below
	// The buffer of encoded frames not yet written, and the sequences it holds.
	buf      []byte
	bufFirst uint64 // the first sequence in the buffer, when bufCount > 0
	bufCount int
	nextSeq  uint64 // the next sequence to assign
	head     uint64 // the last durable sequence
	maxSeg   uint64 // the highest segment id ever created
	active   *activeSegment
	sealed   []*segment // ascending by id

	// sealedRefs is the good part of sealed as segRefs, valid while refsValid; every writer of
	// sealed clears refsValid (decision 103). Guarded by mu.
	sealedRefs []*segRef
	refsValid  bool

	holes    []Hole
	marker   tail   // the last published quill.tail
	tailBuf  []byte // scratch for encoding the marker; used only under mu
	failed   error  // sticky: an fsync error wrapping ErrFailed
	bgErr    error  // the last background or retryable error, for Err()
	flushErr error  // the last flush's error, while flushes are failing; nil once one succeeds
	loadErr  error  // a sticky load error (a sealed segment that failed), for Err()
	closed   bool
	// justSealed is set by a rotation until the marker naming the new segment has been
	// published and fsynced into the directory (decision 65).
	justSealed bool
	// pendingPublish is set when the data is durable but the marker could not be published
	// (or the next segment created); the next flush retries (decision 60 d).
	pendingPublish bool
	// rotErr is the error of the last rotation due at a flush, while it keeps failing; nil once one
	// succeeds. While set, an append is refused at the ceiling (decision 60 d).
	rotErr error
	// flushed is made by the first waiter (waitChanLocked) and closed and set to nil by the next
	// flush attempt; waiters block on it without the lock (Sync, ReadWait).
	flushed  chan struct{}
	flushGen uint64

	markerAt time.Time // when the marker was last written
	dropped  uint64    // segments deleted while some consumer's offset was below them
	stale    []string  // protected consumers found stale at the last prune pass
	diskFull bool      // MinFreeBytes cannot be met: appends fail with ENOSPC

	flushReq chan struct{} // signals the flusher
	stop     chan struct{}
	done     chan struct{}
	fsyncs   int // fsyncs of segment data, for tests

	bufPool sync.Pool
}

// Hole is a run of sequences missing from the middle of a log: a damaged or deleted segment.
type Hole struct {
	From, To uint64
}

// segment is a sealed segment: immutable, so its range and index are fixed.
type segment struct {
	id     uint64
	path   string
	first  uint64
	last   uint64
	size   int64
	index  []anchor
	at     int64 // the first frame's At
	lastAt int64
	bad    error // a *SegmentError when the segment failed to load; its range is a hole
}

// activeSegment is the newest segment, the one the writer appends to.
type activeSegment struct {
	id         uint64
	path       string
	f          fsx.File
	durable    int64 // bytes fsynced: quill.tail.off
	first      uint64
	frames     int
	index      []anchor
	at         int64
	lastAt     int64
	nextAnchor int64 // the byte position at or after which the next anchor is taken
	readers    int   // reads in progress on f (so rotation and Close do not close it under them)
	retired    bool  // rotated out or the log closed; closed when readers reaches 0
}

// closeIfIdle closes the descriptor, once, when the segment is retired and no read is in
// flight (decisions 43 and 95), and reports the close error. It is the one home of that rule:
// rotation, Close and the last read's release all call it. Call it with l.mu held.
func (a *activeSegment) closeIfIdle() error {
	if !a.retired || a.readers > 0 || a.f == nil {
		return nil
	}
	err := a.f.Close()
	a.f = nil
	return err
}

func (l *Log) path(name string) string { return filepath.Join(l.dir, name) }

// Open opens a log for writing, creating it if the directory is new, and takes the
// directory's lock: this process is the log's one assigned writer (decision 62).
func Open(cfg Config) (*Log, error) {
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	l := &Log{cfg: cfg, fs: cfg.fs, dir: filepath.Clean(cfg.Dir),
		flushReq: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	l.tailPath = l.path(tailName)
	l.bufPool.New = func() any { b := make([]byte, 0, windowChunk); return &b }
	if err := l.mkdirAll(l.dir); err != nil {
		return nil, err
	}
	lock, err := takeLock(l.fs, l.path(lockName), cfg.Perm)
	if err != nil {
		return nil, err
	}
	l.lock = lock
	if err := l.create(); err != nil {
		_ = lock.Close()
		return nil, err
	}
	if err := l.load(); err != nil {
		if l.active != nil && l.active.f != nil {
			_ = l.active.f.Close()
		}
		_ = lock.Close()
		return nil, err
	}
	l.removeStaleTemps()
	if !cfg.NoMaintenance { // an offline tool never replaces the writer's policy (decision 99)
		if err := l.writePolicy(); err != nil {
			l.bgErr = err
		}
	}
	// Protected consumers exist, protected, before any consumer process starts (decision 67);
	// one already protected, even if its process is running, is left alone.
	for _, name := range cfg.ProtectedConsumers {
		if err := l.consumerStore().protect(name); err != nil {
			_ = l.active.f.Close()
			_ = lock.Close()
			return nil, fmt.Errorf("protected consumer %q: %w", name, err)
		}
	}
	go l.flusher()
	return l, nil
}

// mkdirAll creates dir and any missing parents, fsyncing each into its parent.
func (l *Log) mkdirAll(dir string) error {
	if info, err := l.fs.Lstat(dir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%w: %s is not a directory", ErrNoLog, dir)
		}
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(dir)
	if parent != dir {
		if err := l.mkdirAll(parent); err != nil {
			return err
		}
	}
	if err := l.fs.Mkdir(dir, l.cfg.dirPerm()); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	return l.fs.SyncDir(parent)
}

// create decides whether the directory is a new log and, if so, creates it; it also finishes
// an interrupted creation: whichever of consumers/, the first segment and quill.tail is
// missing is created, when the log holds no segments and no non-zero marker. The lock is held.
func (l *Log) create() error {
	names, err := l.fs.ReadDir(l.dir)
	if err != nil {
		return err
	}
	hasFormat, hasTail, hasSegment, hasConsumers, other := false, false, false, false, false
	for _, n := range names {
		switch {
		case n == lockName, strings.HasPrefix(n, tempPrefix):
		case n == formatName:
			hasFormat = true
		case n == tailName:
			hasTail = true
		case n == consumersDir:
			hasConsumers = true
		default:
			if _, ok := parseSegmentName(n); ok {
				hasSegment = true
			} else {
				other = true
			}
		}
	}
	switch {
	case !hasFormat && !other && !hasTail && !hasSegment && !hasConsumers:
		// New. The marker is made durable before anything else exists, so no crash can leave
		// a segment or consumers/ without it (decision 81).
		id, err := newLogID()
		if err != nil {
			return err
		}
		if err := writeState(l.fs, l.path(formatName), encodeFormat(id), l.cfg.Perm); err != nil {
			return err
		}
		if err := l.fs.SyncDir(l.dir); err != nil {
			return err
		}
	case !hasFormat:
		return fmt.Errorf("%w: %s holds files but no %s", ErrNoLog, l.dir, formatName)
	}
	changed := false
	if !hasConsumers {
		if err := l.fs.Mkdir(l.path(consumersDir), l.cfg.dirPerm()); err != nil {
			return err
		}
		changed = true
	}
	if !hasSegment {
		// No segment at all: an interrupted creation, unless the marker says records existed.
		if hasTail {
			data, err := readState(l.fs, l.path(tailName))
			if err != nil {
				return err
			}
			if tl, err := decodeTail(data); err != nil || tl != (tail{}) {
				return fmt.Errorf("%w: no segment files", ErrCorrupt)
			}
		}
		f, err := l.fs.OpenFile(l.path(segmentName(0)), fsx.O_RDWR|fsx.O_CREATE|fsx.O_EXCL, l.cfg.Perm)
		if err != nil {
			return err
		}
		if err := f.Sync(); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		changed = true
	}
	if !hasTail {
		// Without a marker the directory is an interrupted creation only if no segment holds
		// a byte (decision 62).
		for _, n := range names {
			if _, ok := parseSegmentName(n); !ok {
				continue
			}
			info, err := l.fs.Lstat(l.path(n))
			if err != nil {
				return err
			}
			if info.Size() > 0 {
				return fmt.Errorf("%w: segments with frames but no %s", ErrCorrupt, tailName)
			}
		}
		if err := writeState(l.fs, l.path(tailName), encodeTail(tail{}), l.cfg.Perm); err != nil {
			return err
		}
		changed = true
	}
	if changed {
		return l.fs.SyncDir(l.dir)
	}
	return nil
}

// Close flushes, publishes the marker, fsyncs the directory, releases the lock and returns
// the final flush's error. Pending Sync waiters get ErrClosed unless their sequence became
// durable.
func (l *Log) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return ErrClosed
	}
	l.closed = true
	close(l.stop)
	l.mu.Unlock()
	<-l.done

	l.mu.Lock()
	defer l.mu.Unlock()
	err := l.flushLocked()
	if err == nil && l.pendingPublish {
		err = l.publishLocked()
	}
	if serr := l.fs.SyncDir(l.dir); err == nil {
		err = serr
	}
	if a := l.active; a != nil && a.f != nil {
		// A reader that took its view before Close still uses a.f outside the lock: retire
		// the segment and let the last release close it, as rotation does (decision 95).
		a.retired = true
		if cerr := a.closeIfIdle(); err == nil {
			err = cerr
		}
	}
	if cerr := l.lock.Close(); err == nil {
		err = cerr
	}
	l.broadcastLocked()
	return err
}

// Head returns the last durable sequence, 0 when the log has none.
func (l *Log) Head() uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.head
}

// Err returns the sticky load error, the failed state, or the last background error.
func (l *Log) Err() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch {
	case l.failed != nil:
		return l.failed
	case l.loadErr != nil:
		return l.loadErr
	}
	return l.bgErr
}

// ID returns the log's identity, as recorded in quill.format and every consumer file.
func (l *Log) ID() string { return string(l.id) }

// removeStaleTemps deletes .tmp-* files older than an hour in the directory and consumers/,
// never younger ones, which may belong to a live consumer (decision 67).
func (l *Log) removeStaleTemps() {
	cutoff := l.cfg.clock().Add(-time.Hour)
	for _, dir := range []string{l.dir, l.path(consumersDir)} {
		removeTempsBefore(l.fs, dir, cutoff)
	}
}

// removeTempsBefore deletes the .tmp-* files in dir last modified before cutoff, best effort.
func removeTempsBefore(fsys fsx.FS, dir string, cutoff time.Time) {
	names, err := fsys.ReadDir(dir)
	if err != nil {
		return
	}
	for _, n := range names {
		if !strings.HasPrefix(n, tempPrefix) {
			continue
		}
		p := filepath.Join(dir, n)
		if info, err := fsys.Lstat(p); err == nil && info.ModTime().Before(cutoff) {
			_ = fsys.Remove(p)
		}
	}
}

// writePolicy records the settings the writer opened with (decision 29).
func (l *Log) writePolicy() error {
	c := l.cfg
	p := policy{Retention: c.Retention.String(), SegmentBytes: c.SegmentBytes, SegmentMaxAge: c.SegmentMaxAge.String(),
		MaxBytes: c.MaxBytes, MinFreeBytes: c.MinFreeBytes, MaxRecords: c.MaxRecords, MaxPayloadBytes: int64(c.MaxPayloadBytes),
		ConsumerTTL: c.ConsumerTTL.String(), ProtectedConsumers: c.ProtectedConsumers}
	if p.ProtectedConsumers == nil {
		p.ProtectedConsumers = []string{}
	}
	return writeState(l.fs, l.path(policyName), encodePolicy(p), l.cfg.Perm)
}
