//go:build !windows

package fd

import (
	"fmt"
	"os"
	"syscall"
)

// Open returns a writer for fd N if it is open and writable in this process,
// or (nil, false) otherwise. The returned *os.File should be Closed by the
// caller (or left to process exit).
//
// We require write-mode because container runtimes (Docker, runc, GitHub
// Actions Linux runners) routinely leak a read-only descriptor — typically
// /sys/fs/cgroup/cpu.max — as fd 3 in every child process. Without the
// writable check, an unwrapped invocation would attempt to write to that
// read-only fd and fail with EBADF.
func Open(n int) (*os.File, bool) {
	if !Available(n) {
		return nil, false
	}
	return os.NewFile(uintptr(n), fmt.Sprintf("fd%d", n)), true
}

// Available reports whether fd N is open and writable in this process. It
// does not retain the file handle — for one-shot checks only.
func Available(n int) bool {
	flags, err := fcntlGetFlags(n)
	if err != nil {
		return false
	}
	mode := flags & syscall.O_ACCMODE
	return mode == syscall.O_WRONLY || mode == syscall.O_RDWR
}

func fcntlGetFlags(fd int) (int, error) {
	r, _, e := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(syscall.F_GETFL), 0)
	if e != 0 {
		return 0, e
	}
	return int(r), nil
}
