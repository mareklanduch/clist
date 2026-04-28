package tui

import (
	"sort"
	"strings"
	"time"

	"clist/internal/storage"
	"clist/internal/task"
)

func (m *Model) reload() {
	tasks, err := storage.AllTasks(m.db)
	if err != nil {
		m.err = err
		return
	}
	m.err = nil
	m.tasks = tasks

	if stats, err := storage.CompletionStats(m.db); err == nil {
		m.stats = stats
	}

	m.counts = m.computeCounts()
	m.applyFilter()
}

// computeCounts tallies view badge counts in a single pass over all tasks.
// Called only from reload(), so the O(n) scan happens once per data refresh.
func (m *Model) computeCounts() [5]int {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var counts [5]int
	for _, t := range m.tasks {
		if !t.Archived && t.Status != task.StatusDone {
			counts[0]++ // All
		}
		if !t.Archived && t.Status != task.StatusDone {
			createdDate := time.Date(t.CreatedAt.Year(), t.CreatedAt.Month(), t.CreatedAt.Day(), 0, 0, 0, 0, t.CreatedAt.Location())
			if t.IsDueToday() || createdDate.Equal(today) {
				counts[1]++ // Today
			}
		}
		if !t.Archived && t.Status == task.StatusWaiting {
			counts[2]++ // Waiting
		}
		if !t.Archived && t.Status == task.StatusSomeday {
			counts[3]++ // Someday
		}
		if t.Archived {
			counts[4]++ // Archive
		}
	}
	return counts
}

func (m *Model) applyFilter() {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var result []task.Task
	for _, t := range m.tasks {
		switch m.view {
		case ViewAll:
			if t.Archived {
				continue
			}
		case ViewToday:
			if t.Archived {
				continue
			}
			createdDate := time.Date(t.CreatedAt.Year(), t.CreatedAt.Month(), t.CreatedAt.Day(), 0, 0, 0, 0, t.CreatedAt.Location())
			if !t.IsDueToday() && !createdDate.Equal(today) {
				continue
			}
		case ViewWaiting:
			if t.Status != task.StatusWaiting || t.Archived {
				continue
			}
		case ViewSomeday:
			if t.Status != task.StatusSomeday || t.Archived {
				continue
			}
		case ViewStats:
			continue
		case ViewArchive:
			if !t.Archived {
				continue
			}
		}

		if m.search != "" {
			q := strings.ToLower(m.search)
			matched := strings.Contains(strings.ToLower(t.Title), q)
			for _, tag := range t.Tags {
				if strings.Contains(strings.ToLower(tag), q) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		result = append(result, t)
	}

	sort.SliceStable(result, func(i, j int) bool {
		iDone := result[i].Status == task.StatusDone
		jDone := result[j].Status == task.StatusDone
		if iDone != jDone {
			return !iDone
		}
		pi := task.PrioritySortOrder(result[i].Priority)
		pj := task.PrioritySortOrder(result[j].Priority)
		if pi != pj {
			return pi < pj
		}
		if result[i].DueDate == nil && result[j].DueDate == nil {
			return false
		}
		if result[i].DueDate == nil {
			return false
		}
		if result[j].DueDate == nil {
			return true
		}
		return result[i].DueDate.Before(*result[j].DueDate)
	})

	m.filtered = result
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if m.scrollOffset > m.selected {
		m.scrollOffset = m.selected
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// ensureVisible advances scrollOffset until the selected task fits within the
// visible line budget. Accounts for multi-line tasks and the ▲ indicator line.
func (m *Model) ensureVisible() {
	if len(m.filtered) == 0 || m.width == 0 {
		return
	}
	if m.selected < m.scrollOffset {
		m.scrollOffset = m.selected
		return
	}

	// Mirror renderMain's showDetail condition so listW matches the renderer.
	const sidebarW = 26
	const detailW = 30
	detailShown := m.mode == ModeNormal &&
		len(m.filtered) > 0 &&
		m.selected < len(m.filtered) &&
		m.view != ViewStats
	extra := 0
	if detailShown {
		extra = detailW
	}
	// border+padding=4, selection prefix=2
	listW := max(m.width-sidebarW-extra-4-2, 10)
	// m.height - 2 (main box border) - 1 (▼ reserve)
	capacity := max(m.height-3, 1)

	for m.scrollOffset < m.selected {
		effective := capacity
		if m.scrollOffset > 0 {
			effective-- // ▲ takes 1 line
		}
		lineCount := 0
		visible := false
		for i := m.scrollOffset; i < len(m.filtered); i++ {
			lc := len(m.renderTaskLines(m.filtered[i], listW))
			if lineCount+lc > effective {
				break
			}
			lineCount += lc
			if i == m.selected {
				visible = true
				break
			}
		}
		if visible {
			break
		}
		m.scrollOffset++
	}
	// Safety: task taller than viewport — show it at top rather than above it.
	if m.scrollOffset > m.selected {
		m.scrollOffset = m.selected
	}
}
