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

// Column indices into a tab-split debug line.
const (
	colElapsed = iota
	colOpID
	colStatus
	colTook
	colName
	colDetail
	colErr
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

	startCols := splitCols(t, lines[0])
	doneCols := splitCols(t, lines[1])

	if startCols[colOpID] != doneCols[colOpID] {
		t.Errorf("start/done ids differ: %q vs %q", startCols[colOpID], doneCols[colOpID])
	}
	if startCols[colStatus] != "start" {
		t.Errorf("want status=start, got %q", startCols[colStatus])
	}
	if doneCols[colStatus] != "done" {
		t.Errorf("want status=done, got %q", doneCols[colStatus])
	}
	if startCols[colTook] != "-" {
		t.Errorf("start line should have '-' took placeholder, got %q", startCols[colTook])
	}
	if doneCols[colTook] == "" || doneCols[colTook] == "-" {
		t.Errorf("done line should have a took value, got %q", doneCols[colTook])
	}
	if startCols[colName] != "git" {
		t.Errorf("want name=git, got %q", startCols[colName])
	}
	if startCols[colDetail] != "status" {
		t.Errorf("want detail=status, got %q", startCols[colDetail])
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

	failedCols := splitCols(t, lines[1])
	if failedCols[colStatus] != "failed" {
		t.Errorf("want status=failed, got %q", failedCols[colStatus])
	}
	if len(failedCols) <= colErr || failedCols[colErr] != "worktree contains modified files" {
		t.Errorf("error column missing or wrong: %v", failedCols)
	}
	if failedCols[colName] != "git" || failedCols[colDetail] != "worktree remove /x" {
		t.Errorf("name/detail wrong: name=%q detail=%q", failedCols[colName], failedCols[colDetail])
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
	idA := splitCols(t, lines[0])[colOpID]
	idB := splitCols(t, lines[1])[colOpID]
	if idA == idB {
		t.Errorf("expected distinct ids, both = %s", idA)
	}
	if splitCols(t, lines[2])[colOpID] != idA {
		t.Errorf("a's done id mismatch")
	}
	if splitCols(t, lines[3])[colOpID] != idB {
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

	for _, line := range nonEmptyLines(out) {
		cols := splitCols(t, line)
		if len(cols) < colDetail+1 {
			t.Errorf("malformed line, only %d cols: %q", len(cols), stripANSI(line))
			continue
		}
		if !strings.HasPrefix(cols[colElapsed], "[") || !strings.HasSuffix(cols[colElapsed], "]") {
			t.Errorf("elapsed not bracketed: %q", cols[colElapsed])
		}
		if !strings.HasPrefix(cols[colOpID], "op#") {
			t.Errorf("op id missing prefix: %q", cols[colOpID])
		}
		switch cols[colStatus] {
		case "start", "done", "failed":
		default:
			t.Errorf("unexpected status %q", cols[colStatus])
		}
	}
}

func TestOp_multilineErrorCollapsed(t *testing.T) {
	out := withBuffer(t, func() {
		end := Op("op")
		end(errors.New("first\nsecond\nthird"))
	})
	if strings.Count(out, "\n") != 2 {
		t.Errorf("expected exactly 2 newlines (one per line), got %q", out)
	}
	if !strings.Contains(out, "first | second | third") {
		t.Errorf("multiline error not collapsed: %q", out)
	}
}

func TestOp_columnsStableUnderDefaultAwk(t *testing.T) {
	// Default awk splits on whitespace and collapses runs — empty columns
	// would shift everything left. The "-" placeholder keeps $5 stable
	// across start/done rows in the fixed-position columns.
	out := withBuffer(t, func() {
		end := Op("git", []string{"status"})
		end(nil)
	})
	lines := nonEmptyLines(out)
	startFields := strings.Fields(stripANSI(lines[0]))
	doneFields := strings.Fields(stripANSI(lines[1]))
	if len(startFields) < 5 || len(doneFields) < 5 {
		t.Fatalf("not enough fields: start=%v done=%v", startFields, doneFields)
	}
	// columns 1..5 are fixed; default awk's $5 must be the op name on both rows.
	if startFields[4] != "git" || doneFields[4] != "git" {
		t.Errorf("default-awk $5 unstable: start=%q done=%q", startFields[4], doneFields[4])
	}
}

func TestOp_trailingEmptyErrOmitted(t *testing.T) {
	// On success, err is trailing-empty and should not appear as "-".
	out := withBuffer(t, func() {
		end := Op("git", []string{"status"})
		end(nil)
	})
	cols := splitCols(t, nonEmptyLines(out)[1])
	if len(cols) != 6 {
		t.Errorf("done line should have exactly 6 columns, got %d: %v", len(cols), cols)
	}
}

func TestOp_awkExtractsCommandColumn(t *testing.T) {
	// Verifies the contract: splitting on tab and taking the detail column
	// yields the raw command args, regardless of how many spaces they contain.
	out := withBuffer(t, func() {
		end := Op("git", []string{"worktree", "remove", "/path with spaces/x"})
		end(nil)
	})
	startCols := splitCols(t, nonEmptyLines(out)[0])
	if startCols[colDetail] != "worktree remove /path with spaces/x" {
		t.Errorf("detail column not preserved through tabs: %q", startCols[colDetail])
	}
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// splitCols strips any ANSI styling and splits a debug line on tabs.
func splitCols(t *testing.T, line string) []string {
	t.Helper()
	return strings.Split(stripANSI(line), "\t")
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
