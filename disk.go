package quill

import "syscall"

// freeBytes reads the free space available to unprivileged processes on the filesystem
// holding dir, with statfs (decision 25). A variable, so tests can inject a value.
var freeBytes = func(dir string) (uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return uint64(st.Bavail) * uint64(st.Bsize), nil
}
