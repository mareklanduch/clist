// Package task defines the core Task model and parsing logic.
// It is UI- and storage-agnostic: it does not import lipgloss, sql, or bubbletea.
package task

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Priority levels.
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// Status values.
type Status int

const (
	StatusTodo Status = iota
	StatusInProgress
	StatusDone
	StatusWaiting
	StatusSomeday
)

// Task represents a single todo item.
type Task struct {
	ID          int64
	Title       string
	Priority    Priority
	Status      Status
	Archived    bool
	DueDate     *time.Time
	Tags        []string
	CreatedAt   time.Time
	CompletedAt *time.Time
	Notes       string
}

// PriorityIcon returns a 2-char icon for the priority.
func (t Task) PriorityIcon() string {
	switch t.Priority {
	case PriorityCritical:
		return "!!"
	case PriorityHigh:
		return "! "
	case PriorityMedium:
		return "~ "
	default:
		return ". "
	}
}

// PriorityLabel returns the human-readable priority name.
func (t Task) PriorityLabel() string {
	switch t.Priority {
	case PriorityCritical:
		return "Critical"
	case PriorityHigh:
		return "High"
	case PriorityMedium:
		return "Medium"
	default:
		return "Low"
	}
}

// StatusIcon returns a 3-char status icon.
func (t Task) StatusIcon() string {
	switch t.Status {
	case StatusTodo:
		return "[ ]"
	case StatusInProgress:
		return "[>]"
	case StatusDone:
		return "[x]"
	case StatusWaiting:
		return "[~]"
	case StatusSomeday:
		return "[?]"
	default:
		return "[ ]"
	}
}

// StatusLabel returns the human-readable status name.
func (t Task) StatusLabel() string {
	switch t.Status {
	case StatusTodo:
		return "Todo"
	case StatusInProgress:
		return "In Progress"
	case StatusDone:
		return "Done"
	case StatusWaiting:
		return "Waiting"
	case StatusSomeday:
		return "Someday"
	default:
		return "Todo"
	}
}

// IsOverdue returns true if the task has a due date in the past and isn't done.
func (t Task) IsOverdue() bool {
	if t.Status == StatusDone || t.DueDate == nil {
		return false
	}
	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	dueDate := time.Date(t.DueDate.Year(), t.DueDate.Month(), t.DueDate.Day(), 0, 0, 0, 0, t.DueDate.Location())
	return dueDate.Before(todayDate)
}

// IsDueToday returns true if the task is due today and isn't done.
func (t Task) IsDueToday() bool {
	if t.Status == StatusDone || t.DueDate == nil {
		return false
	}
	today := time.Now()
	return t.DueDate.Year() == today.Year() &&
		t.DueDate.Month() == today.Month() &&
		t.DueDate.Day() == today.Day()
}

// DaysDue returns the number of days until (positive) or since (negative) the due date.
func (t Task) DaysDue() int {
	if t.DueDate == nil {
		return 0
	}
	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	dueDate := time.Date(t.DueDate.Year(), t.DueDate.Month(), t.DueDate.Day(), 0, 0, 0, 0, time.Local)
	return int(dueDate.Sub(todayDate).Hours() / 24)
}

// DueDaysStr returns a human-friendly due date string.
func (t Task) DueDaysStr() string {
	if t.DueDate == nil {
		return ""
	}
	days := t.DaysDue()
	switch {
	case days == 0:
		return "today"
	case days < 0:
		return fmt.Sprintf("-%dd overdue", -days)
	default:
		return fmt.Sprintf("+%dd", days)
	}
}

// PriorityFromStr converts a string to a Priority.
func PriorityFromStr(s string) Priority {
	switch strings.ToLower(s) {
	case "critical":
		return PriorityCritical
	case "high":
		return PriorityHigh
	case "low":
		return PriorityLow
	default:
		return PriorityMedium
	}
}

// PriorityToStr converts a Priority to its DB string.
func PriorityToStr(p Priority) string {
	switch p {
	case PriorityCritical:
		return "critical"
	case PriorityHigh:
		return "high"
	case PriorityLow:
		return "low"
	default:
		return "medium"
	}
}

// StatusFromStr converts a string to a Status.
func StatusFromStr(s string) Status {
	switch strings.ToLower(s) {
	case "inprogress", "in_progress", "in-progress":
		return StatusInProgress
	case "done":
		return StatusDone
	case "waiting":
		return StatusWaiting
	case "someday":
		return StatusSomeday
	default:
		return StatusTodo
	}
}

// StatusToStr converts a Status to its DB string.
func StatusToStr(s Status) string {
	switch s {
	case StatusInProgress:
		return "inprogress"
	case StatusDone:
		return "done"
	case StatusWaiting:
		return "waiting"
	case StatusSomeday:
		return "someday"
	default:
		return "todo"
	}
}

// PrioritySortOrder returns sort order where 0 = highest priority.
func PrioritySortOrder(p Priority) int {
	switch p {
	case PriorityCritical:
		return 0
	case PriorityHigh:
		return 1
	case PriorityMedium:
		return 2
	default:
		return 3
	}
}

// ParseInput parses GTD-style task input:
//
//	"Buy milk #groceries !high due:today @work"
//
// vaultName is empty when no @vault token is present.
func ParseInput(s string) (title string, tags []string, priority Priority, dueDate *time.Time, vaultName string) {
	priority = PriorityMedium
	var titleParts []string

	for _, token := range strings.Fields(s) {
		switch {
		case strings.HasPrefix(token, "#") && len(token) > 1:
			tags = append(tags, token[1:])
		case strings.HasPrefix(token, "!") && len(token) > 1:
			priority = PriorityFromStr(token[1:])
		case strings.HasPrefix(token, "due:") && len(token) > 4:
			dueDate = parseDueDate(token[4:])
		case strings.HasPrefix(token, "@") && len(token) > 1:
			vaultName = strings.ToLower(token[1:])
		default:
			titleParts = append(titleParts, token)
		}
	}

	title = strings.Join(titleParts, " ")
	return
}

// ToInputString serializes the task back to the GTD add-syntax string so it can
// be pre-filled in the edit input box.
func (t Task) ToInputString() string {
	parts := []string{t.Title}
	for _, tag := range t.Tags {
		parts = append(parts, "#"+tag)
	}
	if t.Priority != PriorityMedium {
		parts = append(parts, "!"+PriorityToStr(t.Priority))
	}
	if t.DueDate != nil {
		parts = append(parts, "due:"+t.DueDate.Format("2006-01-02"))
	}
	return strings.Join(parts, " ")
}

// parseDueDate accepts: YYYY-MM-DD, today, Nd / -Nd (days), Nm / -Nm (months).
func parseDueDate(s string) *time.Time {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if s == "today" {
		return &today
	}

	if len(s) >= 2 {
		suffix := s[len(s)-1]
		if suffix == 'd' || suffix == 'm' {
			if n, err := strconv.Atoi(s[:len(s)-1]); err == nil {
				var d time.Time
				if suffix == 'd' {
					d = today.AddDate(0, 0, n)
				} else {
					d = today.AddDate(0, n, 0)
				}
				return &d
			}
		}
	}

	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}
	return nil
}
