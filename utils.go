package vimengine

func (e *Engine) moveLines(text []rune, pos int, delta int) int {
	lineStart, col := 0, 0
	currLine := 0

	for i := 0; i < pos; i++ {
		if text[i] == '\n' {
			currLine++
			lineStart = i + 1
			col = 0
		} else {
			col++
		}
	}
	_ = lineStart

	targetLine := currLine + delta
	if targetLine < 0 {
		targetLine = 0
	}

	tLineStart := 0
	tLineIdx := 0
	for i := 0; i < len(text); i++ {
		if tLineIdx == targetLine {
			tLineStart = i
			break
		}
		if text[i] == '\n' {
			tLineIdx++
			if tLineIdx == targetLine && i+1 < len(text) {
				tLineStart = i + 1
			}
		}
	}

	if tLineIdx < targetLine {
		return len(text)
	}

	targetPos := tLineStart
	for i := 0; i < col; i++ {
		if targetPos >= len(text) || text[targetPos] == '\n' {
			break
		}
		targetPos++
	}

	return targetPos
}

func (e *Engine) findLineStart(text []rune, pos int) int {
	for i := pos - 1; i >= 0; i-- {
		if text[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

func (e *Engine) findLineEnd(text []rune, pos int) int {
	for i := pos; i < len(text); i++ {
		if text[i] == '\n' {
			return i
		}
	}
	return len(text)
}

func (e *Engine) moveWord(text []rune, pos int, dir int, bigWord bool) int {
	if dir > 0 {
		for i := pos; i < len(text); i++ {
			if text[i] == ' ' || text[i] == '\n' {
				for j := i; j < len(text); j++ {
					if text[j] != ' ' && text[j] != '\n' {
						return j
					}
				}
				return len(text)
			}
		}
		return len(text)
	} else {
		if pos == 0 {
			return 0
		}
		i := pos - 1
		for i >= 0 && (text[i] == ' ' || text[i] == '\n') {
			i--
		}
		for i >= 0 && text[i] != ' ' && text[i] != '\n' {
			i--
		}
		return i + 1
	}
}
