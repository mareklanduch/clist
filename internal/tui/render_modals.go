package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"clist/internal/task"
)

func (m Model) renderDeleteModal() string {
	title := ""
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		title = m.filtered[m.selected].Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}
	}

	content := fmt.Sprintf("\n  Delete: \"%s\"?\n\n  y/Enter: Yes        n/Esc: Cancel\n", title)

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(colRed).
		Padding(0, 2).Width(54).
		Render(lipgloss.NewStyle().Foreground(colRed).Bold(true).Render("Delete Task") + "\n" + content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) renderHelp() string {
	title := lipgloss.NewStyle().Foreground(colCyan).Bold(true)
	k := lipgloss.NewStyle().Foreground(colYellow)
	d := lipgloss.NewStyle().Foreground(colWhite)
	dim := lipgloss.NewStyle().Foreground(colGray)

	var all []string
	all = append(all, title.Render("Navigation"))
	all = append(all, "  "+k.Render("↑/↓  j/k")+"       "+d.Render("Navigate tasks"))
	all = append(all, "  "+k.Render("1-6")+"            "+d.Render("Switch views"))
	all = append(all, "  "+k.Render("Tab / Shift+Tab")+"  "+d.Render("Next / prev view"))
	all = append(all, "")
	all = append(all, title.Render("Task Actions"))
	all = append(all, "  "+k.Render("a")+"              "+d.Render("Add new task"))
	all = append(all, "  "+k.Render("e")+"              "+d.Render("Edit task (opens in input box)"))
	all = append(all, "  "+k.Render("Space")+"          "+d.Render("Toggle done"))
	all = append(all, "  "+k.Render("A")+"              "+d.Render("Archive / unarchive"))
	all = append(all, "  "+k.Render("d / Delete")+"     "+d.Render("Delete task"))
	all = append(all, "  "+k.Render("p")+"              "+d.Render("Cycle priority"))
	all = append(all, "  "+k.Render("s")+"              "+d.Render("Cycle status"))
	all = append(all, "")
	all = append(all, title.Render("Search"))
	all = append(all, "  "+k.Render("/")+"              "+d.Render("Enter search mode"))
	all = append(all, "  "+k.Render("Esc")+"            "+d.Render("Clear search / cancel"))
	all = append(all, "")
	all = append(all, title.Render("Add Task Syntax"))
	all = append(all, "  "+d.Render("title #tag !priority due:DATE"))
	all = append(all, "  "+dim.Render("DATE: YYYY-MM-DD | today | 1d | -1d | 1m | -1m"))
	all = append(all, "  "+dim.Render("priority: !critical  !high  !medium  !low"))
	all = append(all, "")
	all = append(all, title.Render("Vault"))
	all = append(all, "  "+k.Render("v")+"              "+d.Render("Open vault manager (switch/add/remove)"))
	all = append(all, "")
	all = append(all, title.Render("Other"))
	all = append(all, "  "+k.Render("h / ?")+"          "+d.Render("Toggle this help  (any key closes)"))
	all = append(all, "  "+k.Render("q")+"              "+d.Render("Quit"))

	// Sizing: border=2, padding top+bottom=2, header "title\n\n"=2 → overhead=6
	const overhead = 6
	modalW := min(66, m.width-4)
	boxH := min(m.height-2, len(all)+overhead)
	visibleH := max(boxH-overhead, 1)

	// Clamp scroll
	maxOff := max(len(all)-visibleH, 0)
	off := min(m.helpScroll, maxOff)

	// Reserve rows for indicators
	showTop := off > 0
	showBot := off < maxOff
	avail := visibleH
	if showTop {
		avail--
	}
	if showBot {
		avail--
	}
	if avail < 1 {
		avail = 1
	}

	end := min(off+avail, len(all))
	visible := all[off:end]

	var parts []string
	if showTop {
		parts = append(parts, dim.Render("  ▲  scroll up"))
	}
	parts = append(parts, visible...)
	if showBot {
		parts = append(parts, dim.Render("  ▼  scroll down  (j/k)"))
	}

	header := title.Render("Keyboard Shortcuts") + "\n\n"
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(colCyan).
		Padding(1, 2).Width(modalW).Height(boxH - 2).
		Render(header + strings.Join(parts, "\n"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) renderVaultPicker() string {
	vaults := m.vault.Vaults
	dim := lipgloss.NewStyle().Foreground(colGray)

	var rows []string
	for i, v := range vaults {
		marker := "  "
		if v.Name == m.vault.Active {
			marker = "● "
		}
		typeTag := ""
		if v.IsRemote() {
			typeTag = dim.Render(" [remote:" + v.Token + "]")
		}
		row := fmt.Sprintf("  %s%-18s", marker, v.Name)
		switch {
		case i == m.pickerIdx:
			rows = append(rows, lipgloss.NewStyle().Background(colBlue).Foreground(colWhite).Bold(true).Render(row+" ◀")+typeTag)
		case v.Name == m.vault.Active:
			rows = append(rows, lipgloss.NewStyle().Foreground(colCyan).Render(row)+typeTag)
		default:
			rows = append(rows, lipgloss.NewStyle().Foreground(colWhite).Render(row)+typeTag)
		}
	}

	var bottom string
	switch m.mode {
	case ModeVaultAdd:
		prompt := lipgloss.NewStyle().Foreground(colCyan).Render("  New local vault name:")
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).BorderForeground(colCyan).
			Width(24).Render(m.input.View())
		hint := dim.Render("  Enter: create  ·  Esc: cancel")
		bottom = "\n" + prompt + "\n" + inputBox + "\n" + hint

	case ModeVaultConfirmRemove:
		name := ""
		if m.pickerIdx < len(vaults) {
			name = vaults[m.pickerIdx].Name
		}
		warn := lipgloss.NewStyle().Foreground(colRed).Bold(true).Render(fmt.Sprintf("  Remove %q?", name))
		hint := dim.Render("  y/Enter: yes  ·  n/Esc: cancel")
		bottom = "\n" + warn + "\n" + hint

	case ModeVaultRemoteToken:
		prompt := lipgloss.NewStyle().Foreground(colCyan).Render("  Token (blank = generate new):")
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).BorderForeground(colCyan).
			Width(36).Render(m.input.View())
		hint := dim.Render("  Enter: confirm  ·  Esc: cancel")
		bottom = "\n" + prompt + "\n" + inputBox + "\n" + hint

	case ModeVaultRemoteName:
		short := m.remoteVaultToken
		if len(short) > 8 {
			short = short[:8] + "…"
		}
		prompt := lipgloss.NewStyle().Foreground(colCyan).Render(fmt.Sprintf("  Name for token %s:", short))
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).BorderForeground(colCyan).
			Width(24).Render(m.input.View())
		hint := dim.Render("  Enter: save  ·  Esc: cancel")
		bottom = "\n" + prompt + "\n" + inputBox + "\n" + hint

	default:
		hint := dim.Render("  Enter/s: switch  ·  a: local  ·  r: remote  ·  d: remove  ·  Esc: close")
		bottom = "\n" + hint
	}

	titleLine := lipgloss.NewStyle().Foreground(colCyan).Bold(true).Render("Vaults")
	content := "\n" + strings.Join(rows, "\n") + "\n" + bottom

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(colCyan).
		Padding(0, 2).
		Render(titleLine + content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) renderPicker() string {
	isStatus := m.mode == ModePickStatus

	var titleStr string
	var rows []string

	if isStatus {
		titleStr = "Set Status"
		for i, s := range pickerStatuses {
			dummy := task.Task{Status: s}
			row := fmt.Sprintf("  %d  %s  %-14s", i+1, dummy.StatusIcon(), dummy.StatusLabel())
			if i == m.pickerIdx {
				rows = append(rows, lipgloss.NewStyle().Background(colBlue).Foreground(colWhite).Bold(true).Render(row+" ◀"))
			} else {
				rows = append(rows, lipgloss.NewStyle().Foreground(colWhite).Render(row))
			}
		}
	} else {
		titleStr = "Set Priority"
		for i, p := range pickerPriorities {
			dummy := task.Task{Priority: p}
			row := fmt.Sprintf("  %d  %s  %-14s", i+1, dummy.PriorityIcon(), dummy.PriorityLabel())
			if i == m.pickerIdx {
				rows = append(rows, lipgloss.NewStyle().Background(colBlue).Foreground(colWhite).Bold(true).Render(row+" ◀"))
			} else {
				rows = append(rows, lipgloss.NewStyle().Foreground(priorityColor(p)).Render(row))
			}
		}
	}

	hint := lipgloss.NewStyle().Foreground(colGray).Render("  j/k navigate  ·  Enter confirm  ·  1-N instant pick  ·  Esc cancel")
	content := "\n" + strings.Join(rows, "\n") + "\n\n" + hint

	titleLine := lipgloss.NewStyle().Foreground(colCyan).Bold(true).Render(titleStr)
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(colCyan).
		Padding(0, 2).
		Render(titleLine + content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}
