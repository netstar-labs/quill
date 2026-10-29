package quill

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/netstar-labs/quill/internal/atomicfile"
	"github.com/netstar-labs/quill/internal/fsx"
)

// Archive copies the sealed segments of the log at srcDir that the consumer named _archive
// has not yet copied to dstDir, verifying every frame's CRC on the way, writes the
// destination's quill.format and quill.tail so it opens with OpenReader, then commits the
// consumer (decisions 28, 67, 68). It is idempotent, runs beside a live writer (it reads
// only sealed segments, through a Reader), and stops with an error at a corrupt sealed
// segment. When the source has lost segments the destination holds the same hole, and
// Archive returns a *GapError naming it after copying what exists. It refuses, before
// writing to it, a destination that is the source log itself and a destination whose
// writer lock is held (ErrLocked): quill.tail is the writer's file.
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
	c, err := r.consumerStore().open("_archive", ConsumerOptions{Protect: ProtectOn}, true)
	if err != nil {
		return err
	}
	defer c.Close()
	dstDir = filepath.Clean(dstDir)
	if err := archiveDest(fsys, srcDir, dstDir, r.id, r.opts.clock().Add(-time.Hour)); err != nil {
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
	var lastSeq, lastID uint64
	var lastSize int64
	copied := 0
	var copyErr error
	for _, s := range v.segs {
		if s.active || s.last < offset {
			continue
		}
		size, last, err := copySegment(fsys, s, dstDir)
		if err != nil {
			copyErr = err
			break
		}
		if gap == nil && copied > 0 && s.first != lastSeq+1 {
			gap = &GapError{From: lastSeq + 1, To: s.first - 1}
		}
		lastSeq, lastID, lastSize = last, s.id, size
		copied++
	}
	if copied > 0 {
		tl := tail{Seq: lastSeq, Seg: lastID, Off: uint64(lastSize), MaxSeg: lastID}
		if err := writeState(fsys, filepath.Join(dstDir, tailName), encodeTail(tl), DefaultPerm); err != nil {
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
func archiveDest(fsys fsx.FS, srcDir, dstDir string, id logID, staleBefore time.Time) error {
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
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	dirPerm := Config{Perm: DefaultPerm}.dirPerm()
	if err := fsys.Mkdir(dstDir, dirPerm); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	// The directory's own entry lives in its parent: syncing dstDir below makes what is inside
	// it durable, not the name that reaches it. Without this a power loss can keep the committed
	// _archive offset and lose the destination. Also done when an earlier run made the directory.
	if err := fsys.SyncDir(filepath.Dir(dstDir)); err != nil {
		return err
	}
	data, err := readState(fsys, filepath.Join(dstDir, formatName))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := writeState(fsys, filepath.Join(dstDir, formatName), encodeFormat(id), DefaultPerm); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		have, err := decodeFormat(data)
		if err != nil {
			return fmt.Errorf("%w: destination %s: %v", ErrCorrupt, formatName, err)
		}
		if have != id {
			return fmt.Errorf("%w: the destination archives log %s, the source is %s", ErrWrongLog, have, id)
		}
	}
	if err := fsys.Mkdir(filepath.Join(dstDir, consumersDir), dirPerm); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	if _, err := fsys.Lstat(filepath.Join(dstDir, tailName)); errors.Is(err, fs.ErrNotExist) {
		if err := writeState(fsys, filepath.Join(dstDir, tailName), encodeTail(tail{}), DefaultPerm); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	removeTempsBefore(fsys, dstDir, staleBefore)
	return fsys.SyncDir(dstDir)
}

// copySegment verifies a sealed segment whole, then copies it to the destination under its
// own name through atomicfile.WriteFrom (temp file, fsync, rename). A destination file of the same size is kept. It
// returns the size and the segment's last sequence.
func copySegment(fsys fsx.FS, s *segRef, dstDir string) (int64, uint64, error) {
	r, size, closer, err := s.open()
	if err != nil {
		return 0, 0, err
	}
	defer closer()
	info, err := verifySegment(r, size, fmt.Sprintf("segment %d", s.id))
	if err != nil {
		return 0, 0, err
	}
	dst := filepath.Join(dstDir, segmentName(s.id))
	if di, err := fsys.Lstat(dst); err == nil && di.Mode().IsRegular() && di.Size() == size {
		return size, info.last.Seq, nil
	}
	err = atomicfile.WriteFrom(fsys, dst, DefaultPerm, func(f fsx.File) error {
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
