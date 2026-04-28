// Package storage handles persistence of tasks in an embedded SQLite database.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"clist/internal/task"

	_ "modernc.org/sqlite"
)

// AppDir is the OS-appropriate config directory name for CLIst.
const AppDir = "clist"

// DayStat holds the completion count for a single day.
type DayStat struct {
	Date  time.Time
	Count int
}

// dbPath returns the path to the SQLite database file.
func dbPath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create db directory %s: %w", dir, err)
	}
	return filepath.Join(dir, "clist.db"), nil
}

func dataDir() (string, error) {
	if runtime.GOOS == "windows" {
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			profile := os.Getenv("USERPROFILE")
			if profile == "" {
				return "", fmt.Errorf("could not determine APPDATA directory")
			}
			appdata = filepath.Join(profile, "AppData", "Roaming")
		}
		return filepath.Join(appdata, AppDir), nil
	}
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return "", fmt.Errorf("could not determine home directory (set HOME or USERPROFILE)")
	}
	return filepath.Join(home, ".local", "share", AppDir), nil
}

// Open opens (or creates) the SQLite database and ensures the schema exists.
func Open() (*sql.DB, error) {
	path, err := dbPath()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %s: %w", path, err)
	}

	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		return nil, fmt.Errorf("failed to set pragmas: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		priority TEXT NOT NULL DEFAULT 'medium',
		status TEXT NOT NULL DEFAULT 'todo',
		due_date TEXT,
		tags TEXT DEFAULT '',
		project TEXT DEFAULT '',
		created_at TEXT NOT NULL,
		completed_at TEXT,
		notes TEXT DEFAULT '',
		archived INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		return nil, fmt.Errorf("failed to create tasks table: %w", err)
	}

	// Migration: add archived column to existing databases.
	if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN archived INTEGER NOT NULL DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("failed to migrate archived column: %w", err)
		}
	}

	return db, nil
}

// AllTasks returns every task ordered by priority then due date.
func AllTasks(db *sql.DB) ([]task.Task, error) {
	rows, err := db.Query(`
		SELECT id, title, priority, status, due_date, tags, created_at, completed_at, notes, archived
		FROM tasks
		ORDER BY
		  CASE priority
		    WHEN 'critical' THEN 0
		    WHEN 'high' THEN 1
		    WHEN 'medium' THEN 2
		    WHEN 'low' THEN 3
		    ELSE 4
		  END,
		  CASE WHEN due_date IS NULL THEN 1 ELSE 0 END,
		  due_date ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func scanTask(rows *sql.Rows) (task.Task, error) {
	var (
		id           int64
		title        string
		priorityStr  string
		statusStr    string
		dueDateStr   sql.NullString
		tagsStr      sql.NullString
		createdAtStr string
		completedStr sql.NullString
		notes        sql.NullString
		archived     int
	)

	if err := rows.Scan(&id, &title, &priorityStr, &statusStr, &dueDateStr, &tagsStr, &createdAtStr, &completedStr, &notes, &archived); err != nil {
		return task.Task{}, fmt.Errorf("failed to scan task row: %w", err)
	}

	t := task.Task{
		ID:       id,
		Title:    title,
		Priority: task.PriorityFromStr(priorityStr),
		Status:   task.StatusFromStr(statusStr),
		Archived: archived != 0,
		Notes:    notes.String,
	}

	if tagsStr.Valid && tagsStr.String != "" {
		for _, tag := range strings.Split(tagsStr.String, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				t.Tags = append(t.Tags, tag)
			}
		}
	}

	if dueDateStr.Valid && dueDateStr.String != "" {
		if d, err := time.Parse("2006-01-02", dueDateStr.String); err == nil {
			t.DueDate = &d
		}
	}

	t.CreatedAt = parseDateTime(createdAtStr)
	if completedStr.Valid && completedStr.String != "" {
		ct := parseDateTime(completedStr.String)
		t.CompletedAt = &ct
	}

	return t, nil
}

func parseDateTime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Local()
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t
	}
	return time.Now()
}

// AddTask inserts a new task.
func AddTask(db *sql.DB, t task.Task) error {
	var dueDateStr *string
	if t.DueDate != nil {
		s := t.DueDate.Format("2006-01-02")
		dueDateStr = &s
	}

	tagsStr := strings.Join(t.Tags, ",")
	createdAt := t.CreatedAt.Format(time.RFC3339)

	_, err := db.Exec(`
		INSERT INTO tasks (title, priority, status, due_date, tags, created_at, notes, archived)
		VALUES (?, ?, 'todo', ?, ?, ?, ?, 0)
	`, t.Title, task.PriorityToStr(t.Priority), dueDateStr, tagsStr, createdAt, t.Notes)
	if err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}
	return nil
}

// UpdateStatus updates the status (and completed_at if done) of a task.
func UpdateStatus(db *sql.DB, id int64, status task.Status) error {
	var completedAt *string
	if status == task.StatusDone {
		s := time.Now().Format(time.RFC3339)
		completedAt = &s
	}

	_, err := db.Exec(`UPDATE tasks SET status = ?, completed_at = ? WHERE id = ?`,
		task.StatusToStr(status), completedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	return nil
}

// UpdatePriority updates the priority of a task.
func UpdatePriority(db *sql.DB, id int64, priority task.Priority) error {
	_, err := db.Exec(`UPDATE tasks SET priority = ? WHERE id = ?`,
		task.PriorityToStr(priority), id)
	if err != nil {
		return fmt.Errorf("failed to update task priority: %w", err)
	}
	return nil
}

// UpdateArchived sets or clears the archived flag for a task.
func UpdateArchived(db *sql.DB, id int64, archived bool) error {
	v := 0
	if archived {
		v = 1
	}
	_, err := db.Exec(`UPDATE tasks SET archived = ? WHERE id = ?`, v, id)
	if err != nil {
		return fmt.Errorf("failed to update archived: %w", err)
	}
	return nil
}

// DeleteTask removes a task by ID.
func DeleteTask(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

// CompletionStats returns the number of tasks completed per day for the last 7 days.
func CompletionStats(db *sql.DB) ([]DayStat, error) {
	today := time.Now()
	sevenDaysAgo := today.AddDate(0, 0, -6)
	startStr := sevenDaysAgo.Format("2006-01-02")

	rows, err := db.Query(`
		SELECT DATE(completed_at) as day, COUNT(*) as cnt
		FROM tasks
		WHERE status = 'done'
		  AND completed_at IS NOT NULL
		  AND DATE(completed_at) >= ?
		GROUP BY day
		ORDER BY day ASC
	`, startStr)
	if err != nil {
		return nil, fmt.Errorf("failed to query completion stats: %w", err)
	}
	defer rows.Close()

	rawData := make(map[string]int)
	for rows.Next() {
		var dayStr string
		var cnt int
		if err := rows.Scan(&dayStr, &cnt); err != nil {
			return nil, err
		}
		rawData[dayStr] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats := make([]DayStat, 0, 7)
	for i := 0; i < 7; i++ {
		day := sevenDaysAgo.AddDate(0, 0, i)
		stats = append(stats, DayStat{Date: day, Count: rawData[day.Format("2006-01-02")]})
	}

	return stats, nil
}
