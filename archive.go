package quill

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/netstar-labs/quill/internal/atomicfile"
	"github.com/netstar-labs/quill/internal/fsx"
)

// Archive copies the sealed segments of the log at srcDir that the consumer named _archive
// has not yet copied to dstDir, verifying every frame's CRC on the way, writes the
// destination's quill.format and quill.tail so it opens with OpenReader, then commits the
// consumer (decisions 28, 67, 68). It is idempotent, runs beside a live writer (it reads
// only sealed segments, through a Reader), and stops with an error at a corrupt sealed
// segment, and at two segments that overlap (ErrCorrupt, as a Reader and Open do, decision
// 102), before it copies the earlier of the two. When the source has lost segments the
// destination holds the same hole, and Archive returns a *GapError naming it after copying what exists. The gap is found by the
// run that copies past it, in two ways: the _archive offset is below the first retained
// sequence or a segment is missing between two it copies, or the destination's quill.tail
// names a last record below the first segment it copies, which is how a loss shows after the
// consumer was removed and recreated at the first retained sequence (at 1 when a damaged oldest
// segment left a gap below it).
// Later runs have nothing to copy and return nil; `quill check` on the destination keeps
// reporting the hole. It refuses, before
// writing to it, a destination that is the source log itself and a destination whose
// writer lock is held (ErrLocked): quill.tail is the writer's file. quill.lock is always 0600,
// so a destination lock this account cannot open (it belongs to another account) is treated as
// a writer that may be live: Archive refuses with an error that is both ErrLocked and
// fs.ErrPermission (decision 51); a destination directory that cannot be searched is refused
// with a permission error that names the directory and claims no writer. A destination that
// exists and has no quill.format must be empty, ignoring .tmp-* files and quill.lock as
// Open does for a new log; otherwise it is not a log and Archive returns ErrNoLog without
// writing to it, so a mistyped destination is never written into and no file in it is
// replaced (decision 85). A destination whose quill.format is newer than this build is
// ErrFutureVersion, another log's is ErrWrongLog, and any other unusable marker is ErrCorrupt.
//
// The destination's segments and state files, and the source's consumer files, take the
// source log's file mode exactly, read from its quill.tail (decision 51). The destination's
// directories are requested with the matching execute bits and go through the process
// umask, so they can come out narrower.
//
// The archiver is a protected consumer, so retention keeps a segment until it is copied;
// the caps still win, so an archive can fall behind under disk pressure.
func Archive(srcDir, dstDir string) error {
	return archive(fsx.OS{}, srcDir, dstDir)
}

func archive(fsys fsx.FS, srcDir, dstDir string) error {
	r, err := OpenReader(srcDir, ReaderOptions{fs: fsys})
	if err != nil {
		return err
	}
	defer r.Close()
	c, err := r.consumerStore().open(archiveConsumer, ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		return err
	}
	defer c.Close()
	dstDir = filepath.Clean(dstDir)
	if err := archiveDest(fsys, srcDir, dstDir, r.id, r.perm, r.opts.clock().Add(-time.Hour)); err != nil {
		return err
	}
	v, release, err := r.view()
	if err != nil {
		return err
	}
	defer release()
	offset := c.Offset()
	var gap *GapError
	if offset < v.first {
		gap = &GapError{From: offset, To: v.first - 1}
	}
	// What the destination holds: its quill.tail names the last record of the last run. A zero
	// Seq is a new destination (the whole source is then the archive's to copy) and a marker
	// that cannot be read is about to be replaced, so neither says anything about a hole.
	var destSeq uint64
	if data, err := readState(fsys, filepath.Join(dstDir, tailName)); err == nil {
		if dt, err := decodeTail(data); err == nil {
			destSeq = dt.Seq
		}
	}
	var lastSeq, lastID uint64
	var lastSize int64
	copied := 0
	var copyErr error
	for _, s := range v.segs {
		if s.active || s.last < offset {
			continue
		}
		size, last, err := copySegment(fsys, s, dstDir, r.perm)
		if err != nil {
			copyErr = err
			break
		}
		if gap == nil && copied > 0 && s.first != lastSeq+1 {
			gap = &GapError{From: lastSeq + 1, To: s.first - 1}
		}
		// The first segment copied begins past the destination's last record: the records between
		// were lost from the source before any archiver copied them, which the offset cannot show
		// after the consumer was removed and recreated at the first retained sequence (or at 1 when a
		// damaged oldest segment left a gap below it).
		if gap == nil && copied == 0 && destSeq > 0 && s.first > destSeq+1 {
			gap = &GapError{From: destSeq + 1, To: s.first - 1}
		}
		lastSeq, lastID, lastSize = last, s.id, size
		copied++
	}
	if copied > 0 {
		tl := tail{Seq: lastSeq, Seg: lastID, Off: uint64(lastSize), MaxSeg: lastID}
		if err := writeState(fsys, filepath.Join(dstDir, tailName), encodeTail(tl), r.perm); err != nil {
			return err
		}
		if err := fsys.SyncDir(dstDir); err != nil {
			return err
		}
		if err := c.Commit(lastSeq + 1); err != nil {
			return err
		}
	}
	if copyErr != nil {
		return copyErr
	}
	if gap != nil {
		return gap
	}
	return nil
}

