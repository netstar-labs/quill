//go:build linux || darwin

// Package atomicfile replaces a file the way a crash cannot half-observe: the data goes to a
// temp file beside the target, is fsynced, and is renamed over the target. Every operation
// goes through the fsx seam, so the crash simulator can test it.
//
// Write does not fsync the directory. After a process crash the rename is complete; after a
// power loss it may be undone, leaving the previous version, and never a torn or empty file,
// because the data was synced before the name moved (decision 40). Callers that need the
// rename itself to be durable call WriteSyncDir, or fsx.FS.SyncDir at the points decision 65
// names.
package atomicfile

import (
	"crypto/rand"
	"encoding/hex"
	"io/fs"
	"path/filepath"

	"github.com/netstar-labs/quill/internal/fsx"
)

// TempPrefix starts every temp file name. Scans ignore such names, and the writer removes
// those older than an hour when it opens a log (decision 67).
const TempPrefix = ".tmp-"

// Write replaces path with data: temp file (TempPrefix plus a random suffix, in the same
// directory, created exclusively and never through a symbolic link), chmod to perm exactly,
// write, fsync, rename. A failure at any step removes the temp file and leaves path as it was.
func Write(fsys fsx.FS, path string, data []byte, perm fs.FileMode) error {
	return WriteFrom(fsys, path, perm, func(f fsx.File) error {
		_, err := f.WriteAt(data, 0)
		return err
	})
}

// tempName is filepath.Join(dir, TempPrefix+hex(suffix)) with one allocation instead of three.
// dir comes from filepath.Dir, so it is already clean and Join's Clean would change nothing;
// only the two shapes Join treats specially remain: "." (a bare file name, no directory part)
// and "/" (the separator is already there). This file builds on linux and darwin only, so the
// separator is always "/".
func tempName(dir string, suffix [8]byte) string {
	var name [len(TempPrefix) + 2*len(suffix)]byte
	copy(name[:], TempPrefix)
	hex.Encode(name[len(TempPrefix):], suffix[:])
	switch dir {
	case ".":
		return string(name[:])
	case "/":
		return "/" + string(name[:])
	}
	return dir + "/" + string(name[:])
}

// WriteFrom is Write for content that is streamed rather than held in memory: fill writes the
// whole content into the temp file, which is then fsynced and renamed over path. An error from
// fill fails the write like any other step, removing the temp file and leaving path as it was.
func WriteFrom(fsys fsx.FS, path string, perm fs.FileMode, fill func(f fsx.File) error) error {
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return err
	}
	tmp := tempName(filepath.Dir(path), suffix)
	f, err := fsys.OpenFile(tmp, fsx.O_RDWR|fsx.O_CREATE|fsx.O_EXCL, perm)
	if err != nil {
		return err
	}
	err = f.Chmod(perm)
	if err == nil {
		err = fill(f)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil { // closed exactly once, whatever failed before
		err = cerr
	}
	if err == nil {
		err = fsys.Rename(tmp, path)
	}
	if err != nil {
		_ = fsys.Remove(tmp) // no failed write leaves a temp file behind
	}
	return err
}

// WriteSyncDir is Write followed by an fsync of the containing directory, so the rename
// itself survives a power loss.
func WriteSyncDir(fsys fsx.FS, path string, data []byte, perm fs.FileMode) error {
	if err := Write(fsys, path, data, perm); err != nil {
		return err
	}
	return fsys.SyncDir(filepath.Dir(path))
}
