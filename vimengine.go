package vimengine

import (
	"strings"
	"unicode"
)

// Submode represents the current vim mode.
type Submode int

const (
	Normal Submode = iota
	Insert
	Command
	Visual
	VisualLine
	Replace
)

// State for Normal mode parsing
type ParserState int

const (
	StateNormal ParserState = iota
	StateOperatorPending
	StateFindPending
	StateTextObjectPending
	StateReplacePending
)

// Engine is a portable mini-vim component for Go TUIs.
type Engine struct {
	Submode Submode
	Badge   string

	OnExit                 func()
	OnSubmit               func()
	OnSubmodeChanged       func()
	OnCommandBufferChanged func(string)

	commandBuffer string
	register      []rune

	// Parser state
	parserState ParserState
	count       int
	hasCount    bool
	opCount     int // count provided before operator (e.g. 2d3w -> opCount=2, count=3)
	operator    string
	pendingFind rune // f, F, t, T
	
	// repeat
	lastCommand string
	
	// search
	searchBuffer string
}

// New creates a new VimEngine.
func New() *Engine {
	return &Engine{
		Submode: Normal,
		Badge:   "-- NORMAL --",
		parserState: StateNormal,
	}
}

// setSubmode updates the current mode and fires callbacks.
func (e *Engine) setSubmode(m Submode) {
	if e.Submode == m {
		return
	}
	e.Submode = m
	e.resetParser()
	switch m {
	case Normal:
		e.Badge = "-- NORMAL --"
	case Insert:
		e.Badge = "-- INSERT --"
	case Command:
		e.Badge = ":" + e.commandBuffer
	case Visual:
		e.Badge = "-- VISUAL --"
	case VisualLine:
		e.Badge = "-- VISUAL LINE --"
	case Replace:
		e.Badge = "-- REPLACE --"
	}
	if e.OnSubmodeChanged != nil {
		e.OnSubmodeChanged()
	}
}

func (e *Engine) resetParser() {
	e.parserState = StateNormal
	e.count = 0
	e.hasCount = false
	e.opCount = 0
	e.operator = ""
	e.pendingFind = 0
}

func (e *Engine) totalCount() int {
    c1 := e.count
    if c1 == 0 {
        c1 = 1
    }
    c2 := e.opCount
    if c2 == 0 {
        c2 = 1
    }
    return c1 * c2
}

// HandleKey processes a key event. Returns true if the key was handled by the engine.
func (e *Engine) HandleKey(key Key, editor Editor) bool {
	switch e.Submode {
	case Normal:
		return e.handleNormalMode(key, editor)
	case Insert:
		return e.handleInsertMode(key, editor)
	case Command:
		return e.handleCommandMode(key, editor)
	case Visual, VisualLine:
		return e.handleVisualMode(key, editor)
	case Replace:
		return e.handleReplaceMode(key, editor)
	}
	return false
}

