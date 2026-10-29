package quill

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/netstar-labs/quill/internal/atomicfile"
	"github.com/netstar-labs/quill/internal/fsx"
)

// Archive copies the sealed segments of the log at srcDir that the consumer named _archive
// has not yet copied to dstDir, verifying every frame's CRC on the way, writes the
// destination's quill.format and quill.tail so it opens with OpenReader, then commits the
// consumer (decisions 28, 67, 68). It is idempotent, runs beside a live writer (it reads
// only sealed segments, through a Reader), and stops with an error at a corrupt sealed
// segment. When the source has lost segments the destination holds the same hole, and
// Archive returns a *GapError naming it after copying what exists.
//
// The archiver is a protected consumer, so retention keeps a segment until it is copied;
// the caps still win, so an archive can fall behind under disk pressure.
func Archive(srcDir, dstDir string) error {
	return archive(fsx.OS{}, srcDir, dstDir, 0o600)
}

func archive(fsys fsx.FS, srcDir, dstDir string, perm fs.FileMode) error {
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
	if err := archiveDest(fsys, dstDir, r.id, perm); err != nil {
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
		size, last, err := copySegment(fsys, s, dstDir, perm)
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
		if err := writeState(fsys, filepath.Join(dstDir, tailName), encodeTail(tl), perm); err != nil {
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

// archiveDest creates the destination if needed and checks or writes its format marker.
func archiveDest(fsys fsx.FS, dstDir string, id logID, perm fs.FileMode) error {
	dirPerm := perm | (perm&0o444)>>2
	if err := fsys.Mkdir(dstDir, dirPerm); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	data, err := readState(fsys, filepath.Join(dstDir, formatName))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := writeState(fsys, filepath.Join(dstDir, formatName), encodeFormat(id), perm); err != nil {
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
		if err := writeState(fsys, filepath.Join(dstDir, tailName), encodeTail(tail{}), perm); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return fsys.SyncDir(dstDir)
}

// copySegment verifies a sealed segment whole, then copies it to the destination under its
// own name through atomicfile.WriteFrom (temp file, fsync, rename). A destination file of the same size is kept. It
// returns the size and the segment's last sequence.
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
	dst := filepath.Join(dstDir, segmentName(s.id))
	if di, err := fsys.Lstat(dst); err == nil && di.Mode().IsRegular() && di.Size() == size {
		return size, info.last.Seq, nil
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
