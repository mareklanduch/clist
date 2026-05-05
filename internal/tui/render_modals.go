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

	const w = 54
	center := func(s string) string {
		return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(s)
	}

	heading := lipgloss.NewStyle().Foreground(colRed).Bold(true).Render("Delete Task")
	body := fmt.Sprintf(`Delete: "%s"?`, title)
	hint := lipgloss.NewStyle().Foreground(colGray).Render("y/Enter: Yes  ·  n/Esc: Cancel")

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(colRed).
		Padding(1, 2).Width(w).
		Render(center(heading) + "\n\n" + center(body) + "\n\n" + center(hint))

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
	all = append(all, "  "+k.Render("y")+"              "+d.Render("Copy title to clipboard"))
	all = append(all, "  "+k.Render("Ctrl+Y")+"         "+d.Render("Copy full task (title + tags + priority + due)"))
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

	const w = 50
	// PlaceHorizontal avoids nested Width constraints that corrupt ANSI-coded strings.
	center := func(s string) string {
		return lipgloss.PlaceHorizontal(w, lipgloss.Center, s)
	}
	inputW := w - 2

	var rows []string
	for i, v := range vaults {
		marker := "  "
		if v.Name == m.vault.Active {
			marker = "● "
		}
		isSelected := i == m.pickerIdx

		name := v.Name
		if len(name) > 20 {
			name = name[:19] + "…"
		}
		nameLine := fmt.Sprintf("  %s%s", marker, name)

		switch {
		case isSelected:
			rows = append(rows, lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(colWhite).Width(w).Render(nameLine))
		case v.Name == m.vault.Active:
			rows = append(rows, lipgloss.NewStyle().Foreground(colCyan).Render(nameLine))
		default:
			rows = append(rows, lipgloss.NewStyle().Foreground(colWhite).Render(nameLine))
		}

		if v.IsRemote() {
			token := v.Token
			if len(token) > w-11 { // len("    token: ") == 11
				token = token[:w-12] + "…"
			}
			tokenLine := "    token: " + token
			if isSelected {
				rows = append(rows, lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(colGray).Width(w).Render(tokenLine))
			} else {
				rows = append(rows, dim.Render(tokenLine))
			}
		}
	}

	var bottom string
	switch m.mode {
	case ModeVaultAdd:
		prompt := lipgloss.NewStyle().Foreground(colCyan).Render("New local vault name:")
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).BorderForeground(colCyan).
			Width(inputW).Render(m.input.View())
		hint := dim.Render("Enter: create  ·  Esc: cancel")
		bottom = "\n" + center(prompt) + "\n" + inputBox + "\n" + center(hint)

	case ModeVaultConfirmRemove:
		name := ""
		if m.pickerIdx < len(vaults) {
			name = vaults[m.pickerIdx].Name
		}
		warn := lipgloss.NewStyle().Foreground(colRed).Bold(true).Render(fmt.Sprintf("Remove %q?", name))
		hint := dim.Render("y/Enter: yes  ·  n/Esc: cancel")
		bottom = "\n" + center(warn) + "\n" + center(hint)

	case ModeVaultRemoteToken:
		prompt := lipgloss.NewStyle().Foreground(colCyan).Render("Token (blank = generate new):")
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).BorderForeground(colCyan).
			Width(inputW).Render(m.input.View())
		hint := dim.Render("Enter: confirm  ·  Esc: cancel")
		bottom = "\n" + center(prompt) + "\n" + inputBox + "\n" + center(hint)

	case ModeVaultRemoteName:
		short := m.remoteVaultToken
		if len(short) > 8 {
			short = short[:8] + "…"
		}
		prompt := lipgloss.NewStyle().Foreground(colCyan).Render(fmt.Sprintf("Name for token %s:", short))
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).BorderForeground(colCyan).
			Width(inputW).Render(m.input.View())
		hint := dim.Render("Enter: save  ·  Esc: cancel")
		bottom = "\n" + center(prompt) + "\n" + inputBox + "\n" + center(hint)

	default:
		hint1 := dim.Render("Enter/s: switch  ·  a: local  ·  r: remote")
		hint2 := dim.Render("d: remove  ·  Esc: close")
		bottom = "\n" + center(hint1) + "\n" + center(hint2)
	}

	titleLine := lipgloss.NewStyle().Foreground(colCyan).Bold(true).Render("Vaults")
	content := "\n" + strings.Join(rows, "\n") + "\n" + bottom

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(colCyan).
		Padding(0, 2).
		Render(center(titleLine) + content)

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

	const w = 40
	center := func(s string) string {
		return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(s)
	}

	hint1 := lipgloss.NewStyle().Foreground(colGray).Render("j/k navigate  ·  Enter confirm")
	hint2 := lipgloss.NewStyle().Foreground(colGray).Render("1-N instant pick  ·  Esc cancel")
	titleLine := lipgloss.NewStyle().Foreground(colCyan).Bold(true).Render(titleStr)
	content := "\n" + strings.Join(rows, "\n") + "\n\n" + center(hint1) + "\n" + center(hint2)

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(colCyan).
		Padding(0, 2).Width(w).
		Render(center(titleLine) + content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}
