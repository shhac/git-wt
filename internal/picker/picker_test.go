package picker

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// Special keys we use must come from typed messages, not Runes.
var (
	keyUp    = tea.KeyMsg{Type: tea.KeyUp}
	keyDown  = tea.KeyMsg{Type: tea.KeyDown}
	keyEnter = tea.KeyMsg{Type: tea.KeyEnter}
	keyEsc   = tea.KeyMsg{Type: tea.KeyEsc}
	keyCtrlC = tea.KeyMsg{Type: tea.KeyCtrlC}
	keyHome  = tea.KeyMsg{Type: tea.KeyHome}
	keyEnd   = tea.KeyMsg{Type: tea.KeyEnd}
	keySpace = tea.KeyMsg{Type: tea.KeySpace}
)

// step delivers one key press, returning the resulting model and whether
// the model wants to quit (any tea.Cmd matches tea.Quit by name).
func step[T tea.Model](m T, k tea.KeyMsg) (T, bool) {
	out, cmd := m.Update(k)
	return out.(T), isQuit(cmd)
}

func isQuit(c tea.Cmd) bool {
	if c == nil {
		return false
	}
	// tea.Quit is the only cmd we ever return; calling it should be
	// equivalent. We can't compare functions, but we can match on result.
	msg := c()
	_, ok := msg.(tea.QuitMsg)
	return ok
}

// ----- moveCursor -----

func TestMoveCursor(t *testing.T) {
	cases := []struct {
		name           string
		cursor, n      int
		key            string
		want           int
	}{
		{"up from middle", 1, 3, "up", 0},
		{"k from middle (vim)", 1, 3, "k", 0},
		{"up at top clamps", 0, 3, "up", 0},
		{"down from middle", 1, 3, "down", 2},
		{"j from middle (vim)", 1, 3, "j", 2},
		{"down at bottom clamps", 2, 3, "down", 2},
		{"home from middle", 1, 3, "home", 0},
		{"g from middle (vim)", 1, 3, "g", 0},
		{"end from middle", 1, 3, "end", 2},
		{"G from middle (vim)", 1, 3, "G", 2},
		{"unknown key leaves cursor", 1, 3, "x", 1},
		{"empty list returns -1 on end", 0, 0, "end", -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := moveCursor(c.cursor, c.n, c.key)
			if got != c.want {
				t.Errorf("moveCursor(%d, %d, %q) = %d, want %d", c.cursor, c.n, c.key, got, c.want)
			}
		})
	}
}

// ----- selectModel -----

func TestSelectModel_DownUpNavigation(t *testing.T) {
	m := selectModel{rows: []Row{{Display: "a"}, {Display: "b"}, {Display: "c"}}}
	m, _ = step(m, keyDown)
	if m.cursor != 1 {
		t.Errorf("after Down, cursor=%d, want 1", m.cursor)
	}
	m, _ = step(m, keyDown)
	m, _ = step(m, keyDown) // bound to last
	if m.cursor != 2 {
		t.Errorf("cursor stuck at %d, want 2 (clamped to end)", m.cursor)
	}
	m, _ = step(m, keyUp)
	if m.cursor != 1 {
		t.Errorf("after Up, cursor=%d, want 1", m.cursor)
	}
}

func TestSelectModel_HomeEnd(t *testing.T) {
	m := selectModel{rows: []Row{{Display: "a"}, {Display: "b"}, {Display: "c"}}, cursor: 1}
	m, _ = step(m, keyEnd)
	if m.cursor != 2 {
		t.Errorf("End → cursor=%d, want 2", m.cursor)
	}
	m, _ = step(m, keyHome)
	if m.cursor != 0 {
		t.Errorf("Home → cursor=%d, want 0", m.cursor)
	}
}

func TestSelectModel_EnterQuitsWithoutCancel(t *testing.T) {
	m := selectModel{rows: []Row{{Display: "a"}, {Display: "b"}}, cursor: 1}
	m, quit := step(m, keyEnter)
	if !quit {
		t.Errorf("Enter should issue tea.Quit")
	}
	if m.cancelled {
		t.Errorf("Enter should NOT mark cancelled")
	}
}

