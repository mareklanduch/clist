package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDetail(w, h int) string {
	if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).BorderForeground(colGray).
			Width(w - 2).Height(h - 2).Render("")
	}

	t := m.filtered[m.selected]
	innerW := w - 4

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(colWhite).
		Render(strings.Join(wrap(t.Title, innerW), "\n")))
	lines = append(lines, "")

	prioStr := t.PriorityIcon() + " " + t.PriorityLabel()
	lines = append(lines, "Priority: "+lipgloss.NewStyle().Foreground(priorityColor(t.Priority)).Render(prioStr))
	lines = append(lines, "Status:   "+t.StatusIcon()+" "+t.StatusLabel())

	dueStr := "--"
	if t.DueDate != nil {
		dueStr = t.DueDate.Format("2006-01-02") + " (" + t.DueDaysStr() + ")"
	}
	lines = append(lines, "Due:      "+dueStr)

	if len(t.Tags) > 0 {
		tagStrs := make([]string, len(t.Tags))
		for i, tag := range t.Tags {
			tagStrs[i] = lipgloss.NewStyle().Foreground(colCyan).Render("#" + tag)
		}
		lines = append(lines, "Tags:     "+strings.Join(tagStrs, " "))
	}

	lines = append(lines, "Created:  "+t.CreatedAt.Format("2006-01-02 15:04"))

	if t.CompletedAt != nil {
		lines = append(lines, lipgloss.NewStyle().Foreground(colGreen).
			Render("Completed: "+t.CompletedAt.Format("2006-01-02 15:04")))
	} else {
		lines = append(lines, "Completed: --")
	}

	if t.Notes != "" {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(colGray).Render("Notes:"))
		lines = append(lines, strings.Join(wrap(t.Notes, innerW), "\n"))
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(colGray).Render(fmt.Sprintf("ID: #%d", t.ID)))

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).BorderForeground(colGray).
		Width(w-2).Height(h-2).Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}