func (e *Engine) handleNormalMode(key Key, editor Editor) bool {
	if key.KeyCode == KeyEscape {
		e.resetParser()
		return true
	}

	start, _ := editor.SelectedRange()
	text := editor.Text()

	// Handle pending find character
	if e.parserState == StateFindPending {
		if key.Char != 0 {
			newPos := e.executeFind(text, start, e.pendingFind, key.Char, e.totalCount())
			if e.operator != "" {
				// apply operator from start to newPos
				rng := Range{Start: start, End: newPos, LineWise: false}
				if newPos < start {
					rng.Start, rng.End = newPos, start
				} else if e.pendingFind == 'f' || e.pendingFind == 't' {
				    rng.End++ // inclusive for f
				}
				e.applyOperator(editor, e.operator, rng)
			} else {
				editor.SetSelectedRange(newPos, newPos)
			}
			e.resetParser()
		}
		return true
	}

	// Handle pending text object
	if e.parserState == StateTextObjectPending {
		if key.Char != 0 {
			// Wait, operator was 'd' and we appended 'i', so e.operator is "di".
			// Let's refine this:
			// If operator is "di", then obj = 'i' + key.Char
			opType := e.operator[:len(e.operator)-1]
			objMod := e.operator[len(e.operator)-1]
			
			obj := string(objMod) + string(key.Char)
			
			rng, ok := e.textObject(text, start, obj)
			if ok {
				e.applyOperator(editor, opType, rng)
			}
			e.resetParser()
		}
		return true
	}

	// Replace single char pending
	if e.parserState == StateReplacePending {
	    if key.Char != 0 {
	        c := e.totalCount()
	        endX := start + c
	        if endX > len(text) { endX = len(text) }
	        for i := start; i < endX; i++ {
	            if text[i] == '\n' { endX = i; break }
	        }
	        repl := make([]rune, endX-start)
	        for i := range repl { repl[i] = key.Char }
	        editor.Replace(start, endX, repl)
	        editor.SetSelectedRange(endX-1, endX-1)
	    }
	    e.resetParser()
	    return true
	}

	// Count accumulation
	if unicode.IsDigit(key.Char) {
		val := int(key.Char - '0')
		if val == 0 && !e.hasCount {
			// '0' as motion
			if e.operator != "" {
				newPos := e.findLineStart(text, start)
				e.applyOperator(editor, e.operator, Range{Start: start, End: newPos, LineWise: false})
				e.resetParser()
			} else {
				editor.SetSelectedRange(e.findLineStart(text, start), e.findLineStart(text, start))
			}
			return true
		}
		e.count = e.count*10 + val
		e.hasCount = true
		return true
	}

	// Operators
	isOp := false
	opStr := string(key.Char)
	switch key.Char {
	case 'd', 'y', 'c', '>', '<', '=':
		isOp = true
	case 'g':
		// pending multi-key operator like gU, gu, g~
		if e.operator == "g" {
			// it's actually gg
			opStr = "gg"
			isOp = false
		} else {
			e.operator = "g"
			return true
		}
	}
	
	if e.operator == "g" {
		switch key.Char {
		case 'U', 'u', '~':
			e.operator = "g" + string(key.Char)
			return true
		}
	}

	if isOp {
		if e.parserState == StateOperatorPending {
			if e.operator == opStr { // dd, yy, cc, >>
				rng := Range{Start: start, End: start, LineWise: true}
				
				// extend rng by count lines
				count := e.totalCount()
				if count > 1 {
				    for i := 1; i < count; i++ {
				        rng.End = e.moveLines(text, rng.End, 1)
				    }
				}
				
				e.applyOperator(editor, e.operator, rng)
			}
			e.resetParser()
			return true
		}
		e.operator = opStr
		e.opCount = e.count
		e.count = 0
		e.hasCount = false
		e.parserState = StateOperatorPending
		return true
	}

	// Text Object modifiers inside operator pending
	if e.parserState == StateOperatorPending && (key.Char == 'i' || key.Char == 'a') {
		e.operator = e.operator + string(key.Char)
		e.parserState = StateTextObjectPending
		return true
	}
	
	// Pending find
	if key.Char == 'f' || key.Char == 'F' || key.Char == 't' || key.Char == 'T' {
		e.pendingFind = key.Char
		e.parserState = StateFindPending
		return true
	}

	// Single key non-motion commands
	if e.parserState == StateNormal {
		switch key.Char {
		case ':':
			e.commandBuffer = ""
			e.setSubmode(Command)
			if e.OnCommandBufferChanged != nil {
				e.OnCommandBufferChanged(e.commandBuffer)
			}
			return true
		case 'i':
			e.setSubmode(Insert)
			return true
		case 'a':
			if start < len(text) && text[start] != '\n' {
				editor.SetSelectedRange(start+1, start+1)
			}
			e.setSubmode(Insert)
			return true
		case 'I':
			start = e.findLineFirstNonBlank(text, start)
			editor.SetSelectedRange(start, start)
			e.setSubmode(Insert)
			return true
		case 'A':
			start = e.findLineEnd(text, start)
			editor.SetSelectedRange(start, start)
			e.setSubmode(Insert)
			return true
		case 'o':
			end := e.findLineEnd(text, start)
			editor.Replace(end, end, []rune{'\n'})
			editor.SetSelectedRange(end+1, end+1)
			e.setSubmode(Insert)
			return true
		case 'O':
			lineStart := e.findLineStart(text, start)
			editor.Replace(lineStart, lineStart, []rune{'\n'})
			editor.SetSelectedRange(lineStart, lineStart)
			e.setSubmode(Insert)
			return true
		case 'R':
			e.setSubmode(Replace)
			return true
		case 'v':
			e.setSubmode(Visual)
			return true
		case 'V':
			e.setSubmode(VisualLine)
			return true
		case 'u':
			editor.Undo()
			return true
		case 'r':
			if key.Modifier&ModCtrl != 0 {
				editor.Redo()
				return true
			}
			// replace single char
			e.parserState = StateReplacePending
			return true
		case 'p':
			if len(e.register) > 0 {
				insertPos := start
				if e.isRegisterLinewise(e.register) {
				    insertPos = e.findLineEnd(text, start)
				    if insertPos < len(text) {
				        insertPos++
				    } else {
				        // append newline
				        editor.Replace(insertPos, insertPos, []rune{'\n'})
				        insertPos++
				        text = editor.Text()
				    }
				    // repeat for count
				    c := e.totalCount()
				    for i := 0; i < c; i++ {
				        editor.Replace(insertPos, insertPos, e.register)
				    }
				    editor.SetSelectedRange(insertPos, insertPos)
				} else {
				    if start < len(text) && text[start] != '\n' {
					    insertPos++
				    }
				    c := e.totalCount()
				    for i := 0; i < c; i++ {
				        editor.Replace(insertPos, insertPos, e.register)
				    }
				    editor.SetSelectedRange(insertPos+len(e.register)*c-1, insertPos+len(e.register)*c-1)
				}
			}
			e.resetParser()
			return true
		case 'P':
			if len(e.register) > 0 {
				insertPos := start
				if e.isRegisterLinewise(e.register) {
				    insertPos = e.findLineStart(text, start)
				    c := e.totalCount()
				    for i := 0; i < c; i++ {
				        editor.Replace(insertPos, insertPos, e.register)
				    }
				    editor.SetSelectedRange(insertPos, insertPos)
				} else {
				    c := e.totalCount()
				    for i := 0; i < c; i++ {
				        editor.Replace(insertPos, insertPos, e.register)
				    }
				    editor.SetSelectedRange(insertPos+len(e.register)*c-1, insertPos+len(e.register)*c-1)
				}
			}
			e.resetParser()
			return true
		case 'x':
			if start < len(text) && text[start] != '\n' {
			    c := e.totalCount()
			    endX := start + c
			    if endX > len(text) { endX = len(text) }
			    // don't delete past newline
			    for i := start; i < endX; i++ {
			        if text[i] == '\n' {
			            endX = i
			            break
			        }
			    }
				e.register = text[start : endX]
				editor.Replace(start, endX, nil)
			}
			e.resetParser()
			return true
		case 'D':
		    end := e.findLineEnd(text, start)
		    e.register = text[start : end]
		    editor.Replace(start, end, nil)
		    e.resetParser()
		    return true
		case 'C':
		    end := e.findLineEnd(text, start)
		    e.register = text[start : end]
		    editor.Replace(start, end, nil)
		    e.setSubmode(Insert)
		    return true
		case 's':
		    c := e.totalCount()
		    endX := start + c
		    if endX > len(text) { endX = len(text) }
		    for i := start; i < endX; i++ {
		        if text[i] == '\n' { endX = i; break }
		    }
		    e.register = text[start : endX]
		    editor.Replace(start, endX, nil)
		    e.setSubmode(Insert)
		    return true
		case '~':
		    c := e.totalCount()
		    endX := start + c
		    if endX > len(text) { endX = len(text) }
		    for i := start; i < endX; i++ {
		        if text[i] == '\n' { endX = i; break }
		    }
		    runes := text[start:endX]
		    for i, r := range runes {
		        if unicode.IsUpper(r) { runes[i] = unicode.ToLower(r) } else { runes[i] = unicode.ToUpper(r) }
		    }
		    editor.Replace(start, endX, runes)
		    if endX < len(text) && text[endX] != '\n' { editor.SetSelectedRange(endX, endX) } else { editor.SetSelectedRange(endX-1, endX-1) }
		    e.resetParser()
		    return true
		case 'J':
		    c := e.totalCount()
		    for i := 0; i < c; i++ {
		        end := e.findLineEnd(text, start)
		        if end < len(text) && text[end] == '\n' {
		            // replace \n and leading spaces with single space
		            nextStart := end + 1
		            for nextStart < len(text) && isBlank(text[nextStart]) {
		                nextStart++
		            }
		            editor.Replace(end, nextStart, []rune{' '})
		            text = editor.Text()
		            editor.SetSelectedRange(end, end)
		        }
		    }
		    e.resetParser()
		    return true
		}
	}
	

	// Motions
	motionStr := ""
	if opStr == "gg" {
		motionStr = "gg"
	} else if key.Char != 0 {
		motionStr = string(key.Char)
	}
	if motionStr == "" && key.KeyCode != KeyNone {
	    switch key.KeyCode {
	    case KeyArrowLeft: motionStr = "h"
	    case KeyArrowRight: motionStr = "l"
	    case KeyArrowUp: motionStr = "k"
	    case KeyArrowDown: motionStr = "j"
	    }
	}

	if motionStr != "" {
		newPos, isLinewise := e.moveCursor(text, start, motionStr, e.totalCount())
		
		if e.parserState == StateOperatorPending {
			rng := Range{Start: start, End: newPos, LineWise: isLinewise}
			if newPos < start {
				rng.Start, rng.End = newPos, start
			} else {
			    // exclusive by default, unless end of word etc (vim rules are complex, simplifying)
			    if motionStr == "e" || motionStr == "E" {
			        rng.End++
			    }
			}
			e.applyOperator(editor, e.operator, rng)
			e.resetParser()
			return true
		} else {
			if newPos != start {
				editor.SetSelectedRange(newPos, newPos)
			}
			e.resetParser()
			return true
		}
	}

	return false
}

