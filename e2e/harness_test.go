package e2e

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"testing"
)

// runResult bundles the outputs of a single git-wt invocation.
type runResult struct {
	Stdout   string
	Stderr   string
	FD3      string // captured if RunWithFD is used; "" otherwise
	ExitCode int
}

// runWT invokes the test binary in cwd with args. Stdin is /dev/null.
func runWT(t *testing.T, cwd string, args ...string) runResult {
	t.Helper()
	return doRun(t, cwd, false, args...)
}

// runWTFD invokes the binary with fd 3 wired to a pipe. Use this to test the
// wrapper protocol: pass `--fd 3` (or rely on the default) and read FD3 from
// the result.
func runWTFD(t *testing.T, cwd string, args ...string) runResult {
	t.Helper()
	return doRun(t, cwd, true, args...)
}

func doRun(t *testing.T, cwd string, withFD bool, args ...string) runResult {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = cwd
	cmd.Env = hermeticEnv()
	cmd.Stdin = nil // ensures non-interactive auto-detection

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	var captured bytes.Buffer
	var wg sync.WaitGroup
	var fdReader, fdWriter *os.File
	if withFD {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("pipe: %v", err)
		}
		fdReader, fdWriter = r, w
		cmd.ExtraFiles = []*os.File{w} // becomes fd 3 in the child
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = io.Copy(&captured, r)
		}()
	}

	err := cmd.Run()
	if withFD {
		// Close our copy of the write end so the reader goroutine sees EOF
		// (the child has its own clone via ExtraFiles, already closed by Run).
		_ = fdWriter.Close()
		wg.Wait()
		_ = fdReader.Close()
	}

	res := runResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		FD3:    captured.String(),
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.ExitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run failed (not an exit error): %v", err)
	}
	return res
}

// hermeticEnv returns a minimal env that lets git run without depending on
// the developer's user.name/user.email and without picking up the global or
// system gitignore (which on developer machines often includes things like
// `.gwt/` — exactly the path our tests want to assert is *not* covered).
// PATH is preserved (we still need git).
func hermeticEnv() []string {
	keep := []string{"PATH", "HOME", "USER", "TMPDIR", "LANG", "LC_ALL"}
	env := []string{
		"GIT_AUTHOR_NAME=git-wt-test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=git-wt-test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		// Quiet down git's hint output that would otherwise flood stderr
		"GIT_TERMINAL_PROMPT=0",
		// Ignore developer-side gitconfig + gitignore_global so check-ignore
		// behaviour is reproducible.
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
	}
	for _, k := range keep {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}
