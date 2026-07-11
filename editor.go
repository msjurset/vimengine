package vimengine

// Editor represents the text field component that VimEngine will control.
// Indices are in rune-offsets, not bytes, to properly support Unicode.
type Editor interface {
	// Text returns the entire content of the editor as a slice of runes.
	Text() []rune

	// SetText sets the entire content of the editor.
	SetText(text []rune)

	// SelectedRange returns the current selection in rune indices.
	// If start == end, it represents the cursor position.
	SelectedRange() (start, end int)

	// SetSelectedRange sets the selection/cursor using rune indices.
	SetSelectedRange(start, end int)

	// Replace replaces text in the given range with the new text.
	Replace(start, end int, text []rune)

	// Undo requests the editor to undo the last change.
	Undo()

	// Redo requests the editor to redo the last undone change.
	Redo()

	// VisualLineLocation returns the character offset of a visual line jump (optional).
	// If not supported, it should return -1 to fall back to logical lines.
	VisualLineLocation(from int, lines int) int
}
