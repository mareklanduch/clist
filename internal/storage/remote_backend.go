package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"clist/internal/task"
)

const serverBaseURL = "https://clist.sharpapp.net"

// RemoteBackend implements Backend using the clist-server HTTP API.
type RemoteBackend struct {
	token  string
	client *http.Client
}

// NewRemoteBackend creates a RemoteBackend for the given token.
// No connection is made until the first method call.
func NewRemoteBackend(token string) *RemoteBackend {
	return &RemoteBackend{
		token:  token,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateRemoteVault calls POST /api/vaults and returns the access token.
func CreateRemoteVault() (string, error) {
	resp, err := http.Post(serverBaseURL+"/api/vaults", "application/json", http.NoBody)
	if err != nil {
		return "", fmt.Errorf("connect to clist-server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("rate limit reached — try again later")
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("server returned %d", resp.StatusCode)
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if body.Token == "" {
		return "", fmt.Errorf("server returned empty token")
	}
	return body.Token, nil
}

// apiTask is the JSON shape used by the clist-server API.
type apiTask struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	Priority    string   `json:"priority"`
	Status      string   `json:"status"`
	Archived    bool     `json:"archived"`
	DueDate     *string  `json:"dueDate"`
	Tags        []string `json:"tags"`
	Notes       string   `json:"notes"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
	CompletedAt *string  `json:"completedAt"`
}

func (r *RemoteBackend) tasksURL() string {
	return fmt.Sprintf("%s/api/vaults/%s/tasks", serverBaseURL, r.token)
}

func (r *RemoteBackend) taskURL(id int64) string {
	return fmt.Sprintf("%s/api/vaults/%s/tasks/%d", serverBaseURL, r.token, id)
}

func (r *RemoteBackend) AllTasks() ([]task.Task, error) {
	resp, err := r.client.Get(r.tasksURL())
	if err != nil {
		return nil, fmt.Errorf("fetch tasks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("vault token not found on server")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}
	var raw []apiTask
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode tasks: %w", err)
	}
	tasks := make([]task.Task, 0, len(raw))
	for _, at := range raw {
		tasks = append(tasks, apiTaskToTask(at))
	}
	return tasks, nil
}

func (r *RemoteBackend) AddTask(t task.Task) error {
	body := map[string]any{
		"title":    t.Title,
		"priority": task.PriorityToStr(t.Priority),
		"status":   statusToAPI(t.Status),
		"tags":     t.Tags,
		"notes":    t.Notes,
	}
	if t.DueDate != nil {
		body["dueDate"] = t.DueDate.Format("2006-01-02")
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := r.client.Post(r.tasksURL(), "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("add task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

// TasksSince returns tasks whose UpdatedAt is after the given UTC time.
func (r *RemoteBackend) TasksSince(since time.Time) ([]task.Task, error) {
	url := r.tasksURL() + "?since=" + since.UTC().Format(time.RFC3339)
	resp, err := r.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch updates: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("vault token not found on server")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}
	var raw []apiTask
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode tasks: %w", err)
	}
	tasks := make([]task.Task, 0, len(raw))
	for _, at := range raw {
		tasks = append(tasks, apiTaskToTask(at))
	}
	return tasks, nil
}

func (r *RemoteBackend) UpdateStatus(id int64, status task.Status) error {
	return r.patch(id, map[string]any{"status": statusToAPI(status)})
}

func (r *RemoteBackend) UpdatePriority(id int64, priority task.Priority) error {
	return r.patch(id, map[string]any{"priority": task.PriorityToStr(priority)})
}

func (r *RemoteBackend) UpdateArchived(id int64, archived bool) error {
	return r.patch(id, map[string]any{"archived": archived})
}

func (r *RemoteBackend) UpdateTask(id int64, t task.Task) error {
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}
	body := map[string]any{
		"title":    t.Title,
		"priority": task.PriorityToStr(t.Priority),
		"tags":     tags,
	}
	if t.DueDate != nil {
		body["dueDate"] = t.DueDate.Format("2006-01-02")
	} else {
		body["clearDueDate"] = true
	}
	return r.patch(id, body)
}

func (r *RemoteBackend) DeleteTask(id int64) error {
	req, err := http.NewRequest(http.MethodDelete, r.taskURL(id), nil)
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

// CompletionStats computes 7-day completion stats from the task list.
// The clist-server has no dedicated stats endpoint.
func (r *RemoteBackend) CompletionStats() ([]DayStat, error) {
	tasks, err := r.AllTasks()
	if err != nil {
		return nil, err
	}
	today := time.Now()
	sevenDaysAgo := today.AddDate(0, 0, -6)
	rawData := make(map[string]int)
	for _, t := range tasks {
		if t.Status == task.StatusDone && t.CompletedAt != nil && !t.CompletedAt.Before(sevenDaysAgo) {
			rawData[t.CompletedAt.Format("2006-01-02")]++
		}
	}
	stats := make([]DayStat, 0, 7)
	for i := range 7 {
		day := sevenDaysAgo.AddDate(0, 0, i)
		stats = append(stats, DayStat{Date: day, Count: rawData[day.Format("2006-01-02")]})
	}
	return stats, nil
}

func (r *RemoteBackend) Close() error { return nil }

func (r *RemoteBackend) patch(id int64, body map[string]any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, r.taskURL(id), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("patch task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

func apiTaskToTask(at apiTask) task.Task {
	t := task.Task{
		ID:       at.ID,
		Title:    at.Title,
		Priority: task.PriorityFromStr(at.Priority),
		Status:   statusFromAPI(at.Status),
		Archived: at.Archived,
		Tags:     at.Tags,
		Notes:    at.Notes,
	}
	if at.DueDate != nil {
		if d, err := time.Parse("2006-01-02", *at.DueDate); err == nil {
			t.DueDate = &d
		}
	}
	if at.CreatedAt != "" {
		t.CreatedAt = parseAPITime(at.CreatedAt)
	}
	if at.CompletedAt != nil {
		ct := parseAPITime(*at.CompletedAt)
		t.CompletedAt = &ct
	}
	if t.Tags == nil {
		t.Tags = []string{}
	}
	return t
}

// statusToAPI maps the local Status string to what the server expects.
// The server uses "in-progress"; the local DB uses "inprogress".
func statusToAPI(s task.Status) string {
	if s == task.StatusInProgress {
		return "in-progress"
	}
	return task.StatusToStr(s)
}

// statusFromAPI maps the server's status string to a local Status.
func statusFromAPI(s string) task.Status {
	if s == "in-progress" {
		return task.StatusInProgress
	}
	return task.StatusFromStr(s)
}

func parseAPITime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Local()
	}
	return time.Now()
}
