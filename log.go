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
	// Dir is the log's directory, created if missing. Its base name is the topic name. A
	// symbolic link to a directory is followed, as OpenReader follows one, whether it is Dir
	// itself or a directory above it; the files inside the log directory never are. A relative
	// Dir is made absolute at Open, so a later os.Chdir does not move the log.
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
	// MinFreeBytes is a free-space floor on the log's filesystem (0: off). It is checked at each
	// seal and prune pass. With nothing left to delete, appends fail with an error wrapping
	// ENOSPC, and Err() and Stat().DiskFull report it until a pass finds room. When statfs fails,
	// Err() reports that until a later pass can read the free space, and the floor is not enforced
	// meanwhile.
	MinFreeBytes int64
	// MaxPayloadBytes is the largest accepted record (64 MiB − 32, the ceiling).
	MaxPayloadBytes int
	// MaxBatchBytes bounds an AppendBatch of more than one record (16 MiB). The check is on an
	// upper bound of the encoded size, len(payload)+34 per record, so a batch of many small records
	// is refused before its real encoded size reaches the limit (about 479,000 one-byte records at
	// the default).
	MaxBatchBytes int
	// MaxReadBytes is the payload bytes after which one Read stops: it returns at least one record,
	// and the last record may carry the total past this bound (16 MiB).
	MaxReadBytes int
	// ProtectedConsumers names consumers the writer creates, protected, when it opens the log. A
	// file already protected is left alone, so a running consumer does not block a restart.
	ProtectedConsumers []string
	// Logger receives retention deletions, stale consumers, bulk-expiry warnings and failures,
	// and a warning from Open when it cuts the newest segment past the marker (with the segment,
	// the offset, the bytes cut and whether the first rejected frame was incomplete or damaged).
	// Its handler, Enabled included, runs synchronously with the log's lock held, on the
	// goroutine that made the record (an Append, Flush, Roll, Prune or Close call, or the
	// log's flusher). It must therefore neither call the Log, which deadlocks it, nor block,
	// which stalls every Append, Read, Sync and the flusher: hand the record to another
	// goroutine and return.
	Logger *slog.Logger
	// Perm is the file mode (0600); directories get the matching execute bits (0700). State
	// files (quill.format, quill.tail, quill.policy, consumer offsets) are exactly Perm. The
	// writer's segments, its consumers' lock files and every directory are created through the
	// process umask, so under umask 022 Perm 0660 gives 0640 segments and 0750 directories.
	// quill.lock is always 0600, whatever Perm says, so a reader-only process cannot take the
	// writer's lock (decision 51).
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
	case c.maxActiveBytes() >= maxSegmentFileBytes:
		return bad("SegmentBytes %d plus one flush (up to %d bytes) reaches the %d-byte segment file limit",
			c.SegmentBytes, c.maxActiveBytes()-c.SegmentBytes, int64(maxSegmentFileBytes))
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
// Each flush term is capped at the file limit before the sum, so a size near math.MaxInt64
// cannot wrap the sum negative past validate; a capped term alone already reaches the limit.
func (c Config) maxActiveBytes() int64 {
	capped := func(n int) int64 { return min(int64(n), maxSegmentFileBytes) }
	return c.SegmentBytes + capped(max(c.FlushBytes, c.MaxBufferBytes)) +
		max(capped(c.MaxBatchBytes), capped(c.MaxPayloadBytes)+34)
}

// dirPerm is the directory mode for a file mode: execute wherever read is granted.
func (c Config) dirPerm() fs.FileMode {
	return c.Perm | (c.Perm&0o444)>>2
}

// archiveConsumer is the consumer `Archive` keeps. Its leading underscore is reserved for it
// alone (decision 41).
const archiveConsumer = "_archive"

// validConsumerName is validName plus exactly the archiver's name, so that the consumer
// `Archive` creates can be rewound and removed through the public paths (decision 41).
func validConsumerName(name string) error {
	if name == archiveConsumer {
		return nil
	}
	return validName(name)
}

// validName checks a consumer name: [a-z0-9_-], 1 to 200 characters, no leading underscore
// (reserved). Config.ProtectedConsumers is held to it; the archiver's name is not accepted there.
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
	marker   tail         // the last published quill.tail
	tailBuf  []byte       // scratch for encoding the marker; used only under mu
	failed   error        // sticky: an fsync error wrapping ErrFailed
	bg       [numBg]error // the live background errors, one slot per operation, for Err()
	flushErr error        // the last flush's error, while flushes are failing; nil once one succeeds
	loadErr  error        // a sticky load error (a sealed segment that failed), for Err()
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

	markerAt   time.Time // when the marker was last written
	dropped    uint64    // segments deleted while some consumer's offset was below them
	stale      []string  // protected consumers found stale at the last prune pass
	offsets    []uint64  // every readable consumer's offset at the last prune pass (or Open), for Dropped at a seal
	diskFull   bool      // MinFreeBytes cannot be met: appends fail with ENOSPC
	freeAtPass uint64    // free bytes statfs reported at the last pass that could read it

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
	abs, err := filepath.Abs(cfg.Dir) // bound to the working directory now, not at each later syscall
	if err != nil {
		return nil, err
	}
	dir, err := resolveDirLink(cfg.fs, abs)
	if err != nil {
		return nil, err
	}
	l := &Log{cfg: cfg, fs: cfg.fs, dir: dir,
		flushReq: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	l.tailPath = l.path(tailName)
	l.bufPool.New = func() any { b := make([]byte, 0, windowChunk); return &b }
	if err := l.mkdirAll(l.dir); err != nil {
		return nil, err
	}
	lock, err := takeWriterLock(l.fs, l.path(lockName), func(err error) {
		l.logf(slog.LevelWarn, "quill.lock is wider than 0600 and could not be tightened: a process that can read it can take the writer's lock", "path", l.path(lockName), "error", err)
	})
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
		l.recordPolicy()
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
	l.offsets = l.consumers(cfg.clock()).offsets // Dropped at a seal before the first prune pass
	go l.flusher()
	return l, nil
}

