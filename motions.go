package vimengine

func (e *Engine) moveCursor(text []rune, pos int, motion string, count int) (int, bool) {
	if count < 1 {
		count = 1
	}
	
	newPos := pos
	isLinewise := false

	for i := 0; i < count; i++ {
		switch motion {
		case "h":
			if newPos > 0 && text[newPos-1] != '\n' {
				newPos--
			}
		case "l":
			if newPos < len(text) && text[newPos] != '\n' {
				newPos++
			}
		case "j":
			newPos = e.moveLines(text, newPos, 1)
			isLinewise = true
		case "k":
			newPos = e.moveLines(text, newPos, -1)
			isLinewise = true
		case "w":
			newPos = e.moveWord(text, newPos, 1, false)
		case "W":
			newPos = e.moveWord(text, newPos, 1, true)
		case "b":
			newPos = e.moveWord(text, newPos, -1, false)
		case "B":
			newPos = e.moveWord(text, newPos, -1, true)
		case "e":
			newPos = e.moveWordEnd(text, newPos, 1, false)
		case "E":
			newPos = e.moveWordEnd(text, newPos, 1, true)
		case "ge":
			newPos = e.moveWordEnd(text, newPos, -1, false)
		case "0":
			newPos = e.findLineStart(text, newPos)
		case "^":
			newPos = e.findLineFirstNonBlank(text, newPos)
		case "$":
			newPos = e.findLineEnd(text, newPos)
		case "gg":
			newPos = 0
			isLinewise = true
		case "G":
			newPos = len(text)
			isLinewise = true
		case "{":
			r, ok := e.paragraphObject(text, newPos, false)
			if ok {
				newPos = r.Start
			}
			isLinewise = true
		case "}":
			r, ok := e.paragraphObject(text, newPos, false)
			if ok {
				newPos = r.End
			}
			isLinewise = true
		case "%":
			newPos = e.findMatchingBracket(text, newPos)
		}
	}
	
	// handle line numbers like 10G
	if motion == "G" && e.hasCount {
	    // find line number (count)
	    line := 1
	    newPos = 0
	    for i := 0; i < len(text); i++ {
	        if line == e.count {
	            break
	        }
	        if text[i] == '\n' {
	            line++
	            newPos = i + 1
	        }
	    }
	    e.hasCount = false
	    e.count = 0
	    isLinewise = true
	}

	return newPos, isLinewise
}

func (e *Engine) findLineFirstNonBlank(text []rune, pos int) int {
	start := e.findLineStart(text, pos)
	end := e.findLineEnd(text, pos)
	for i := start; i < end; i++ {
		if !isBlank(text[i]) {
			return i
		}
	}
	return end
}

func (e *Engine) moveWordEnd(text []rune, pos int, dir int, bigWord bool) int {
	// simplified
	if dir > 0 {
		if pos+1 < len(text) {
			pos++
		}
		for pos < len(text) {
			if isBlank(text[pos]) || text[pos] == '\n' {
				pos++
			} else {
				break
			}
		}
		// find end of word
		for pos < len(text) {
			if pos+1 == len(text) || isBlank(text[pos+1]) || text[pos+1] == '\n' {
				return pos
			}
			pos++
		}
	} else {
		if pos > 0 {
			pos--
		}
		for pos > 0 {
			if isBlank(text[pos]) || text[pos] == '\n' {
				pos--
			} else {
				break
			}
		}
		// find start of word
		for pos > 0 {
			if pos-1 < 0 || isBlank(text[pos-1]) || text[pos-1] == '\n' {
				return pos
			}
			pos--
		}
	}
	return pos
}

func (e *Engine) findMatchingBracket(text []rune, pos int) int {
	pairs := map[rune]rune{
		'(': ')', '{': '}', '[': ']',
		')': '(', '}': '{', ']': '[',
	}
	
	// find first bracket on line if not on one
	lineEnd := e.findLineEnd(text, pos)
	curr := pos
	for curr < lineEnd {
		if _, ok := pairs[text[curr]]; ok {
			break
		}
		curr++
	}
	if curr == lineEnd {
		return pos
	}
	
	startChar := text[curr]
	endChar := pairs[startChar]
	dir := 1
	if startChar == ')' || startChar == '}' || startChar == ']' {
		dir = -1
	}
	
	depth := 0
	for i := curr; i >= 0 && i < len(text); i += dir {
		if text[i] == startChar {
			depth++
		} else if text[i] == endChar {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	
	return pos
}
