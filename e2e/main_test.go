// Package e2e drives the built git-wt binary against fresh temp repos
// under t.TempDir() (which lives under the OS temp dir, never inside this
// project). Each test gets a clean repo of its own; nothing mutates the
// developer's real worktrees.
//
// File layout:
//   - main_test.go      — TestMain (binary build) only
//   - harness_test.go   — runWT/runWTFD/doRun, runResult, hermeticEnv
//   - fixtures_test.go  — newRepo, newRepoWithRemote, mustGit, mustWrite,
//                         mustExist, mustNotExist, orphanBranch
//   - commands_test.go  — the actual test functions
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// binPath is set in TestMain to the freshly-built git-wt binary.
var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "git-wt-e2e-bin-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "mktemp:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binPath = filepath.Join(tmpDir, "git-wt")
	cmd := exec.Command("go", "build", "-o", binPath, "../cmd/git-wt")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "build failed:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
