package vimengine

// Modifier flags for a key event.
type Modifier int

const (
	ModNone Modifier = 0
	ModCtrl Modifier = 1 << iota
	ModAlt
	ModShift
)

// Key is a representation of a keyboard event.
type Key struct {
	Char     rune
	KeyCode  int
	Modifier Modifier
}

// Common key codes that map to non-printable or control characters.
const (
	KeyNone = iota
	KeyEnter
	KeyEscape
	KeyBackspace
	KeyDelete
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
)
