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
		taskLns := m.renderTaskLines(m.filtered[i], w-2)
		if len(lines)+len(taskLns) > capacity {
			break
		}
		selected := i == m.selected
		for j, ln := range taskLns {
			prefix := "  "
			if j == 0 && selected {
				prefix = "▶ "
			}
			var line string
			if selected {
				// Width(w) fills the whole line with background so the selection stripe
				// is visible even when inner ANSI resets clear the background mid-content.
				line = lipgloss.NewStyle().
					Background(lipgloss.Color("236")).Width(w).
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
func (m Model) renderTaskLines(t task.Task, w int) []string {
	// Visible-char layout: prioIcon(2) + " " + statIcon(3) + " " = 7 chars.
	const iconW = 7

	// Measure raw suffix (tags + due) to check if it fits on the last wrapped line.
	var rawParts []string
	for _, tag := range t.Tags {
		rawParts = append(rawParts, " #"+tag)
	}
	if t.DueDate != nil {
		rawParts = append(rawParts, " "+t.DueDaysStr())
	}
	suffixLen := 0
	for _, p := range rawParts {
		suffixLen += len(p)
	}

	// All lines wrap at the same width; suffix goes at the end of the last one.
	contW := max(w-iconW, 8)
	segs := wrapTwoWidth(t.Title, contW, contW)
	// If the last segment plus the suffix overflows, add a blank segment so the
	// suffix lands on its own indented line.
	if suffixLen > 0 && len([]rune(segs[len(segs)-1]))+suffixLen > contW {
		segs = append(segs, "")
	}

	// Priority icon
	prioIcon := lipgloss.NewStyle().Foreground(priorityColor(t.Priority)).Render(t.PriorityIcon())

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
	statIcon := lipgloss.NewStyle().Foreground(sCol).Render(t.StatusIcon())

	// Title style
	var titleSt lipgloss.Style
	switch {
	case t.Status == task.StatusDone:
		titleSt = lipgloss.NewStyle().Foreground(colGray).Strikethrough(true)
	case t.Archived:
		titleSt = lipgloss.NewStyle().Foreground(colGray)
	case t.IsOverdue():
		titleSt = lipgloss.NewStyle().Foreground(colRed).Bold(true)
	case t.IsDueToday():
		titleSt = lipgloss.NewStyle().Foreground(colCyan)
	default:
		titleSt = lipgloss.NewStyle().Foreground(colWhite)
	}

	// Styled tags
	var tagParts []string
	for _, tag := range t.Tags {
		tagParts = append(tagParts, lipgloss.NewStyle().Foreground(colCyan).Render("#"+tag))
	}
	tagsStr := ""
	if len(tagParts) > 0 {
		tagsStr = " " + strings.Join(tagParts, " ")
	}

	// Styled due date
	dueStr := ""
	if t.DueDate != nil {
		days := t.DaysDue()
		var dueSt lipgloss.Style
		switch {
		case t.IsOverdue():
			dueSt = lipgloss.NewStyle().Foreground(colRed).Bold(true)
		case t.IsDueToday():
			dueSt = lipgloss.NewStyle().Foreground(colCyan).Bold(true)
		case days <= 3:
			dueSt = lipgloss.NewStyle().Foreground(colYellow)
		default:
			dueSt = lipgloss.NewStyle().Foreground(colGray)
		}
		dueStr = " " + dueSt.Render(t.DueDaysStr())
	}

	indent := strings.Repeat(" ", iconW)
	var result []string
	for i, seg := range segs {
		styled := titleSt.Render(seg)
		suffix := ""
		if i == len(segs)-1 {
			suffix = tagsStr + dueStr
		}
		if i == 0 {
			result = append(result, prioIcon+" "+statIcon+" "+styled+suffix)
		} else {
			result = append(result, indent+styled+suffix)
		}
	}
	return result
}
