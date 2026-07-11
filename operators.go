package vimengine

import "unicode"

func (e *Engine) applyOperator(editor Editor, op string, rng Range) {
	start := rng.Start
	end := rng.End

	if start > end {
		start, end = end, start
	}

	text := editor.Text()

	if rng.LineWise {
		start = e.findLineStart(text, start)
		if end < len(text) {
			end = e.findLineEnd(text, end)
		}
		if op == "d" || op == "y" || op == "c" {
			// for linewise operations, typically grab the trailing newline too
			if end < len(text) {
				end++ // include \n
			}
		}
	}

	switch op {
	case "y":
		if start < end {
			e.register = append([]rune{}, text[start:end]...)
		}
		// cursor moves to start of yank
		editor.SetSelectedRange(start, start)
	case "d":
		if start < end {
			e.register = append([]rune{}, text[start:end]...)
			editor.Replace(start, end, nil)
		}
		// bound check
		text = editor.Text()
		if start >= len(text) {
			start = len(text)
		}
		editor.SetSelectedRange(start, start)
	case "c":
		if start < end {
			e.register = append([]rune{}, text[start:end]...)
			editor.Replace(start, end, nil)
		}
		text = editor.Text()
		if start >= len(text) {
			start = len(text)
		}
		editor.SetSelectedRange(start, start)
		e.setSubmode(Insert)
	case ">", "<":
		// Indent / outdent
		s := e.findLineStart(text, start)
		e2 := e.findLineEnd(text, end)
		lines := splitLines(text[s:e2])
		newLines := make([]rune, 0, e2-s)
		for _, line := range lines {
			if len(line) == 0 {
				newLines = append(newLines, '\n')
				continue
			}
			if op == ">" {
				newLines = append(newLines, ' ', ' ') // 2 spaces
				newLines = append(newLines, line...)
			} else {
				// outdent up to 2 spaces
				idx := 0
				spaces := 0
				for idx < len(line) && spaces < 2 && line[idx] == ' ' {
					idx++
					spaces++
				}
				newLines = append(newLines, line[idx:]...)
			}
			newLines = append(newLines, '\n')
		}
		// remove last newline if it wasn't there originally
		if len(newLines) > 0 && e2 >= len(text) {
			newLines = newLines[:len(newLines)-1]
		}
		editor.Replace(s, e2, newLines)
		editor.SetSelectedRange(s, s)
	case "gU", "gu", "g~":
		if start < end {
			runes := text[start:end]
			for i, r := range runes {
				if op == "gU" {
					runes[i] = unicode.ToUpper(r)
				} else if op == "gu" {
					runes[i] = unicode.ToLower(r)
				} else if op == "g~" {
					if unicode.IsUpper(r) {
						runes[i] = unicode.ToLower(r)
					} else {
						runes[i] = unicode.ToUpper(r)
					}
				}
			}
			editor.Replace(start, end, runes)
		}
		editor.SetSelectedRange(start, start)
	}
}

func splitLines(text []rune) [][]rune {
	var lines [][]rune
	start := 0
	for i, r := range text {
		if r == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	if start <= len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}
