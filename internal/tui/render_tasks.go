package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"clist/internal/task"
)

func (m Model) renderTaskList(w, h int) string {
	if len(m.filtered) == 0 {
		return lipgloss.NewStyle().Foreground(colGray).Render("(no tasks)")
	}

	dim := lipgloss.NewStyle().Foreground(colGray)
	clip := lipgloss.NewStyle().MaxWidth(w)
	showTop := m.scrollOffset > 0

	var lines []string
	if showTop {
		lines = append(lines, clip.Render(dim.Render("  ▲  more above")))
	}

	// capacity reserves 1 line for the potential ▼ indicator.
	capacity := h - 1
	endIdx := m.scrollOffset
	for i := m.scrollOffset; i < len(m.filtered); i++ {
		selected := i == m.selected
		selBg := lipgloss.Color("")
		if selected {
			selBg = lipgloss.Color("236")
		}
		taskLns := m.renderTaskLines(m.filtered[i], w-2, selBg)
		if len(lines)+len(taskLns) > capacity {
			break
		}
		for j, ln := range taskLns {
			prefix := "  "
			if j == 0 && selected {
				inputActive := m.mode == ModeAdding || m.mode == ModeEditing ||
					m.mode == ModeSearching
				if inputActive || (m.animFrame/6)%2 == 0 {
					prefix = "▶ "
				} else {
					prefix = "▷ "
				}
			}
			var line string
			if selected {
				line = lipgloss.NewStyle().
					Background(selBg).Width(w).MaxWidth(w).
					Render(prefix + ln)
			} else {
				line = clip.Render(prefix + ln)
			}
			lines = append(lines, line)
		}
		endIdx = i + 1
	}

	showBot := endIdx < len(m.filtered)
	if showBot {
		for len(lines) < h-1 {
			lines = append(lines, "")
		}
		lines = append(lines, clip.Render(dim.Render("  ▼  more below")))
	} else {
		for len(lines) < h {
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n")
}

// renderTaskLines renders one task as 1+ wrapped lines.
// w is the available width before the 2-char "▶ "/"  " prefix added by the caller.
// selBg, when non-empty, is applied to every inner style so the selection highlight
// is not broken by inner ANSI resets (\033[0m).
func (m Model) renderTaskLines(t task.Task, w int, selBg lipgloss.Color) []string {
	withBg := func(s lipgloss.Style) lipgloss.Style {
		if selBg != "" {
			return s.Background(selBg)
		}
		return s
	}
	sp := " "
	if selBg != "" {
		sp = lipgloss.NewStyle().Background(selBg).Render(" ")
	}

	// Visible-char layout: prioIcon(2) + sp + statIcon(3) + sp = 7 chars.
	const iconW = 7

	// Measure raw suffix (tags + due) to check if it fits on the last wrapped line.
	suffixLen := 0
	for _, tag := range t.Tags {
		suffixLen += len([]rune(tag)) + 2 // " #tag"
	}
	if t.DueDate != nil {
		suffixLen += len([]rune(t.DueDaysStr())) + 1
	}

	// All lines wrap at the same width; suffix goes at the end of the last one.
	contW := max(w-iconW, 8)
	segs := wrap(t.Title, contW)
	// If the last segment plus the suffix overflows, add a blank segment so the
	// suffix lands on its own indented line.
	if suffixLen > 0 && len([]rune(segs[len(segs)-1]))+suffixLen > contW {
		segs = append(segs, "")
	}

	// Priority icon
	prioIcon := withBg(lipgloss.NewStyle().Foreground(priorityColor(t.Priority))).Render(t.PriorityIcon())

	// Status icon color
	var sCol lipgloss.Color
	switch t.Status {
	case task.StatusDone:
		sCol = colGray
	case task.StatusInProgress:
		sCol = colGreen
	case task.StatusWaiting:
		sCol = colYellow
	default:
		sCol = colWhite
	}
	statIcon := withBg(lipgloss.NewStyle().Foreground(sCol)).Render(t.StatusIcon())

	// Title style
	var titleSt lipgloss.Style
	switch {
	case t.Status == task.StatusDone:
		titleSt = withBg(lipgloss.NewStyle().Foreground(colGray).Strikethrough(true))
	case t.Archived:
		titleSt = withBg(lipgloss.NewStyle().Foreground(colGray))
	case t.IsOverdue():
		titleSt = withBg(lipgloss.NewStyle().Foreground(colRed).Bold(true))
	case t.IsDueToday():
		titleSt = withBg(lipgloss.NewStyle().Foreground(colLightRed))
	default:
		titleSt = withBg(lipgloss.NewStyle().Foreground(colWhite))
	}

	// Styled tags
	var tagParts []string
	for _, tag := range t.Tags {
		tagParts = append(tagParts, withBg(lipgloss.NewStyle().Foreground(colCyan)).Render("#"+tag))
	}
	tagsStr := ""
	if len(tagParts) > 0 {
		tagsStr = sp + strings.Join(tagParts, sp)
	}

	// Styled due date
	dueStr := ""
	if t.DueDate != nil {
		days := t.DaysDue()
		var dueSt lipgloss.Style
		switch {
		case t.IsOverdue():
			dueSt = withBg(lipgloss.NewStyle().Foreground(colRed).Bold(true))
		case t.IsDueToday():
			dueSt = withBg(lipgloss.NewStyle().Foreground(colCyan).Bold(true))
		case days <= 3:
			dueSt = withBg(lipgloss.NewStyle().Foreground(colYellow))
		default:
			dueSt = withBg(lipgloss.NewStyle().Foreground(colGray))
		}
		dueStr = sp + dueSt.Render(t.DueDaysStr())
	}

	indent := withBg(lipgloss.NewStyle()).Render(strings.Repeat(" ", iconW))
	var result []string
	for i, seg := range segs {
		styled := titleSt.Render(seg)
		suffix := ""
		if i == len(segs)-1 {
			suffix = tagsStr + dueStr
		}
		if i == 0 {
			result = append(result, prioIcon+sp+statIcon+sp+styled+suffix)
		} else {
			result = append(result, indent+styled+suffix)
		}
	}
	return result
}
