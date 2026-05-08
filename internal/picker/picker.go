// Package picker is a small bubbletea-based interactive picker tailored to
// git-wt's needs. Three primitives:
//
//   - SelectOne([]Row)         single-pick, returns the chosen value
//   - SelectMany([]Row)        multi-pick (space toggles, enter confirms)
//   - Confirm(prompt, opts)    pick one of N labeled options
//
// All three:
//   - Pass embedded ANSI in row text through verbatim (no theme override
//     strips per-column color).
//   - Treat ESC and Ctrl-C as cancel — the Run* helpers return ok=false.
//   - Use lipgloss only for the cursor row's bold styling and section
//     padding; the row text is whatever the caller built.
package picker

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Row is a single line in a single/multi picker.
type Row struct {
	Display string // pre-rendered, may include ANSI escape codes
	Value   string // returned to the caller when this row is chosen
}

// Option is a labelled choice in a Confirm prompt.
type Option[T any] struct {
	Label string
	Value T
}

// ErrEmpty is returned if a picker is asked to display zero rows/options.
var ErrEmpty = errors.New("picker: no rows to display")

var (
	cursorStyle = lipgloss.NewStyle().Bold(true)
	titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// ----------------------------------------------------------------------------
// SelectOne
// ----------------------------------------------------------------------------

// SelectOne runs the single-select picker and returns the chosen value.
// ok is false if the user cancelled (ESC, Ctrl-C, q).
func SelectOne(title string, rows []Row) (value string, ok bool, err error) {
	if len(rows) == 0 {
		return "", false, ErrEmpty
	}
	m := selectModel{title: title, rows: rows}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return "", false, err
	}
	fm := final.(selectModel)
	if fm.cancelled {
		return "", false, nil
	}
	return fm.rows[fm.cursor].Value, true, nil
}

type selectModel struct {
	title     string
	rows      []Row
	cursor    int
	cancelled bool
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "ctrl+c", "esc", "q":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		default:
			m.cursor = moveCursor(m.cursor, len(m.rows), k.String())
		}
	}
	return m, nil
}

// moveCursor applies the navigation keys (up/down/home/end and their vim
// counterparts) and returns the new cursor index, clamped to [0, n-1].
// Unrecognised keys leave the cursor unchanged. Shared by selectModel and
// multiModel.
func moveCursor(cursor, n int, key string) int {
	switch key {
	case "up", "k":
		if cursor > 0 {
			return cursor - 1
		}
	case "down", "j":
		if cursor < n-1 {
			return cursor + 1
		}
	case "home", "g":
		return 0
	case "end", "G":
		return n - 1
	}
	return cursor
}

func (m selectModel) View() string {
	var sb strings.Builder
	if m.title != "" {
		sb.WriteString(titleStyle.Render(m.title))
		sb.WriteString("\n")
	}
	for i, r := range m.rows {
		if i == m.cursor {
			sb.WriteString(cursorStyle.Render("> " + r.Display))
		} else {
			sb.WriteString("  " + r.Display)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// ----------------------------------------------------------------------------
// SelectMany
// ----------------------------------------------------------------------------

// SelectMany runs a multi-select picker. ok is false on cancel.
// Returns the values of toggled-on rows in their original order.
func SelectMany(title string, rows []Row) (values []string, ok bool, err error) {
	if len(rows) == 0 {
		return nil, false, ErrEmpty
	}
	m := multiModel{title: title, rows: rows, selected: make(map[int]bool)}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return nil, false, err
	}
	fm := final.(multiModel)
	if fm.cancelled {
		return nil, false, nil
	}
	out := make([]string, 0, len(fm.selected))
	for i, r := range fm.rows {
		if fm.selected[i] {
			out = append(out, r.Value)
		}
	}
	return out, true, nil
}

type multiModel struct {
	title     string
	rows      []Row
	cursor    int
	selected  map[int]bool
	cancelled bool
}

func (m multiModel) Init() tea.Cmd { return nil }

func (m multiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "a":
			// toggle all
			anyOff := false
			for i := range m.rows {
				if !m.selected[i] {
					anyOff = true
					break
				}
			}
			for i := range m.rows {
				m.selected[i] = anyOff
			}
		default:
			m.cursor = moveCursor(m.cursor, len(m.rows), k.String())
		}
	}
	return m, nil
}

func (m multiModel) View() string {
	var sb strings.Builder
	if m.title != "" {
		sb.WriteString(titleStyle.Render(m.title))
		sb.WriteString("\n")
	}
	for i, r := range m.rows {
		mark := "[ ] "
		if m.selected[i] {
			mark = "[*] "
		}
		line := mark + r.Display
		if i == m.cursor {
			sb.WriteString(cursorStyle.Render("> " + line))
		} else {
			sb.WriteString("  " + line)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// ----------------------------------------------------------------------------
// Confirm
// ----------------------------------------------------------------------------

// Confirm runs a one-of-N labelled picker. Useful for action choices
// (e.g. tree only / tree + branch / cancel) where each option carries a
// typed value. ok is false on cancel.
func Confirm[T any](title string, options []Option[T]) (value T, ok bool, err error) {
	var zero T
	if len(options) == 0 {
		return zero, false, ErrEmpty
	}
	rows := make([]Row, len(options))
	for i, o := range options {
		rows[i] = Row{Display: o.Label, Value: ""}
	}
	m := selectModel{title: title, rows: rows}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return zero, false, err
	}
	fm := final.(selectModel)
	if fm.cancelled {
		return zero, false, nil
	}
	return options[fm.cursor].Value, true, nil
}
