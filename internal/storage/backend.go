package storage

import "clist/internal/task"

// Backend is the abstraction over task storage.
// Two implementations: SQLiteBackend (local file) and RemoteBackend (clist-server HTTP API).
type Backend interface {
	AllTasks() ([]task.Task, error)
	AddTask(t task.Task) error
	UpdateStatus(id int64, status task.Status) error
	UpdatePriority(id int64, priority task.Priority) error
	UpdateArchived(id int64, archived bool) error
	UpdateTask(id int64, t task.Task) error
	DeleteTask(id int64) error
	CompletionStats() ([]DayStat, error)
	Close() error
}

// NewBackend opens the right backend based on vault type.
// Pass isRemote=true with token for a remote vault; isRemote=false with path for a local SQLite vault.
func NewBackend(isRemote bool, path, token string) (Backend, error) {
	if isRemote {
		return NewRemoteBackend(token), nil
	}
	return NewSQLiteBackend(path)
}
