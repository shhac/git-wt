//go:build windows

package fd

import (
	"os"
	"testing"
)

// On Windows the wrapper protocol is unsupported. Both helpers always report
// the fd as unavailable, regardless of whether the underlying handle is
// actually open or closed.

func TestOpen_AlwaysFalseOnWindows(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	if _, ok := Open(int(r.Fd())); ok {
		t.Errorf("Open should always return false on Windows")
	}
	if _, ok := Open(999); ok {
		t.Errorf("Open(999) should be false on Windows")
	}
}

func TestAvailable_AlwaysFalseOnWindows(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	if Available(int(r.Fd())) {
		t.Errorf("Available should always be false on Windows")
	}
	if Available(999) {
		t.Errorf("Available(999) should be false on Windows")
	}
}
