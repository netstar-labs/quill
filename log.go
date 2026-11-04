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
	bgLogged [numBg]bool  // the operation's failure has been logged and its recovery has not
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

	markerAt   time.Time        // when the marker was last written
	dropped    uint64           // segments deleted while some consumer's offset was below them
	stale      []string         // protected consumers found stale at the last prune pass
	offsets    []consumerOffset // every readable consumer at the last prune pass (or Open), for Dropped at a seal
	diskFull   bool             // MinFreeBytes cannot be met: appends fail with ENOSPC
	freeAtPass uint64           // free bytes statfs reported at the last pass that could read it
	// lastFlushOK is when a flush last completed (wrote and published, or landed a failed marker publish), or Open's time before
	// the first one (Stats.LastFlushOK).
	lastFlushOK time.Time

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
//
// The lock is a flock, and a Reader's view rests on rename and fsync, so the directory must
// be on a local filesystem (ext4, xfs, btrfs, APFS) with one host or kernel writing to it.
// NFS, SMB/CIFS, FUSE filesystems (sshfs, s3fs and the like) and a directory shared between
// VMs or containers with separate kernels (virtiofs, 9p) do not provide these coherently.
// Open does not detect them: on one, two writers can both believe they hold the lock and
// corrupt the log, and nothing fails at write time.
//
// A second Open beside a live writer fails with ErrLocked. That is so only for an account that
// can open quill.lock, which is always 0600 (decision 51): under a different account than the
// one that created the log (root, which bypasses file permissions, excepted) Open fails with the
// permission error from opening quill.lock (EACCES, which matches fs.ErrPermission and names
// the file), not ErrLocked, whether or not a writer is live. The lock is never tried, so quill
// cannot say a writer is live, and for an account that cannot open the lock the log is almost
// always someone else's. A caller that wants to detect a live writer treats both as "not this
// account's log, or a writer may be live". Archive differs: it only refuses, so it reports the
// same condition as an error that is both ErrLocked and fs.ErrPermission.
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
	l := &Log{cfg: cfg, fs: cfg.fs, dir: dir, lastFlushOK: cfg.clock(),
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
	// A consumer ahead of the head (a log restored to an older state) resumes mid-log, unflagged,
	// once the head passes it; say so while the operator can still rewind it (decision 36). It
	// reuses the read above, and like the other Open warnings runs with no lock held.
	for _, o := range l.offsets {
		if head := l.Head(); o.offset > head+1 {
			l.logf(slog.LevelWarn, "consumer offset is ahead of the head; rewind it with quill commit", "consumer", o.name, "offset", o.offset, "head", head)
		}
	}
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
// an interrupted creation. consumers/ is created if it is missing, whatever else the log holds;
// the first segment is created only when the log holds no segments and no non-zero marker;
// quill.tail is created only when no segment holds a byte, and a marker-less log with a
// segment that does is ErrCorrupt (decision 62). The lock is held.
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

// bgInfo names an operation in the log, and gives the Warn line written when it starts to fail.
// An empty warn means its caller logs the transitions itself (the policy write).
var bgInfo = [numBg]struct{ op, warn string }{
	bgFlush:   {"flush", "cannot write the buffered records; they stay in memory and the write is retried"},
	bgPublish: {"marker publish", "cannot publish quill.tail; readers keep the old marker and the publish is retried"},
	bgRotate:  {"rotation", "cannot create the next segment; the active one keeps growing and the rotation is retried"},
	bgPrune:   {"segment delete", "cannot delete segment"},
}

// setBg records the live error of an operation, replacing its earlier one. The first failure
// since the operation last succeeded is logged at Warn, with attrs; a retry that fails again is
// not (decision 60). Called with the lock held.
func (l *Log) setBg(k bgKind, err error, attrs ...any) {
	l.bg[k] = err
	if info := bgInfo[k]; info.warn != "" && !l.bgLogged[k] {
		l.bgLogged[k] = true
		l.logf(slog.LevelWarn, info.warn, append([]any{"op", info.op, "err", err}, attrs...)...)
	}
}

// clearBg drops the error of an operation that has just succeeded, and logs at Info that it
// recovered if its failure was logged. Called with the lock held.
func (l *Log) clearBg(k bgKind) {
	l.bg[k] = nil
	if l.bgLogged[k] {
		l.bgLogged[k] = false
		l.logf(slog.LevelInfo, "background operation recovered", "op", bgInfo[k].op)
	}
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
