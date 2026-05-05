package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"

	"clist/internal/storage"
	"clist/internal/task"
	"clist/internal/vault"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		m.pollUpdates()
		return m, m.tickCmd()
	case animTickMsg:
		m.animFrame++
		return m, m.animTickCmd()
	case cursor.BlinkMsg:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch m.mode {
		case ModeNormal:
			return m.updateNormal(msg)
		case ModeAdding:
			return m.updateAdding(msg)
		case ModeEditing:
			return m.updateEditing(msg)
		case ModeSearching:
			return m.updateSearching(msg)
		case ModeConfirmDelete:
			return m.updateConfirmDelete(msg)
		case ModePickStatus:
			return m.updatePickStatus(msg)
		case ModePickPriority:
			return m.updatePickPriority(msg)
		case ModePickVault:
			return m.updatePickVault(msg)
		case ModeVaultAdd:
			return m.updateVaultAdd(msg)
		case ModeVaultConfirmRemove:
			return m.updateVaultConfirmRemove(msg)
		case ModeVaultRemoteToken:
			return m.updateVaultRemoteToken(msg)
		case ModeVaultRemoteName:
			return m.updateVaultRemoteName(msg)
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
		m.input.Placeholder = "Buy milk #groceries !high due:today @work"
		m.input.SetValue("")
		return m, m.input.Focus()
	case "e":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			t := m.filtered[m.selected]
			m.editingID = t.ID
			m.mode = ModeEditing
			m.input.Placeholder = "Buy milk #groceries !high due:today"
			m.input.SetValue(t.ToInputString())
			m.input.CursorEnd()
			return m, m.input.Focus()
		}
	case "A":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			t := m.filtered[m.selected]
			newArchived := !t.Archived
			if err := m.backend.UpdateArchived(t.ID, newArchived); err != nil {
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
			if err := m.backend.UpdateStatus(t.ID, newStatus); err != nil {
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
			for i, p := range pickerPriorities {
				if p == t.Priority {
					m.pickerIdx = i
					break
				}
			}
			m.mode = ModePickPriority
		}
	case "s":
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			t := m.filtered[m.selected]
			for i, s := range pickerStatuses {
				if s == t.Status {
					m.pickerIdx = i
					break
				}
			}
			m.mode = ModePickStatus
		}
	case "/":
		m.mode = ModeSearching
		m.input.Placeholder = "Search tasks..."
		m.input.SetValue("")
		return m, m.input.Focus()
	case "v":
		if m.vault != nil {
			m.openVaultPicker()
			m.mode = ModePickVault
		}
	}
	return m, nil
}

func (m Model) updateAdding(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			title, tags, priority, dueDate, vaultName := task.ParseInput(val)
			if title != "" {
				t := task.Task{
					Title:     title,
					Tags:      tags,
					Priority:  priority,
					DueDate:   dueDate,
					CreatedAt: time.Now(),
				}
				backend := m.backend
				targetVault := m.vault.Active
				openedAlt := false
				if vaultName != "" {
					if v := m.vault.Get(vaultName); v != nil {
						if vaultName != m.vault.Active {
							if altBackend, err := storage.NewBackend(v.IsRemote(), v.Path, v.Token); err == nil {
								backend = altBackend
								openedAlt = true
							}
						}
						targetVault = vaultName
					} else {
						m.status = fmt.Sprintf("Error: vault %q not found", vaultName)
						m.mode = ModeNormal
						m.input.Blur()
						return m, nil
					}
				}
				if err := backend.AddTask(t); err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				} else if targetVault != m.vault.Active {
					m.status = fmt.Sprintf("Added to [%s]: %s", targetVault, title)
				} else {
					m.status = "Added: " + title
					m.reload()
				}
				if openedAlt {
					_ = backend.Close()
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

func (m Model) updateEditing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			title, tags, priority, dueDate, _ := task.ParseInput(val)
			if title != "" {
				t := task.Task{
					Title:    title,
					Tags:     tags,
					Priority: priority,
					DueDate:  dueDate,
				}
				if err := m.backend.UpdateTask(m.editingID, t); err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				} else {
					m.status = "Updated: " + title
					editedID := m.editingID
					m.reload()
					for i, ft := range m.filtered {
						if ft.ID == editedID {
							m.selected = i
							m.ensureVisible()
							break
						}
					}
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
			if err := m.backend.DeleteTask(t.ID); err != nil {
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

func (m Model) updatePickStatus(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.pickerIdx < len(pickerStatuses)-1 {
			m.pickerIdx++
		}
	case "k", "up":
		if m.pickerIdx > 0 {
			m.pickerIdx--
		}
	case "1", "2", "3", "4", "5":
		idx := int(msg.String()[0] - '1')
		if idx < len(pickerStatuses) {
			m.pickerIdx = idx
			return m.applyPickStatus()
		}
	case "enter":
		return m.applyPickStatus()
	case "esc", "q":
		m.mode = ModeNormal
	}
	return m, nil
}

func (m Model) applyPickStatus() (tea.Model, tea.Cmd) {
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		t := m.filtered[m.selected]
		if err := m.backend.UpdateStatus(t.ID, pickerStatuses[m.pickerIdx]); err != nil {
			m.status = fmt.Sprintf("Error: %v", err)
		} else {
			m.reload()
		}
	}
	m.mode = ModeNormal
	return m, nil
}

func (m Model) updatePickPriority(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.pickerIdx < len(pickerPriorities)-1 {
			m.pickerIdx++
		}
	case "k", "up":
		if m.pickerIdx > 0 {
			m.pickerIdx--
		}
	case "1", "2", "3", "4":
		idx := int(msg.String()[0] - '1')
		if idx < len(pickerPriorities) {
			m.pickerIdx = idx
			return m.applyPickPriority()
		}
	case "enter":
		return m.applyPickPriority()
	case "esc", "q":
		m.mode = ModeNormal
	}
	return m, nil
}

func (m Model) applyPickPriority() (tea.Model, tea.Cmd) {
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		t := m.filtered[m.selected]
		if err := m.backend.UpdatePriority(t.ID, pickerPriorities[m.pickerIdx]); err != nil {
			m.status = fmt.Sprintf("Error: %v", err)
		} else {
			m.reload()
		}
	}
	m.mode = ModeNormal
	return m, nil
}

