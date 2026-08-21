package maintenance

import (
	"context"
	"errors"
	"example.com/distributed-search-service/internal/index"
	"sync"
	"time"
)

type TaskState string

const (
	TaskPending   TaskState = "pending"
	TaskRunning   TaskState = "running"
	TaskSucceeded TaskState = "succeeded"
	TaskFailed    TaskState = "failed"
	TaskPaused    TaskState = "paused"
)

type Task struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	CollectionID string    `json:"collection_id"`
	State        TaskState `json:"state"`
	Progress     float64   `json:"progress"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
type Service struct {
	mu     sync.RWMutex
	tasks  map[string]Task
	paused bool
}

func New() *Service { return &Service{tasks: map[string]Task{}} }
func (s *Service) ScheduleCompact(ctx context.Context, id, collection string, engine *index.Engine) (Task, error) {
	s.mu.Lock()
	if s.paused {
		s.mu.Unlock()
		return Task{}, errors.New("maintenance is paused")
	}
	if _, ok := s.tasks[id]; ok {
		s.mu.Unlock()
		return Task{}, errors.New("task already exists")
	}
	task := Task{ID: id, Type: "compact", CollectionID: collection, State: TaskPending, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	s.tasks[id] = task
	s.mu.Unlock()
	go s.runCompact(ctx, id, engine)
	return task, nil
}
func (s *Service) runCompact(ctx context.Context, id string, engine *index.Engine) {
	s.update(id, TaskRunning, .1, "")
	select {
	case <-ctx.Done():
		s.update(id, TaskFailed, .1, ctx.Err().Error())
		return
	case <-time.After(10 * time.Millisecond):
	}
	if err := engine.Flush(); err != nil {
		s.update(id, TaskFailed, .5, err.Error())
		return
	}
	s.update(id, TaskSucceeded, 1, "")
}
func (s *Service) update(id string, state TaskState, progress float64, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.tasks[id]
	t.State = state
	t.Progress = progress
	t.Error = message
	t.UpdatedAt = time.Now().UTC()
	s.tasks[id] = t
}
func (s *Service) Get(id string) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	return t, ok
}
func (s *Service) Pause()  { s.mu.Lock(); defer s.mu.Unlock(); s.paused = true }
func (s *Service) Resume() { s.mu.Lock(); defer s.mu.Unlock(); s.paused = false }