// resolveDirLink lets a log directory be reached through a symbolic link, as OpenReader allows
// and a deployment that links its data directory needs: the deepest part of dir that exists,
// when it is itself a link, is resolved once, and what is missing below it is appended. Opening
// a directory to fsync it never follows a link (decision 51), so the log must live at the
// resolved path; the files inside it are still opened without following one. A link that does
// not lead to a directory is ErrNoLog. Anything else is left for mkdirAll to decide.
func resolveDirLink(fsys fsx.FS, dir string) (string, error) {
	existing, missing := dir, ""
	for {
		info, err := fsys.Lstat(existing)
		if err == nil {
			if info.Mode()&fs.ModeSymlink == 0 {
				return dir, nil
			}
			break
		}
		parent := filepath.Dir(existing)
		if !errors.Is(err, fs.ErrNotExist) || parent == existing {
			return dir, nil
		}
		missing, existing = filepath.Join(filepath.Base(existing), missing), parent
	}
	// Only a real file system reports a link (the crash simulator has none), so resolving
	// through the os package here does not bypass the seam.
	resolved, err := filepath.EvalSymlinks(existing)
	if err == nil {
		var info fs.FileInfo
		if info, err = fsys.Lstat(resolved); err == nil && info.IsDir() {
			return filepath.Join(resolved, missing), nil
		}
	}
	return "", fmt.Errorf("%w: %s is a symbolic link that does not lead to a directory", ErrNoLog, existing)
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

// Err returns what is wrong now: the failed state (ErrFailed) or, when the log is not failed, the
// sticky load error of the first damaged sealed segment, joined with every background error that
// is still live (errors.Join; errors.Is matches each). A background error is dropped when the
// operation that set it succeeds, so it is nil when nothing is failing and the log holds no
// damaged segment, and it does not disappear because an older error is also set. The load error
// goes when retention deletes the segment it names. With MinFreeBytes set, a free-space floor that
// cannot be met (ENOSPC) or checked (statfs failed) is one of the background errors, until a pass reads room.
func (l *Log) Err() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var errs [1 + numBg]error
	errs[0] = l.failed
	if errs[0] == nil {
		errs[0] = l.loadErr
	}
	copy(errs[1:], l.bg[:])
	var only error
	n := 0
	for _, e := range errs {
		if e != nil {
			only = e
			n++
		}
	}
	if n <= 1 {
		return only
	}
	return errors.Join(errs[:]...)
}

// bgKind names the slot of a background error: each is cleared by the operation that sets it.
type bgKind uint8

const (
	bgFlush   bgKind = iota // a data write
	bgPublish               // the marker's temp file, rename or the fsync after it
	bgRotate                // creating the next segment
	bgPrune                 // unlinking a deleted segment
	bgPolicy                // writing quill.policy; a flush and a prune pass retry it while it is set (decision 29)
	bgDisk                  // MinFreeBytes: statfs failed at the last pass, or diskFull is set (decision 25)
	numBg
)

// setBg records the live error of an operation, replacing its earlier one. Called with the lock held.
func (l *Log) setBg(k bgKind, err error) { l.bg[k] = err }

// clearBg drops the error of an operation that has just succeeded. Called with the lock held.
func (l *Log) clearBg(k bgKind) { l.bg[k] = nil }

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

// recordPolicy writes quill.policy, or retries it after a failure. A failed write removes the
// file instead of leaving the previous writer's settings there, where check, ls and an offline
// prune would read them as this writer's promise (decision 29); the error goes to Err() and the
// first one to the logger. The next flush or prune pass, or Close, tries again. Called with the
// log's lock held (Open, before the flusher starts, is the one caller that has none to take).
func (l *Log) recordPolicy() {
	err := l.writePolicy()
	if err == nil {
		if l.bg[bgPolicy] != nil {
			l.clearBg(bgPolicy)
			l.logf(slog.LevelInfo, "quill.policy written after an earlier failure")
		}
		return
	}
	first := l.bg[bgPolicy] == nil
	if first {
		l.logf(slog.LevelWarn, "quill.policy could not be written; the previous one is removed and the write is retried at each flush and prune pass", "err", err)
	}
	_ = l.fs.Remove(l.path(policyName)) // best effort; a missing file is the point
	l.setBg(bgPolicy, err)
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