// openVaultPicker positions pickerIdx on the active vault.
func (m *Model) openVaultPicker() {
	for i, v := range m.vault.Vaults {
		if v.Name == m.vault.Active {
			m.pickerIdx = i
			return
		}
	}
	m.pickerIdx = 0
}

func (m Model) updatePickVault(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	vaults := m.vault.Vaults
	switch msg.String() {
	case "j", "down":
		if m.pickerIdx < len(vaults)-1 {
			m.pickerIdx++
		}
	case "k", "up":
		if m.pickerIdx > 0 {
			m.pickerIdx--
		}
	case "enter", "s":
		return m.applyPickVault()
	case "a":
		m.input.Placeholder = "work, personal, …"
		m.input.SetValue("")
		m.mode = ModeVaultAdd
		return m, m.input.Focus()
	case "r":
		m.remoteVaultToken = ""
		m.input.Placeholder = "leave blank to generate a new token"
		m.input.SetValue("")
		m.mode = ModeVaultRemoteToken
		return m, m.input.Focus()
	case "d":
		if len(vaults) > 0 {
			m.mode = ModeVaultConfirmRemove
		}
	case "esc", "q":
		m.mode = ModeNormal
	}
	return m, nil
}

func (m Model) applyPickVault() (tea.Model, tea.Cmd) {
	vaults := m.vault.Vaults
	if m.pickerIdx >= len(vaults) {
		m.mode = ModeNormal
		return m, nil
	}
	chosen := vaults[m.pickerIdx]
	if chosen.Name == m.vault.Active {
		m.mode = ModeNormal
		return m, nil
	}

	newBackend, err := storage.NewBackend(chosen.IsRemote(), chosen.Path, chosen.Token)
	if err != nil {
		m.status = fmt.Sprintf("Error opening vault: %v", err)
		m.mode = ModeNormal
		return m, nil
	}

	if err := m.vault.Switch(chosen.Name); err != nil {
		_ = newBackend.Close()
		m.status = fmt.Sprintf("Error switching vault: %v", err)
		m.mode = ModeNormal
		return m, nil
	}

	_ = m.backend.Close()
	m.backend = newBackend
	m.status = fmt.Sprintf("Switched to vault %q", chosen.Name)
	m.selected = 0
	m.scrollOffset = 0
	m.search = ""
	m.reload()
	m.mode = ModeNormal
	return m, nil
}

func (m Model) updateVaultAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.input.Value())
		m.input.Blur()
		if name != "" {
			if err := m.vault.Add(name); err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				m.status = fmt.Sprintf("Vault %q created", name)
				for i, v := range m.vault.Vaults {
					if v.Name == strings.ToLower(name) {
						m.pickerIdx = i
						break
					}
				}
			}
		}
		m.mode = ModePickVault
	case "esc":
		m.input.Blur()
		m.mode = ModePickVault
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) updateVaultConfirmRemove(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		vaults := m.vault.Vaults
		if m.pickerIdx < len(vaults) {
			name := vaults[m.pickerIdx].Name
			activeChanged, err := m.vault.Remove(name)
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				m.status = fmt.Sprintf("Vault %q removed", name)
				if activeChanged {
					av := m.vault.ActiveVault()
					newBackend, err := storage.NewBackend(av.IsRemote(), av.Path, av.Token)
					if err != nil {
						m.status = fmt.Sprintf("Error opening vault: %v", err)
					} else {
						_ = m.backend.Close()
						m.backend = newBackend
						m.selected = 0
						m.scrollOffset = 0
						m.search = ""
						m.reload()
					}
				}
				if m.pickerIdx >= len(m.vault.Vaults) {
					m.pickerIdx = len(m.vault.Vaults) - 1
				}
			}
		}
		m.mode = ModePickVault
	case "n", "esc":
		m.mode = ModePickVault
	}
	return m, nil
}

// --- remote vault creation steps ---

func (m Model) updateVaultRemoteToken(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		token := strings.TrimSpace(m.input.Value())
		m.input.Blur()
		if token == "" {
			t, err := storage.CreateRemoteVault()
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
				m.mode = ModePickVault
				return m, nil
			}
			m.remoteVaultToken = t
		} else {
			m.remoteVaultToken = token
		}
		m.input.Placeholder = "vault name, e.g. work"
		m.input.SetValue("")
		m.mode = ModeVaultRemoteName
		return m, m.input.Focus()
	case "esc":
		m.input.Blur()
		m.mode = ModePickVault
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) updateVaultRemoteName(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.input.Value())
		m.input.Blur()
		if name != "" {
			if err := m.vault.AddRemoteVault(name, m.remoteVaultToken); err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				short := m.remoteVaultToken
				if len(short) > 8 {
					short = short[:8] + "…"
				}
				m.status = fmt.Sprintf("Remote vault %q added (token: %s)", name, short)
				for i, v := range m.vault.Vaults {
					if v.Name == strings.ToLower(name) {
						m.pickerIdx = i
						break
					}
				}
			}
		}
		m.mode = ModePickVault
	case "esc":
		m.input.Blur()
		m.mode = ModePickVault
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

// ensure vault package is used
var _ = vault.Config{}

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
