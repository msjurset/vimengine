# vimengine

A portable, modular mini-vim component for Go TUIs.

This package provides Vim-style modal editing logic that you can plug into any text editor widget in your Go TUI framework (e.g., `bubbletea`, `tview`, etc.). It isolates the logic of vim keys, buffers, submodes, and operations so your TUI only has to implement a thin `Editor` interface.

## Integration Example

Implement the `Editor` interface on your TUI text field wrapper. Below is a conceptual example of how you might integrate this within a standard event loop (like Bubble Tea):

```go
package main

import (
	"fmt"
	"github.com/msjurset/vimengine"
)

// 1. Wrap your underlying text component
type MyTextEditor struct {
	text   []rune
	cursor int
}

// 2. Implement the Editor interface
func (m *MyTextEditor) Text() []rune { return m.text }
func (m *MyTextEditor) SetText(t []rune) { m.text = t }
func (m *MyTextEditor) SelectedRange() (int, int) { return m.cursor, m.cursor }
func (m *MyTextEditor) SetSelectedRange(s, e int) { m.cursor = s }
func (m *MyTextEditor) Replace(s, e int, text []rune) {
	newText := append([]rune{}, m.text[:s]...)
	newText = append(newText, text...)
	newText = append(newText, m.text[e:]...)
	m.text = newText
}
func (m *MyTextEditor) Undo() { /* implement undo logic */ }
func (m *MyTextEditor) Redo() { /* implement redo logic */ }
func (m *MyTextEditor) VisualLineLocation(from, lines int) int { return -1 }

// 3. Use in your TUI application
func main() {
	editor := &MyTextEditor{text: []rune("Hello World")}
	engine := vimengine.New()

	// Set up callbacks
	engine.OnSubmit = func() { fmt.Println("File saved!") }
	engine.OnExit = func() { fmt.Println("Exiting editor!") }
	engine.OnSubmodeChanged = func() { 
		fmt.Printf("\nMode changed to: %s\n", engine.Badge) 
	}
	engine.OnCommandBufferChanged = func(cmd string) {
		fmt.Printf("\nCommand line: :%s\n", cmd)
	}

	// Example: Translating a UI framework's keypress into vimengine.Key
	// In a real app, this happens in your Update() loop.
	keypresses := []vimengine.Key{
		{Char: 'c'}, {Char: 'i'}, {Char: 'w'}, // ciw (Change Inner Word)
		{Char: 'G'}, {Char: 'o'},              // Type 'Go'
		{KeyCode: vimengine.KeyEscape},        // Esc
	}

	for _, k := range keypresses {
		handled := engine.HandleKey(k, editor)
		if !handled {
			// Engine didn't consume it (e.g. typing text in Insert mode).
			// You pass this character to your native text component:
			if k.Char != 0 {
				editor.Replace(editor.cursor, editor.cursor, []rune{k.Char})
				editor.cursor++
			}
		}
	}
}
```

## Features Supported

**Movement**
`h j k l`, `w W b B e E ge`, `0 ^ $`, `gg G`, `NG`, `{ } %`, `f<x> F<x> t<x> T<x>`

**Operators & Mutations**
`d`, `y`, `c`, `x D C J s ~`, `p P`, `<`, `>`, `=`, `gU gu g~`
Supports multi-key counts (e.g. `2d3w`, `5yy`).

**Text Objects**
Use with operators (e.g. `yip`, `diw`, `ci"`).
Supported objects:
- `w`, `W` (inner/around word/WORD)
- `p` (inner/around paragraph)
- `(`, `)`, `b` (inner/around parentheses)
- `{`, `}`, `B` (inner/around curly brackets)
- `[`, `]` (inner/around square brackets)
- `"`, `'`, ``` ` ``` (inner/around quotes)

**Modes**
- Normal, Insert, Replace (`R`), Visual (`v`), VisualLine (`V`), Command (`:`)

**Command Line**
- `:w`, `:q`, `:wq`

**Undo / Redo**
- `u`, `Ctrl-r` (Delegated to your Editor implementation)

## Keeping Up to Date

Because `vimengine` is designed as an independent, loosely-coupled component, we recommend adding a dedicated target to your project's `Makefile` (or equivalent build orchestration tool) to easily pull the latest vim features.

For example:

```makefile
update-vim:
	@echo "Updating vimengine to the latest version..."
	go get github.com/msjurset/vimengine@latest
	go mod tidy
	@echo "Smoke test vim mode (ctrl+o, i, Esc, dd), then commit."
```

## Licensing

MIT License