func (e *Engine) executeFind(text []rune, pos int, findType, char rune, count int) int {
	dir := 1
	if findType == 'F' || findType == 'T' {
		dir = -1
	}
	
	lineStart := e.findLineStart(text, pos)
	lineEnd := e.findLineEnd(text, pos)
	
	curr := pos + dir
	found := 0
	
	for curr >= lineStart && curr < lineEnd {
		if text[curr] == char {
			found++
			if found == count {
				if findType == 't' {
					return curr - 1
				} else if findType == 'T' {
					return curr + 1
				}
				return curr
			}
		}
		curr += dir
	}
	
	return pos
}

func (e *Engine) isRegisterLinewise(r []rune) bool {
    if len(r) == 0 { return false }
    return r[len(r)-1] == '\n'
}

func (e *Engine) handleInsertMode(key Key, editor Editor) bool {
	if key.KeyCode == KeyEscape || (key.Char == 'c' && key.Modifier&ModCtrl != 0) {
		e.setSubmode(Normal)
		start, _ := editor.SelectedRange()
		if start > 0 {
			editor.SetSelectedRange(start-1, start-1)
		}
		return true
	}
	return false
}

func (e *Engine) handleCommandMode(key Key, editor Editor) bool {
	if key.KeyCode == KeyEscape || (key.Char == 'c' && key.Modifier&ModCtrl != 0) {
		e.setSubmode(Normal)
		return true
	}
	if key.KeyCode == KeyEnter {
		cmd := strings.TrimSpace(e.commandBuffer)
		e.setSubmode(Normal)
		switch cmd {
		case "q":
			if e.OnExit != nil {
				e.OnExit()
			}
		case "w":
			if e.OnSubmit != nil {
				e.OnSubmit()
			}
		case "wq", "qw":
			if e.OnSubmit != nil {
				e.OnSubmit()
			}
			if e.OnExit != nil {
				e.OnExit()
			}
		}
		return true
	}
	if key.KeyCode == KeyBackspace || key.KeyCode == KeyDelete {
		if len(e.commandBuffer) > 0 {
			e.commandBuffer = e.commandBuffer[:len(e.commandBuffer)-1]
		} else {
			e.setSubmode(Normal)
			return true
		}
	} else if key.Char != 0 {
		e.commandBuffer += string(key.Char)
	}
	e.Badge = ":" + e.commandBuffer
	if e.OnCommandBufferChanged != nil {
		e.OnCommandBufferChanged(e.commandBuffer)
	}
	return true
}

