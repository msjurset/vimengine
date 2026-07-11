package vimengine

import "unicode"

// Range represents a span of text in rune indices.
type Range struct {
	Start    int
	End      int
	LineWise bool // if true, the operation affects whole lines
}

// TextObject bounds computation
func (e *Engine) textObject(text []rune, pos int, obj string) (Range, bool) {
	if len(obj) != 2 {
		return Range{}, false
	}
	
	modifier := obj[0] // 'i' or 'a'
	kind := obj[1]     // 'w', 'W', 'p', '(', '{', '[', '"', '\'', '`'
	
	switch kind {
	case 'w':
		return e.wordObject(text, pos, modifier == 'i', false)
	case 'W':
		return e.wordObject(text, pos, modifier == 'i', true)
	case 'p':
		return e.paragraphObject(text, pos, modifier == 'i')
	case '(', ')', 'b':
		return e.bracketObject(text, pos, modifier == 'i', '(', ')')
	case '{', '}', 'B':
		return e.bracketObject(text, pos, modifier == 'i', '{', '}')
	case '[', ']':
		return e.bracketObject(text, pos, modifier == 'i', '[', ']')
	case '"', '\'', '`':
		return e.quoteObject(text, pos, modifier == 'i', rune(kind))
	}
	return Range{}, false
}

func isWordChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_'
}

func isBlank(c rune) bool {
	return c == ' ' || c == '\t'
}

func (e *Engine) wordObject(text []rune, pos int, inner bool, bigWord bool) (Range, bool) {
	if pos >= len(text) {
		return Range{}, false
	}
	
	// Simplify: just find boundaries
	isWC := func(c rune) bool {
		if bigWord {
			return !isBlank(c) && c != '\n'
		}
		return isWordChar(c)
	}

	start := pos
	end := pos

	// Expand left
	if isWC(text[pos]) {
		for start > 0 && isWC(text[start-1]) {
			start--
		}
		for end < len(text) && isWC(text[end]) {
			end++
		}
	} else if !isBlank(text[pos]) && text[pos] != '\n' { // punctuation
		for start > 0 && !isWC(text[start-1]) && !isBlank(text[start-1]) && text[start-1] != '\n' {
			start--
		}
		for end < len(text) && !isWC(text[end]) && !isBlank(text[end]) && text[end] != '\n' {
			end++
		}
	} else { // on blank
		for start > 0 && isBlank(text[start-1]) {
			start--
		}
		for end < len(text) && isBlank(text[end]) {
			end++
		}
	}

	if !inner {
		// aw includes trailing whitespace, or leading if no trailing
		hasTrailing := false
		origEnd := end
		for end < len(text) && isBlank(text[end]) {
			end++
			hasTrailing = true
		}
		if !hasTrailing {
			for start > 0 && isBlank(text[start-1]) {
				start--
			}
		}
		// if we only expanded to the exact same word end, but inner is false, 
		// we return the word + whitespace.
		if end == origEnd && start == pos {
		    // fallback
		}
	}

	return Range{Start: start, End: end, LineWise: false}, true
}

func (e *Engine) paragraphObject(text []rune, pos int, inner bool) (Range, bool) {
	// A paragraph is defined by blank lines
	start := pos
	end := pos

	// Expand up
	for start > 0 {
		// is the line before this blank?
		prevLineStart := e.findLineStart(text, start-1)
		prevLineEnd := e.findLineEnd(text, start-1)
		isBlankLine := true
		for i := prevLineStart; i < prevLineEnd; i++ {
			if !isBlank(text[i]) {
				isBlankLine = false
				break
			}
		}
		if isBlankLine {
			break
		}
		start = prevLineStart
	}

	// Expand down
	for end < len(text) {
		nextLineEnd := e.findLineEnd(text, end)
		isBlankLine := true
		for i := end; i < nextLineEnd; i++ {
			if text[i] != '\n' && !isBlank(text[i]) {
				isBlankLine = false
				break
			}
		}
		if isBlankLine && end != pos { // don't stop on the very first blank if we started on it
			break
		}
		end = nextLineEnd + 1 // move to next line start
		if end >= len(text) {
			end = len(text)
			break
		}
	}

	if !inner {
		// Include following blank lines
		for end < len(text) {
			nextLineEnd := e.findLineEnd(text, end)
			isBlankLine := true
			for i := end; i < nextLineEnd; i++ {
				if text[i] != '\n' && !isBlank(text[i]) {
					isBlankLine = false
					break
				}
			}
			if !isBlankLine {
				break
			}
			end = nextLineEnd + 1
		}
	}

	return Range{Start: start, End: end, LineWise: true}, true
}

func (e *Engine) bracketObject(text []rune, pos int, inner bool, open, close rune) (Range, bool) {
	// Simplistic bracket matching
	start := -1
	end := -1

	// find open
	depth := 0
	for i := pos; i >= 0; i-- {
		if text[i] == close && i != pos {
			depth++
		} else if text[i] == open {
			if depth == 0 {
				start = i
				break
			}
			depth--
		}
	}

	if start == -1 {
		return Range{}, false
	}

	// find close
	depth = 0
	for i := pos; i < len(text); i++ {
		if text[i] == open && i != pos {
			depth++
		} else if text[i] == close {
			if depth == 0 {
				end = i
				break
			}
			depth--
		}
	}

	if end == -1 {
		return Range{}, false
	}

	if inner {
		return Range{Start: start + 1, End: end, LineWise: false}, true
	}
	return Range{Start: start, End: end + 1, LineWise: false}, true
}

func (e *Engine) quoteObject(text []rune, pos int, inner bool, quote rune) (Range, bool) {
	// Find quotes on current line
	lineStart := e.findLineStart(text, pos)
	lineEnd := e.findLineEnd(text, pos)

	var quotes []int
	for i := lineStart; i < lineEnd; i++ {
		if text[i] == quote {
			// ignore escaped
			if i > 0 && text[i-1] == '\\' {
				continue
			}
			quotes = append(quotes, i)
		}
	}

	for i := 0; i < len(quotes)-1; i += 2 {
		q1 := quotes[i]
		q2 := quotes[i+1]
		if pos >= q1 && pos <= q2 {
			if inner {
				return Range{Start: q1 + 1, End: q2, LineWise: false}, true
			}
			return Range{Start: q1, End: q2 + 1, LineWise: false}, true
		}
	}

	return Range{}, false
}
