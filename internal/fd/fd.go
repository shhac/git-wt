// Package fd implements the wrapper-protocol file-descriptor handling.
//
// Wrapper mode: the shell function from `git-wt alias` opens fd N via
// `N>&1 1>&2`. The binary detects this and writes the target path to fd N;
// the wrapper captures it and `cd`s the parent shell.
//
// Bare mode: fd N is not open; the binary writes the path to stdout
// instead, with a `→ cd '...'` hint on stderr. Supports
// `cd "$(git-wt go branch)"`.
//
// On Windows both Open and Available are stubbed to always return the
// not-open response — the wrapper protocol is a POSIX construct. The
// Windows binary always operates in bare mode; PowerShell / cmd.exe
// wrappers should use stdout capture.
package fd
