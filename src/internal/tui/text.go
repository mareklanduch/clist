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

// wrapTwoWidth wraps text so the first segment fits firstW and all subsequent
// segments fit contW. Calling with firstW==contW is equivalent to plain word-wrap.
func wrapTwoWidth(text string, firstW, contW int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var segs []string
	var cur strings.Builder
	limit := firstW
	for _, word := range words {
		if cur.Len() == 0 {
			cur.WriteString(word)
		} else if cur.Len()+1+len(word) <= limit {
			cur.WriteByte(' ')
			cur.WriteString(word)
		} else {
			segs = append(segs, cur.String())
			cur.Reset()
			cur.WriteString(word)
			limit = contW
		}
	}
	segs = append(segs, cur.String())
	return segs
}
