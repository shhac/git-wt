package wt

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// RenameAside moves dir to a sibling "<dir>.removing-<pid>[-<n>]" so the
// original path frees up immediately (renames within a directory are atomic
// and instant, unlike recursive deletion). Returns the new path. Leftover
// candidates from a crashed earlier run are skipped by suffix.
func RenameAside(dir string) (string, error) {
	var lastErr error
	for i := 0; i < 10; i++ {
		cand := fmt.Sprintf("%s.removing-%d-%d", dir, os.Getpid(), i)
		if err := os.Rename(dir, cand); err == nil {
			return cand, nil
		} else {
			lastErr = err
		}
	}
	return "", fmt.Errorf("rename %s aside: %w", dir, lastErr)
}

// deleteWorkers caps the unlink pool. Parallel unlinking is contention-bound
// (filesystem locks), not CPU-bound; on APFS throughput roughly doubles by 8
// workers and degrades past that.
func deleteWorkers() int {
	if n := runtime.NumCPU(); n < 8 {
		return n
	}
	return 8
}

// DeleteTree removes root and everything beneath it, fixing the conditions
// that make a plain recursive delete strand a half-removed tree: directories
// without owner write/exec permission, and (on platforms that have them)
// immutable file flags. Symlinks are unlinked, never followed. Deletion fans
// out across subtrees on a small worker pool — large trees go roughly twice
// as fast as `rm -rf`.
//
// progress, if non-nil, is called from a separate goroutine roughly every
// 100ms with the running count of entries deleted, and once more (from the
// calling goroutine) with the final count after all workers finish.
func DeleteTree(root string, progress func(deleted int)) error {
	var count atomic.Int64
	stop := make(chan struct{})
	var ticker sync.WaitGroup
	if progress != nil {
		ticker.Add(1)
		go func() {
			defer ticker.Done()
			t := time.NewTicker(100 * time.Millisecond)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					progress(int(count.Load()))
				}
			}
		}()
	}

	err := deleteTreeParallel(root, &count)
	close(stop)
	ticker.Wait()
	if progress != nil {
		progress(int(count.Load()))
	}
	return err
}

// deleteTreeParallel removes root by fanning its depth-2 subdirectories
// (the widest level of typical cache/dependency trees) out to the worker
// pool, then sweeping the remaining skeleton serially. Subtrees are
// disjoint, so workers never contend on the same entries.
func deleteTreeParallel(root string, count *atomic.Int64) error {
	tick := func() { count.Add(1) }

	units, err := collectSubtrees(root)
	if err != nil || len(units) < 2 {
		return removeAllFixing(root, tick)
	}

	jobs := make(chan string)
	var firstErr error
	var mu sync.Mutex
	var wg sync.WaitGroup
	for w := 0; w < deleteWorkers(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				if err := removeAllFixing(p, tick); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
			}
		}()
	}
	for _, u := range units {
		jobs <- u
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	return removeAllFixing(root, tick)
}

// collectSubtrees returns the depth-2 directories under root (or a depth-1
// directory itself when it has no subdirectories) as disjoint parallel work
// units. Directories are fixed up (immutable flags, owner rwx) as they are
// listed, since listing is the same access the deletion will need.
func collectSubtrees(root string) ([]string, error) {
	level1, err := listSubdirsFixing(root)
	if err != nil {
		return nil, err
	}
	var units []string
	for _, d1 := range level1 {
		p1 := filepath.Join(root, d1)
		level2, err := listSubdirsFixing(p1)
		if err != nil || len(level2) == 0 {
			units = append(units, p1)
			continue
		}
		for _, d2 := range level2 {
			units = append(units, filepath.Join(p1, d2))
		}
	}
	return units, nil
}

// listSubdirsFixing lists dir's immediate subdirectory names (symlinks
// excluded — ReadDir reports the link itself, not its target), clearing
// immutable flags and restoring owner rwx on dir first.
func listSubdirsFixing(dir string) ([]string, error) {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return nil, err
	}
	clearImmutable(dir)
	_ = os.Chmod(dir, info.Mode().Perm()|0o700)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var subs []string
	for _, e := range entries {
		if e.IsDir() {
			subs = append(subs, e.Name())
		}
	}
	return subs, nil
}

func removeAllFixing(path string, tick func()) error {
	// Fast path: files, symlinks, and empty directories unlink directly.
	if err := os.Remove(path); err == nil {
		tick()
		return nil
	}

	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		// A non-directory that refused to unlink: an immutable flag on the
		// file itself (EPERM) is the fixable case — parent-permission issues
		// were handled before recursing down here.
		clearImmutable(path)
		if err := os.Remove(path); err != nil {
			return err
		}
		tick()
		return nil
	}

	// A populated directory: make sure we can list it and unlink its
	// children, then recurse.
	clearImmutable(path)
	_ = os.Chmod(path, info.Mode().Perm()|0o700)
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := removeAllFixing(filepath.Join(path, e.Name()), tick); err != nil {
			return err
		}
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	tick()
	return nil
}
