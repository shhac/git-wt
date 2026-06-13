//go:build !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package wt

// clearImmutable is a no-op where BSD file flags don't exist. (Linux
// immutability via chattr +i requires CAP_LINUX_IMMUTABLE to clear — not
// something a user-level tool should attempt.)
func clearImmutable(string) {}
