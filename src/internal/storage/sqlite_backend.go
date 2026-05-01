package storage

import (
	"database/sql"

	"clist/internal/task"
)

// SQLiteBackend implements Backend using a local SQLite database.
type SQLiteBackend struct {
	db *sql.DB
}

// NewSQLiteBackend opens (or creates) a SQLite database at path.
func NewSQLiteBackend(path string) (*SQLiteBackend, error) {
	db, err := OpenAt(path)
	if err != nil {
		return nil, err
	}
	return &SQLiteBackend{db: db}, nil
}

func (s *SQLiteBackend) AllTasks() ([]task.Task, error) { return AllTasks(s.db) }
func (s *SQLiteBackend) AddTask(t task.Task) error      { return AddTask(s.db, t) }
func (s *SQLiteBackend) UpdateStatus(id int64, st task.Status) error {
	return UpdateStatus(s.db, id, st)
}
func (s *SQLiteBackend) UpdatePriority(id int64, p task.Priority) error {
	return UpdatePriority(s.db, id, p)
}
func (s *SQLiteBackend) UpdateArchived(id int64, archived bool) error {
	return UpdateArchived(s.db, id, archived)
}
func (s *SQLiteBackend) UpdateTask(id int64, t task.Task) error {
	return UpdateTask(s.db, id, t)
}
func (s *SQLiteBackend) DeleteTask(id int64) error           { return DeleteTask(s.db, id) }
func (s *SQLiteBackend) CompletionStats() ([]DayStat, error) { return CompletionStats(s.db) }
func (s *SQLiteBackend) Close() error                        { return s.db.Close() }
