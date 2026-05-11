package debug

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// withBuffer runs fn with Enabled=true and Out pointed at a fresh buffer,
// returning its contents. It restores globals afterwards.
func withBuffer(t *testing.T, fn func()) string {
	t.Helper()
	prevEnabled := Enabled
	prevOut := Out
	t.Cleanup(func() {
		Enabled = prevEnabled
		Out = prevOut
	})
	var buf bytes.Buffer
	Enabled = true
	Out = &buf
	fn()
	return buf.String()
}

func TestOp_disabledIsNoOp(t *testing.T) {
	prevEnabled := Enabled
	prevOut := Out
	t.Cleanup(func() {
		Enabled = prevEnabled
		Out = prevOut
	})
	var buf bytes.Buffer
	Enabled = false
	Out = &buf

	end := Op("git", []string{"status"})
	end(nil)

	if buf.Len() != 0 {
		t.Fatalf("disabled Op wrote to buffer: %q", buf.String())
	}
}

func TestOp_successPair(t *testing.T) {
	out := withBuffer(t, func() {
		end := Op("git", []string{"status"})
		end(nil)
	})

	lines := nonEmptyLines(out)
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d: %q", len(lines), out)
	}
	startID := parseOpID(t, lines[0])
	doneID := parseOpID(t, lines[1])
	if startID != doneID {
		t.Errorf("start/done ids differ: %d vs %d", startID, doneID)
	}
	if !strings.Contains(lines[0], "start") {
		t.Errorf("first line missing 'start': %q", lines[0])
	}
	if !strings.Contains(lines[1], "done in ") {
		t.Errorf("second line missing 'done in': %q", lines[1])
	}
	if !strings.Contains(lines[0], "git status") {
		t.Errorf("first line missing detail: %q", lines[0])
	}
}

func TestOp_errorPair(t *testing.T) {
	out := withBuffer(t, func() {
		end := Op("git", []string{"worktree", "remove", "/x"})
		end(errors.New("worktree contains modified files"))
	})

	lines := nonEmptyLines(out)
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[1], "failed in ") {
		t.Errorf("second line missing 'failed in': %q", lines[1])
	}
	if !strings.Contains(lines[1], "worktree contains modified files") {
		t.Errorf("error message missing from line: %q", lines[1])
	}
}

func TestOp_uniqueIDsAcrossCalls(t *testing.T) {
	out := withBuffer(t, func() {
		a := Op("a")
		b := Op("b")
		a(nil)
		b(nil)
	})

	lines := nonEmptyLines(out)
	if len(lines) != 4 {
		t.Fatalf("want 4 lines, got %d: %q", len(lines), out)
	}
	idA := parseOpID(t, lines[0])
	idB := parseOpID(t, lines[1])
	if idA == idB {
		t.Errorf("expected distinct ids, both = %d", idA)
	}
	// the start/done lines should still pair up
	if parseOpID(t, lines[2]) != idA {
		t.Errorf("a's done id mismatch")
	}
	if parseOpID(t, lines[3]) != idB {
		t.Errorf("b's done id mismatch")
	}
}

func TestOp_concurrentLinesNotInterleaved(t *testing.T) {
	out := withBuffer(t, func() {
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				end := Op("worker", fmt.Sprintf("n=%d", i))
				end(nil)
			}(i)
		}
		wg.Wait()
	})

	// Each line should be a complete debug line — no partial writes.
	linePattern := regexp.MustCompile(`(?m)^\[[^\]]+\] op#\d+ .* (start|done in .+|failed in .+)$`)
	for _, line := range nonEmptyLines(out) {
		// strip any ANSI from ui.Dim
		stripped := stripANSI(line)
		if !linePattern.MatchString(stripped) {
			t.Errorf("malformed line: %q", stripped)
		}
	}
}

func TestOp_multilineErrorCollapsed(t *testing.T) {
	out := withBuffer(t, func() {
		end := Op("op")
		end(errors.New("first\nsecond\nthird"))
	})
	if strings.Count(out, "\n") != 2 { // start + done, each terminated
		t.Errorf("expected exactly 2 newlines (one per line), got %q", out)
	}
	if !strings.Contains(out, "first | second | third") {
		t.Errorf("multiline error not collapsed: %q", out)
	}
}

var (
	opIDPattern = regexp.MustCompile(`op#(\d+)`)
	ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

func parseOpID(t *testing.T, line string) uint64 {
	t.Helper()
	m := opIDPattern.FindStringSubmatch(stripANSI(line))
	if m == nil {
		t.Fatalf("no op id in line: %q", line)
	}
	var id uint64
	fmt.Sscanf(m[1], "%d", &id)
	return id
}

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
