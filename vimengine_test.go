package vimengine

import (
	"testing"
)

type mockEditor struct {
	text  []rune
	start int
	end   int
}

func (m *mockEditor) Text() []rune { return m.text }
func (m *mockEditor) SetText(t []rune) { m.text = t }
func (m *mockEditor) SelectedRange() (int, int) { return m.start, m.end }
func (m *mockEditor) SetSelectedRange(s, e int) { m.start = s; m.end = e }
func (m *mockEditor) Replace(s, e int, text []rune) {
	newText := append([]rune{}, m.text[:s]...)
	newText = append(newText, text...)
	newText = append(newText, m.text[e:]...)
	m.text = newText
}
func (m *mockEditor) Undo() {}
func (m *mockEditor) Redo() {}
func (m *mockEditor) VisualLineLocation(from, lines int) int { return -1 }

func TestEngine_NormalMode_Movement(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		startPos int
		keys     []rune
		wantPos  int
	}{
		{"move right", "hello", 0, []rune{'l'}, 1},
		{"move right at end", "hello", 5, []rune{'l'}, 5},
		{"move left", "hello", 3, []rune{'h'}, 2},
		{"move left at start", "hello", 0, []rune{'h'}, 0},
		{"move word", "hello world", 0, []rune{'w'}, 6},
		{"move to end", "hello", 0, []rune{'$'}, 5},
		{"move to start", "hello", 5, []rune{'0'}, 0},
		{"move down", "hello\nworld", 0, []rune{'j'}, 6},
		{"move up", "hello\nworld", 6, []rune{'k'}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			ed := &mockEditor{text: []rune(tt.initial), start: tt.startPos, end: tt.startPos}
			
			for _, k := range tt.keys {
				e.HandleKey(Key{Char: k}, ed)
			}
			
			if ed.start != tt.wantPos {
				t.Errorf("got pos %d, want %d", ed.start, tt.wantPos)
			}
		})
	}
}

func TestEngine_Modes(t *testing.T) {
	e := New()
	ed := &mockEditor{text: []rune("hello"), start: 0, end: 0}

	// Normal to Insert
	e.HandleKey(Key{Char: 'i'}, ed)
	if e.Submode != Insert {
		t.Errorf("expected Insert mode, got %v", e.Submode)
	}

	// Insert back to Normal
	e.HandleKey(Key{KeyCode: KeyEscape}, ed)
	if e.Submode != Normal {
		t.Errorf("expected Normal mode, got %v", e.Submode)
	}

	// Normal to Command
	e.HandleKey(Key{Char: ':'}, ed)
	if e.Submode != Command {
		t.Errorf("expected Command mode, got %v", e.Submode)
	}

	// Type in command
	e.HandleKey(Key{Char: 'w'}, ed)
	if e.commandBuffer != "w" {
		t.Errorf("expected command buffer 'w', got %s", e.commandBuffer)
	}

	// Enter command
	submitted := false
	e.OnSubmit = func() { submitted = true }
	e.HandleKey(Key{KeyCode: KeyEnter}, ed)
	if !submitted {
		t.Errorf("expected OnSubmit to be called")
	}
	if e.Submode != Normal {
		t.Errorf("expected return to Normal mode")
	}
}

func TestEngine_Operations(t *testing.T) {
	e := New()
	ed := &mockEditor{text: []rune("hello world"), start: 0, end: 0}

	// Delete character
	e.HandleKey(Key{Char: 'x'}, ed)
	if string(ed.text) != "ello world" {
		t.Errorf("expected 'ello world', got '%s'", string(ed.text))
	}

	// Paste character
	e.HandleKey(Key{Char: 'p'}, ed)
	if string(ed.text) != "ehllo world" {
		t.Errorf("expected 'ehllo world', got '%s'", string(ed.text))
	}

	// Visual delete
	ed.SetText([]rune("hello world"))
	ed.SetSelectedRange(0, 0)
	e.HandleKey(Key{Char: 'v'}, ed)
	e.HandleKey(Key{Char: 'l'}, ed)
	e.HandleKey(Key{Char: 'l'}, ed)
	e.HandleKey(Key{Char: 'd'}, ed)
	
	if string(ed.text) != "llo world" {
		t.Errorf("expected 'llo world', got '%s'", string(ed.text))
	}
}
