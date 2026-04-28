package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"clist/internal/task"
)

func (m Model) renderStats(w, h int) string {
	innerW := w - 6

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colCyan).Bold(true).Render(statsBanner) + "\n\n")

	streak := 0
	for i := len(m.stats) - 1; i >= 0; i-- {
		if m.stats[i].Count > 0 {
			streak++
		} else {
			break
		}
	}

	totalDone, overdue, totalActive := 0, 0, 0
	for _, t := range m.tasks {
		if t.Archived {
			continue
		}
		if t.Status == task.StatusDone {
			totalDone++
		} else {
			totalActive++
			if t.IsOverdue() {
				overdue++
			}
		}
	}

	// panelW = content width so that two bordered+padded panels fill innerW.
	panelW := max((innerW-10)/2, 12)

	plural := "s"
	if streak == 1 {
		plural = ""
	}
	streakText := fmt.Sprintf("🔥 Current streak: %d day%s\n Done: %d  Overdue: %d",
		streak, plural, totalDone, overdue)
	streakBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).BorderForeground(colGray).
		Width(panelW).Padding(0, 1).MarginRight(2).
		Render(
			lipgloss.NewStyle().Foreground(colYellow).Bold(true).Render(" Streak ") + "\n" +
				lipgloss.NewStyle().Foreground(colYellow).Render(streakText),
		)

	ratio := 0.0
	if totalActive > 0 {
		ratio = float64(overdue) / float64(totalActive)
		if ratio > 1.0 {
			ratio = 1.0
		}
	}
	gaugeW := max(panelW-2, 2)
	gauge := renderProgressBar(ratio, gaugeW, colRed, colGray)
	overdueBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).BorderForeground(colGray).
		Width(panelW).Padding(0, 1).
		Render(
			lipgloss.NewStyle().Foreground(colRed).Bold(true).Render(" Overdue Ratio ") + "\n" +
				gauge + "\n" +
				lipgloss.NewStyle().Foreground(colWhite).
					Render(fmt.Sprintf("%d/%d overdue", overdue, totalActive)),
		)

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, streakBox, overdueBox))

	// Clip to available height with scroll support.
	innerH := max(h-4, 1)
	allLines := strings.Split(sb.String(), "\n")
	maxOff := max(len(allLines)-innerH, 0)
	off := min(m.scrollOffset, maxOff)
	visible := strings.Join(allLines[off:min(off+innerH, len(allLines))], "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).BorderForeground(colGray).
		Width(w-2).Height(h-2).Padding(1, 2).
		Render(visible)
}

func renderProgressBar(ratio float64, width int, filledCol, emptyCol lipgloss.Color) string {
	if width < 1 {
		return ""
	}
	filled := min(int(ratio*float64(width)), width)
	return lipgloss.NewStyle().Foreground(filledCol).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(emptyCol).Render(strings.Repeat("░", width-filled))
}
