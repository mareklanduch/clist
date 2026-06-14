package tui

import "strings"

// runesTruncate truncates s to at most maxW runes. Safe for plain text
// strings — apply styling after truncation.
func runesTruncate(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxW {
		return s
	}
	return string(r[:maxW])
}

// truncateEllipsis truncates s to maxW runes, appending "..." when truncated.
func truncateEllipsis(s string, maxW int) string {
	r := []rune(s)
	if len(r) <= maxW {
		return s
	}
	if maxW <= 3 {
		return string(r[:maxW])
	}
	return string(r[:maxW-3]) + "..."
}

// wrap word-wraps text so every segment fits within w runes. Words longer
// than w are hard-broken so no segment can exceed the width.
func wrap(text string, w int) []string {
	if w < 1 {
		w = 1
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var segs []string
	var cur []rune
	flush := func() {
		segs = append(segs, string(cur))
		cur = cur[:0]
	}
	for _, word := range words {
		r := []rune(word)
		// Hard-break words that can never fit on one line.
		for len(r) > w {
			if len(cur) > 0 {
				flush()
			}
			segs = append(segs, string(r[:w]))
			r = r[w:]
		}
		switch {
		case len(cur) == 0:
			cur = append(cur, r...)
		case len(cur)+1+len(r) <= w:
			cur = append(cur, ' ')
			cur = append(cur, r...)
		default:
			flush()
			cur = append(cur, r...)
		}
	}
	flush()
	return segs
}