func (e *Engine) handleVisualMode(key Key, editor Editor) bool {
	if key.KeyCode == KeyEscape || (key.Char == 'c' && key.Modifier&ModCtrl != 0) {
		e.setSubmode(Normal)
		start, _ := editor.SelectedRange()
		editor.SetSelectedRange(start, start)
		return true
	}

	start, end := editor.SelectedRange()
	
	text := editor.Text()
	
	realStart, realEnd := start, end
	if realStart > realEnd {
	    realStart, realEnd = realEnd, realStart
	}

	if key.Char == 'y' {
		if realStart != realEnd {
		    if e.Submode == VisualLine {
		        s := e.findLineStart(text, realStart)
		        e2 := e.findLineEnd(text, realEnd)
		        // append newline so it pastes as linewise
		        e.register = append([]rune{}, text[s:e2]...)
		        e.register = append(e.register, '\n')
		    } else {
			    e.register = append([]rune{}, text[realStart:realEnd]...)
			}
		}
		e.setSubmode(Normal)
		editor.SetSelectedRange(realStart, realStart)
		return true
	}

	if key.Char == 'd' || key.Char == 'x' || key.Char == 'c' {
		if realStart != realEnd {
		    if e.Submode == VisualLine {
		        s := e.findLineStart(text, realStart)
		        e2 := e.findLineEnd(text, realEnd)
		        if e2 < len(text) { e2++ } // delete trailing newline
		        e.register = append([]rune{}, text[s:e2]...)
			    editor.Replace(s, e2, nil)
		    } else {
			    e.register = append([]rune{}, text[realStart:realEnd]...)
			    editor.Replace(realStart, realEnd, nil)
			}
		}
		if key.Char == 'c' {
			e.setSubmode(Insert)
		} else {
			e.setSubmode(Normal)
		}
		editor.SetSelectedRange(realStart, realStart)
		return true
	}
	
	if key.Char == '~' || key.Char == 'U' || key.Char == 'u' {
	    s := realStart
	    e2 := realEnd
	    if e.Submode == VisualLine {
	        s = e.findLineStart(text, realStart)
	        e2 = e.findLineEnd(text, realEnd)
	    }
	    runes := text[s:e2]
	    for i, r := range runes {
	        if key.Char == 'U' { runes[i] = unicode.ToUpper(r) }
	        if key.Char == 'u' { runes[i] = unicode.ToLower(r) }
	        if key.Char == '~' { 
	            if unicode.IsUpper(r) { runes[i] = unicode.ToLower(r) } else { runes[i] = unicode.ToUpper(r) }
	        }
	    }
	    editor.Replace(s, e2, runes)
	    e.setSubmode(Normal)
	    editor.SetSelectedRange(s, s)
	    return true
	}

	motionStr := ""
	if key.Char != 0 {
		motionStr = string(key.Char)
	}
	if motionStr == "" && key.KeyCode != KeyNone {
	    switch key.KeyCode {
	    case KeyArrowLeft: motionStr = "h"
	    case KeyArrowRight: motionStr = "l"
	    case KeyArrowUp: motionStr = "k"
	    case KeyArrowDown: motionStr = "j"
	    }
	}
	
	if motionStr != "" {
	    newPos, _ := e.moveCursor(text, end, motionStr, 1) // no count for simplicity in visual
	    editor.SetSelectedRange(start, newPos)
	    return true
	}

	return false
}

func (e *Engine) handleReplaceMode(key Key, editor Editor) bool {
	if key.KeyCode == KeyEscape || (key.Char == 'c' && key.Modifier&ModCtrl != 0) {
		e.setSubmode(Normal)
		start, _ := editor.SelectedRange()
		if start > 0 {
			editor.SetSelectedRange(start-1, start-1)
		}
		return true
	}

	if key.Char != 0 && key.KeyCode != KeyBackspace && key.KeyCode != KeyEnter {
		start, _ := editor.SelectedRange()
		text := editor.Text()
		if start < len(text) && text[start] != '\n' {
			editor.Replace(start, start+1, []rune{key.Char})
		} else {
			editor.Replace(start, start, []rune{key.Char})
		}
		editor.SetSelectedRange(start+1, start+1)
		return true
	}
	
	if key.KeyCode == KeyBackspace {
	    start, _ := editor.SelectedRange()
	    if start > 0 {
	        editor.SetSelectedRange(start-1, start-1)
	    }
	    return true
	}

	return false
}
