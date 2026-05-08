//go:build !windows

package fd

import (
	"fmt"
	"os"
	"syscall"
)

// Open returns a writer for fd N if it's open in this process, or
// (nil, false) otherwise. The returned *os.File should be Closed by the
// caller (or left to process exit).
func Open(n int) (*os.File, bool) {
	var stat syscall.Stat_t
	if err := syscall.Fstat(n, &stat); err != nil {
		return nil, false
	}
	return os.NewFile(uintptr(n), fmt.Sprintf("fd%d", n)), true
}

// Available reports whether fd N is open in this process. It does not
// retain the file handle — for one-shot checks only.
func Available(n int) bool {
	var stat syscall.Stat_t
	return syscall.Fstat(n, &stat) == nil
}
