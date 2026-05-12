//go:build !windows

package fd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_OpenFD(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	// The write end of a pipe is opened O_WRONLY — what a real wrapper
	// passes us. Open should succeed on this.
	f, ok := Open(int(w.Fd()))
	if !ok {
		t.Errorf("expected Open to succeed for the write end of a pipe, got false")
	}
	if f == nil {
		t.Errorf("expected a non-nil *os.File")
	}
}

func TestOpen_ReadOnlyFD(t *testing.T) {
	// Container runtimes (Docker, runc, GitHub Actions Linux runners) often
	// leak a read-only fd as fd 3 in every child. Without a write-mode
	// check, Open would return ok=true and the caller would EBADF on write.
	tmp := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(tmp, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, ok := Open(int(f.Fd())); ok {
		t.Errorf("expected Open to refuse a read-only fd, got ok=true")
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

	if !Available(int(w.Fd())) {
		t.Errorf("Available should be true for the write end of a pipe")
	}
	if Available(int(r.Fd())) {
		t.Errorf("Available should be false for a read-only fd (read end of a pipe)")
	}
	if Available(999) {
		t.Errorf("Available(999) should be false")
	}
}
