package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}
	if m.showHelp {
		return m.renderHelp()
	}
	if m.mode == ModeConfirmDelete {
		return m.renderDeleteModal()
	}
	if m.mode == ModePickStatus || m.mode == ModePickPriority {
		return m.renderPicker()
	}
	return m.renderMain()
}

func (m Model) renderMain() string {
	const sidebarW = 26
	const detailW = 30

	showDetail := m.mode == ModeNormal &&
		len(m.filtered) > 0 &&
		m.selected < len(m.filtered) &&
		m.view != ViewStats

	mainW := m.width - sidebarW
	if showDetail {
		mainW -= detailW
	}
	if mainW < 20 {
		mainW = 20
	}

	sidebar := m.renderSidebar(sidebarW, m.height)
	main := m.renderMainContent(mainW, m.height)

	if showDetail {
		detail := m.renderDetail(detailW, m.height)
		return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main, detail)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

func (m Model) renderSidebar(w, h int) string {
	innerW := w - 4

	type viewItem struct {
		key   string
		icon  string
		label string
		view  AppView
		count int
		showN bool
	}
	items := []viewItem{
		{"1", "≡ ", "All Tasks", ViewAll, m.counts[0], true},
		{"2", "◉ ", "Today", ViewToday, m.counts[1], true},
		{"3", "⏳", "Waiting", ViewWaiting, m.counts[2], true},
		{"4", "☁ ", "Someday", ViewSomeday, m.counts[3], true},
		{"5", "▦ ", "Stats", ViewStats, 0, false},
		{"6", "□ ", "Archive", ViewArchive, m.counts[4], true},
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(colCyan).Bold(true).Render("☑ CLIst"))
	lines = append(lines, "")

	for _, item := range items {
		var label string
		if item.showN {
			label = fmt.Sprintf("%s %s%s [%d]", item.key, item.icon, item.label, item.count)
		} else {
			label = fmt.Sprintf("%s %s%s", item.key, item.icon, item.label)
		}
		label = runesTruncate(label, innerW)
		if item.view == m.view {
			label = lipgloss.NewStyle().
				Background(colBlue).Foreground(colWhite).Bold(true).
				Width(innerW).Render(label)
		}
		lines = append(lines, label)
	}

	lines = append(lines, lipgloss.NewStyle().Foreground(colGray).Render(strings.Repeat("─", innerW)))

	overdue := 0
	for _, t := range m.tasks {
		if t.IsOverdue() && !t.Archived {
			overdue++
		}
	}
	if overdue > 0 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(colRed).Bold(true).
			Render(fmt.Sprintf("⚠ %d overdue", overdue)))
	}

	body := strings.Join(lines, "\n")

	innerH := h - 2
	bodyLines := strings.Count(body, "\n") + 1
	hint := lipgloss.NewStyle().Foreground(colGray).Render("h: help")
	padding := max(innerH-bodyLines, 0)
	full := body + strings.Repeat("\n", padding) + hint

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).BorderForeground(colGray).
		Width(w-2).Height(h-2).Padding(0, 1).
		Render(full)
}

func (m Model) renderMainContent(w, h int) string {
	if m.view == ViewStats {
		return m.renderStats(w, h)
	}

	inner := w - 4

	statusLine := m.renderTopStatus(inner)
	statusH := 0
	if statusLine != "" {
		statusH = 1
	}

	var inputSection string
	inputH := 0
	if m.mode == ModeAdding {
		inputSection = m.renderAddInput(inner)
		inputH = 4
	}

	listH := max(h-2-statusH-inputH, 2)
	taskList := m.renderTaskList(inner, listH)

	var parts []string
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	parts = append(parts, taskList)
	if inputH > 0 {
		parts = append(parts, inputSection)
	}

	content := strings.Join(parts, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).BorderForeground(colGray).
		Width(w-2).Height(h-2).Padding(0, 1).
		Render(content)
}

func (m Model) renderTopStatus(maxW int) string {
	if m.mode == ModeSearching {
		return "🔍 " + m.input.View()
	}
	if m.err != nil {
		return lipgloss.NewStyle().Foreground(colRed).Render(truncateEllipsis("DB error: "+m.err.Error(), maxW))
	}
	if m.search != "" {
		return lipgloss.NewStyle().Foreground(colCyan).Render(truncateEllipsis("Filter: "+m.search, maxW))
	}
	if m.status != "" {
		return lipgloss.NewStyle().Foreground(colGreen).Render(truncateEllipsis(m.status, maxW))
	}
	return ""
}

func (m Model) renderAddInput(w int) string {
	prompt := lipgloss.NewStyle().Foreground(colCyan).Render("Add task (GTD syntax):")
	inputBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).BorderForeground(colCyan).
		Width(w - 2).Render(m.input.View())
	return lipgloss.JoinVertical(lipgloss.Left, prompt, inputBox)
}
