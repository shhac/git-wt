//go:build windows

package fd

import "os"

// On Windows the wrapper protocol is unsupported (PowerShell / cmd.exe don't
// support the `N>&1 1>&2` redirect). The binary always operates in bare mode
// — the picker UI on stderr, the path on stdout — and PowerShell / cmd
// wrappers should capture stdout normally:
//
//	# PowerShell
//	function gwt {
//	    $target = & git-wt.exe @args
//	    if ($target) { Set-Location $target }
//	}
//
//	:: cmd.exe via doskey
//	doskey gwt=for /f "delims=" %%i in ('git-wt.exe $*') do cd /d "%%i"
//
// Both Open and Available always report the fd as unavailable.

// Open always returns (nil, false) on Windows.
func Open(_ int) (*os.File, bool) { return nil, false }

// Available always returns false on Windows.
func Available(_ int) bool { return false }