// archiveDest refuses a destination that has a writer of its own, then creates the
// destination if needed and checks or writes its format marker. It also removes the .tmp-*
// files in it last modified before staleBefore: a killed copy leaves one, and no writer's Open
// sweeps this directory (decision 67).
func archiveDest(fsys fsx.FS, srcDir, dstDir string, id logID, perm fs.FileMode, staleBefore time.Time) error {
	// quill.tail has one writer, the lock holder (decisions 8, 62), and Archive writes the
	// destination's. The source is such a destination when the two paths name one directory
	// (os.SameFile follows a symbolic link in the path, which comparing strings would not).
	if si, err := fsys.Lstat(filepath.Join(srcDir, tailName)); err == nil {
		if di, err := fsys.Lstat(filepath.Join(dstDir, tailName)); err == nil && os.SameFile(si, di) {
			return fmt.Errorf("quill: archive destination %s is the source log", dstDir)
		}
	}
	// A live writer on any other directory holds its lock; the lock file is not created here.
	if f, err := fsys.OpenFile(filepath.Join(dstDir, lockName), fsx.O_RDWR, DefaultPerm); err == nil {
		err = f.TryLock()
		_ = f.Close()
		if errors.Is(err, fsx.ErrLocked) {
			return fmt.Errorf("%w: archive destination %s has a live writer", ErrLocked, dstDir)
		}
	} else if errors.Is(err, fs.ErrPermission) {
		// quill.lock is 0600, so a lock this account can see but cannot open belongs to another
		// account: the writer of that log, which may be live. Fail closed. If the lock cannot even
		// be seen, the directory (or a parent) cannot be searched: that is a permission problem
		// with the destination, not evidence of a writer, so it does not claim one.
		lock := filepath.Join(dstDir, lockName)
		if _, serr := fsys.Lstat(lock); serr != nil {
			return fmt.Errorf("quill: archive destination %s cannot be searched, so nothing is written to it: %w", dstDir, err)
		}
		return fmt.Errorf("%w: archive destination %s: its quill.lock cannot be opened, so a writer may be live: %w", ErrLocked, dstDir, err)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	dirPerm := Config{Perm: perm}.dirPerm()
	if err := fsys.Mkdir(dstDir, dirPerm); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	// The directory's own entry lives in its parent: syncing dstDir below makes what is inside
	// it durable, not the name that reaches it. Without this a power loss can keep the committed
	// _archive offset and lose the destination. Also done when an earlier run made the directory.
	if err := fsys.SyncDir(filepath.Dir(dstDir)); err != nil {
		return err
	}
	have, err := loadFormat(fsys, filepath.Join(dstDir, formatName))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// No marker: the directory is new only if it is empty, as Log.create requires (decision
		// 85). Archive writes the marker before anything else, so a destination left by an
		// interrupted run has one and does not come here.
		names, err := fsys.ReadDir(dstDir)
		if err != nil {
			return err
		}
		for _, n := range names {
			if n != lockName && !strings.HasPrefix(n, tempPrefix) {
				return fmt.Errorf("%w: archive destination %s is not empty and has no %s: archive into a new or empty directory, or into an existing archive of this log", ErrNoLog, dstDir, formatName)
			}
		}
		if err := writeState(fsys, filepath.Join(dstDir, formatName), encodeFormat(id), perm); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if have != id {
			return fmt.Errorf("%w: the destination archives log %s, the source is %s", ErrWrongLog, have, id)
		}
	}
	if err := fsys.Mkdir(filepath.Join(dstDir, consumersDir), dirPerm); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	if _, err := fsys.Lstat(filepath.Join(dstDir, tailName)); errors.Is(err, fs.ErrNotExist) {
		if err := writeState(fsys, filepath.Join(dstDir, tailName), encodeTail(tail{}), perm); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	removeTempsBefore(fsys, dstDir, staleBefore)
	return fsys.SyncDir(dstDir)
}

// copySegment verifies a sealed segment whole, then copies it to the destination under its
// own name through atomicfile.WriteFrom (temp file, fsync, rename). A destination file of the
// same size is read and verified the same way and kept only when it is intact and holds the
// same sequence range; a damaged one is replaced (decision 85), a healthy one is not
// rewritten. It returns the size and the segment's last sequence.
func copySegment(fsys fsx.FS, s *segRef, dstDir string, perm fs.FileMode) (int64, uint64, error) {
	r, size, closer, err := s.open()
	if err != nil {
		return 0, 0, err
	}
	defer closer()
	info, err := verifySegment(r, size, fmt.Sprintf("segment %d", s.id))
	if err != nil {
		return 0, 0, err
	}
	// The view derived this segment's last sequence from where the next segment starts, which
	// cannot show an overlap; the whole-segment verification just learned the real one. Frames
	// past the derived end are the overlap Open refuses (decision 102): copying them would
	// publish a marker below the last record copied.
	if info.last.Seq > s.last {
		return 0, 0, fmt.Errorf("%w: segment %d holds sequence %d, inside the next segment (starts at %d)", ErrCorrupt, s.id, info.last.Seq, s.last+1)
	}
	dst := filepath.Join(dstDir, segmentName(s.id))
	if di, err := fsys.Lstat(dst); err == nil && di.Mode().IsRegular() && di.Size() == size {
		kept, err := destIntact(fsys, dst, size, info)
		if err != nil {
			return 0, 0, err
		}
		if kept {
			return size, info.last.Seq, nil
		}
	}
	err = atomicfile.WriteFrom(fsys, dst, perm, func(f fsx.File) error {
		n, err := io.CopyBuffer(io.NewOffsetWriter(f, 0), io.NewSectionReader(r, 0, size), make([]byte, windowChunk))
		if err == nil && n < size {
			err = io.ErrUnexpectedEOF
		}
		return err
	})
	if err != nil {
		return 0, 0, err
	}
	return size, info.last.Seq, nil
}

// destIntact reports whether the destination segment at path, which has the source's size,
// verifies whole and covers the same sequences as the source's info. Damage found by the
// verification (a *SegmentError) is false, so the caller replaces the file; a read that fails
// is an error, since nothing is known about the file.
func destIntact(fsys fsx.FS, path string, size int64, src segInfo) (bool, error) {
	f, err := fsys.OpenFile(path, fsx.O_RDONLY, 0)
	if err != nil {
		return false, err
	}
	defer f.Close()
	di, err := verifySegment(f, size, path)
	var se *SegmentError
	if errors.As(err, &se) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return di.first.Seq == src.first.Seq && di.last.Seq == src.last.Seq, nil
}
