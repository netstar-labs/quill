package quill

import "syscall"

// freeBytes reads the free space available to unprivileged processes on the filesystem
// holding dir, with statfs (decision 25). Tests inject a value through Config.statfs.
func freeBytes(dir string) (uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return availBytes(&st), nil
}

// availBytes is the unit and field choice of freeBytes: blocks available to unprivileged
// processes (Bavail, not Bfree, which counts the root reserve) times the block size. It is
// separate so a test can pin both on a made-up Statfs_t, whatever the filesystem it runs on.
func availBytes(st *syscall.Statfs_t) uint64 {
	return uint64(st.Bavail) * uint64(st.Bsize)
}
