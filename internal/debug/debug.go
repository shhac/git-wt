// Package debug emits a timeline of operations to stderr when --debug is set.
//
// Each Op() call returns a closer that records the duration and error state
// when invoked. Start and done lines share an op#N id so they can be paired
// even when ops overlap.
package debug

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shhac/git-wt/internal/ui"
)

// Enabled gates all output. Set from the --debug flag before any Op() call.
var Enabled bool

// Out is where debug lines are written. Replace in tests; defaults to stderr.
var Out io.Writer = os.Stderr

var (
	start  = time.Now()
	nextID atomic.Uint64
	mu     sync.Mutex // serializes writes so concurrent ops don't interleave lines
)

// Logf emits a one-shot debug line for stateless observations (e.g. "fell
// back to bare mode"). For measurable operations use Op. The line carries
// the elapsed-time prefix so it interleaves cleanly with Op output.
func Logf(format string, args ...any) {
	if !Enabled {
		return
	}
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s] note\t-\t-\t-\t%s", time.Since(start).Truncate(time.Millisecond), oneLine(msg))
	if !ui.Plain {
		line = ui.Dim(line)
	}
	mu.Lock()
	defer mu.Unlock()
	_, _ = fmt.Fprintln(Out, line)
}

// Op records the start of a named operation and returns a closer. Call the
// closer with the operation's error (nil on success) to record its duration.
// The detail args are rendered after the name and are usually the subprocess
// argv or a short path.
//
// When Enabled is false, Op is a near-no-op: it returns a closer that does
// nothing and skips formatting/locking.
func Op(name string, detail ...any) func(err error) {
	if !Enabled {
		return func(error) {}
	}
	id := nextID.Add(1)
	began := time.Now()
	writeLine(id, name, detail, "start", began.Sub(start), 0, nil)
	return func(err error) {
		writeLine(id, name, detail, outcome(err), time.Since(start), time.Since(began), err)
	}
}

func outcome(err error) string {
	if err != nil {
		return "failed"
	}
	return "done"
}

// writeLine emits a single tab-separated line:
//
//	[elapsed]  op#N  status  took  name  detail  [err]
//
// Non-trailing empty columns are filled with "-" so positional awk
// (`awk '{print $5}'`) stays consistent across start/done/failed rows.
// Trailing empties (e.g. `err` on a success) are omitted entirely.
// Tabs let downstream tools (`awk -F'\t'`, `cut`, `sort -k`) extract
// columns cleanly without worrying about spaces in detail args.
func writeLine(id uint64, name string, detail []any, status string, elapsed, took time.Duration, err error) {
	tookStr := ""
	if status != "start" {
		tookStr = took.Truncate(time.Millisecond).String()
	}
	errStr := ""
	if err != nil {
		errStr = oneLine(err.Error())
	}
	cols := []string{
		"[" + elapsed.Truncate(time.Millisecond).String() + "]",
		fmt.Sprintf("op#%d", id),
		status,
		tookStr,
		name,
		renderDetail(detail),
		errStr,
	}
	for len(cols) > 0 && cols[len(cols)-1] == "" {
		cols = cols[:len(cols)-1]
	}
	for i, c := range cols {
		if c == "" {
			cols[i] = "-"
		}
	}
	line := strings.Join(cols, "\t")
	if !ui.Plain {
		line = ui.Dim(line)
	}

	mu.Lock()
	defer mu.Unlock()
	_, _ = fmt.Fprintln(Out, line)
}

// renderDetail flattens detail args into a single space-separated string.
// []string is expanded element-by-element; everything else uses %v.
func renderDetail(detail []any) string {
	if len(detail) == 0 {
		return ""
	}
	parts := make([]string, 0, len(detail))
	for _, d := range detail {
		switch v := d.(type) {
		case []string:
			parts = append(parts, v...)
		case string:
			parts = append(parts, v)
		default:
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}
	return strings.Join(parts, " ")
}

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " | ")
	return s
}
