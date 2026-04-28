package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"clist/internal/storage"
	"clist/internal/task"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		m.reload()
		return m, tickCmd()
	case tea.KeyMsg:
		switch m.mode {
		case ModeNormal:
			return m.updateNormal(msg)
		case ModeAdding:
			return m.updateAdding(msg)
		case ModeSearching:
			return m.updateSearching(msg)
		case ModeConfirmDelete:
			return m.updateConfirmDelete(msg)
		}
	}
	return m, nil
}

func (m *Model) switchView(v AppView) {
	m.view = v
	m.selected = 0
	m.scrollOffset = 0
	m.applyFilter()
}

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		return m.updateHelp(msg)
	}
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h", "?":
		m.showHelp = !m.showHelp
	case "tab":
		m.switchView((m.view + 1) % 6)
	case "shift+tab":
		m.switchView((m.view + 5) % 6)
	case "1":
		m.switchView(ViewAll)
	case "2":
		m.switchView(ViewToday)
	case "3":
		m.switchView(ViewWaiting)
	case "4":
		m.switchView(ViewSomeday)
	case "5":
		m.switchView(ViewStats)
	case "6":
		m.switchView(ViewArchive)
	case "j", "down":
		if m.view == ViewStats {
			m.scrollOffset++
		} else if m.selected < len(m.filtered)-1 {
			m.selected++
			m.ensureVisible()
		}
	case "k", "up":
		if m.view == ViewStats {
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		} else if m.selected > 0 {
			m.selected--
			if m.selected < m.scrollOffset {
				m.scrollOffset = m.selected
			}
		}
	case "a":
		m.mode = ModeAdding
		m.input.Placeholder = "Buy milk #groceries !high due:today  due:1d  due:2m"
		m.input.SetValue("")
		m.input.Focus()
	case "e":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			t := m.filtered[m.selected]
			newArchived := !t.Archived
			if err := storage.UpdateArchived(m.db, t.ID, newArchived); err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				if newArchived {
					m.status = "Archived: " + t.Title
				} else {
					m.status = "Unarchived: " + t.Title
				}
				m.reload()
			}
		}
	case "d", "delete":
		if len(m.filtered) > 0 {
			m.mode = ModeConfirmDelete
		}
	case " ", "c":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			t := m.filtered[m.selected]
			var newStatus task.Status
			if t.Status == task.StatusDone {
				newStatus = task.StatusTodo
			} else {
				newStatus = task.StatusDone
			}
			if err := storage.UpdateStatus(m.db, t.ID, newStatus); err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				if newStatus == task.StatusDone {
					m.status = "Completed: " + t.Title
				} else {
					m.status = "Reopened: " + t.Title
				}
				m.reload()
			}
		}
	case "p":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			t := m.filtered[m.selected]
			if err := storage.UpdatePriority(m.db, t.ID, task.NextPriority(t.Priority)); err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				m.reload()
			}
		}
	case "s":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			t := m.filtered[m.selected]
			if err := storage.UpdateStatus(m.db, t.ID, task.NextStatus(t.Status)); err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				m.reload()
			}
		}
	case "/":
		m.mode = ModeSearching
		m.input.Placeholder = "Search tasks..."
		m.input.SetValue("")
		m.input.Focus()
	}
	return m, nil
}

func (m Model) updateAdding(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			title, tags, priority, dueDate := task.ParseInput(val)
			if title != "" {
				t := task.Task{
					Title:     title,
					Tags:      tags,
					Priority:  priority,
					DueDate:   dueDate,
					CreatedAt: time.Now(),
				}
				if err := storage.AddTask(m.db, t); err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				} else {
					m.status = "Added: " + title
					m.reload()
				}
			}
		}
		m.mode = ModeNormal
		m.input.Blur()
	case "esc":
		m.mode = ModeNormal
		m.input.Blur()
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) updateSearching(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.search = m.input.Value()
		m.mode = ModeNormal
		m.input.Blur()
		m.applyFilter()
	case "esc":
		m.search = ""
		m.mode = ModeNormal
		m.input.Blur()
		m.applyFilter()
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.search = m.input.Value()
		m.applyFilter()
		return m, cmd
	}
	return m, nil
}

func (m Model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			t := m.filtered[m.selected]
			if err := storage.DeleteTask(m.db, t.ID); err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				m.status = "Deleted: " + t.Title
				m.reload()
			}
		}
		m.mode = ModeNormal
	case "n", "esc":
		m.mode = ModeNormal
	}
	return m, nil
}

func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.helpScroll++
	case "k", "up":
		if m.helpScroll > 0 {
			m.helpScroll--
		}
	case "ctrl+c":
		return m, tea.Quit
	default:
		m.showHelp = false
		m.helpScroll = 0
	}
	return m, nil
}
