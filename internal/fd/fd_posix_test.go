//go:build !windows

package fd

import (
	"os"
	"testing"
)

func TestOpen_OpenFD(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	f, ok := Open(int(r.Fd()))
	if !ok {
		t.Errorf("expected Open to succeed for an open pipe fd, got false")
	}
	if f == nil {
		t.Errorf("expected a non-nil *os.File")
	}
}

func TestOpen_ClosedFD(t *testing.T) {
	// fd 999 is essentially guaranteed to be closed in our process.
	_, ok := Open(999)
	if ok {
		t.Errorf("expected Open(999) to return ok=false for a closed fd")
	}
}

func TestAvailable(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	if !Available(int(r.Fd())) {
		t.Errorf("Available should be true for an open fd")
	}
	if Available(999) {
		t.Errorf("Available(999) should be false")
	}
}
