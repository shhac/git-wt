//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package wt

import "golang.org/x/sys/unix"

// clearImmutable strips BSD file flags (uchg & friends) that block deletion
// even for the owner. Best-effort: failures surface later as the unlink error.
func clearImmutable(path string) {
	_ = unix.Chflags(path, 0)
}