func TestSelectModel_EscCancels(t *testing.T) {
	m := selectModel{rows: []Row{{Display: "a"}}}
	m, quit := step(m, keyEsc)
	if !quit {
		t.Errorf("Esc should issue tea.Quit")
	}
	if !m.cancelled {
		t.Errorf("Esc should mark cancelled")
	}
}

func TestSelectModel_CtrlCCancels(t *testing.T) {
	m := selectModel{rows: []Row{{Display: "a"}}}
	m, quit := step(m, keyCtrlC)
	if !quit {
		t.Errorf("Ctrl-C should issue tea.Quit")
	}
	if !m.cancelled {
		t.Errorf("Ctrl-C should mark cancelled (NOT select)")
	}
}

func TestSelectModel_QCancels(t *testing.T) {
	m := selectModel{rows: []Row{{Display: "a"}}}
	m, _ = step(m, key("q"))
	if !m.cancelled {
		t.Errorf("q should mark cancelled")
	}
}

func TestSelectModel_View_ShowsCursorBolded(t *testing.T) {
	m := selectModel{rows: []Row{{Display: "first"}, {Display: "second"}}, cursor: 1}
	out := m.View()
	// We can't easily assert ANSI bold codes are present, but the cursor
	// row should have the "> " prefix and the other one "  ".
	if !contains(out, "> second") {
		t.Errorf("cursor row should show `> second`, got:\n%s", out)
	}
	if !contains(out, "  first") {
		t.Errorf("non-cursor row should show `  first`, got:\n%s", out)
	}
}

// ----- multiModel -----

func TestMultiModel_SpaceToggles(t *testing.T) {
	m := multiModel{rows: []Row{{Display: "a"}, {Display: "b"}}, selected: map[int]bool{}}
	m, _ = step(m, keySpace)
	if !m.selected[0] {
		t.Errorf("space at cursor 0 should toggle on")
	}
	m, _ = step(m, keySpace)
	if m.selected[0] {
		t.Errorf("space again should toggle off")
	}
}

func TestMultiModel_AToggleAll(t *testing.T) {
	m := multiModel{rows: []Row{{Display: "a"}, {Display: "b"}, {Display: "c"}}, selected: map[int]bool{}}
	m, _ = step(m, key("a"))
	for i := range m.rows {
		if !m.selected[i] {
			t.Errorf("'a' should select all; row %d not selected", i)
		}
	}
	m, _ = step(m, key("a"))
	for i := range m.rows {
		if m.selected[i] {
			t.Errorf("second 'a' should clear all; row %d still selected", i)
		}
	}
}

func TestMultiModel_EnterReturnsSelections(t *testing.T) {
	m := multiModel{
		rows:     []Row{{Value: "a"}, {Value: "b"}, {Value: "c"}},
		selected: map[int]bool{0: true, 2: true},
	}
	m, quit := step(m, keyEnter)
	if !quit {
		t.Errorf("Enter should quit")
	}
	if m.cancelled {
		t.Errorf("Enter should NOT cancel")
	}
}

func TestMultiModel_EscCancels(t *testing.T) {
	m := multiModel{rows: []Row{{Display: "a"}}, selected: map[int]bool{}}
	m, quit := step(m, keyEsc)
	if !quit || !m.cancelled {
		t.Errorf("Esc should quit + cancel; quit=%v cancelled=%v", quit, m.cancelled)
	}
}

func TestMultiModel_CtrlCCancels(t *testing.T) {
	m := multiModel{rows: []Row{{Display: "a"}}, selected: map[int]bool{}}
	m, quit := step(m, keyCtrlC)
	if !quit || !m.cancelled {
		t.Errorf("Ctrl-C should quit + cancel; quit=%v cancelled=%v", quit, m.cancelled)
	}
}

func TestMultiModel_View_ShowsSelectionMarkers(t *testing.T) {
	m := multiModel{
		rows:     []Row{{Display: "a"}, {Display: "b"}},
		cursor:   0,
		selected: map[int]bool{0: true},
	}
	out := m.View()
	if !contains(out, "[*] a") {
		t.Errorf("selected row should show `[*]`, got:\n%s", out)
	}
	if !contains(out, "[ ] b") {
		t.Errorf("unselected row should show `[ ]`, got:\n%s", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
