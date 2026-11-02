package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
)

type fakeRepo struct {
	tasks   map[int64]*domain.Task
	next    int64
	err     error
	query   domain.ListQuery
	history []domain.HistoryEntry
}

func newFakeRepo() *fakeRepo { return &fakeRepo{tasks: map[int64]*domain.Task{}} }

func (f *fakeRepo) Create(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.next++
	task.ID = f.next
	cp := *task
	f.tasks[task.ID] = &cp
	return &cp, nil
}

func (f *fakeRepo) List(ctx context.Context, q domain.ListQuery) ([]domain.Task, error) {
	f.query = q
	if f.err != nil {
		return nil, f.err
	}
	var out []domain.Task
	for _, t := range f.tasks {
		if t.UserID == q.UserID {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (f *fakeRepo) Get(ctx context.Context, id, userID int64) (*domain.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	t, ok := f.tasks[id]
	if !ok || t.UserID != userID {
		return nil, domain.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (f *fakeRepo) Update(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	existing, ok := f.tasks[task.ID]
	if !ok || existing.UserID != task.UserID {
		return nil, domain.ErrNotFound
	}
	cp := *task
	f.tasks[task.ID] = &cp
	return &cp, nil
}

func (f *fakeRepo) Delete(ctx context.Context, id, userID int64) error {
	if f.err != nil {
		return f.err
	}
	t, ok := f.tasks[id]
	if !ok || t.UserID != userID {
		return domain.ErrNotFound
	}
	delete(f.tasks, id)
	return nil
}

func (f *fakeRepo) Move(ctx context.Context, move domain.Move) (*domain.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	t, ok := f.tasks[move.ID]
	if !ok || t.UserID != move.UserID {
		return nil, domain.ErrNotFound
	}
	t.Status = move.Status
	cp := *t
	return &cp, nil
}

func (f *fakeRepo) History(ctx context.Context, taskID, userID int64, limit, offset int) ([]domain.HistoryEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	if _, err := f.Get(ctx, taskID, userID); err != nil {
		return nil, err
	}
	return f.history, nil
}

// today é 2026-10-30 em UTC; o piso aceito para prazos é 2026-10-29.
var today = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func newService(repo *fakeRepo) *TaskService {
	svc := New(repo)
	svc.now = func() time.Time { return today }
	return svc
}

func date(raw string) *domain.Date {
	d, err := domain.ParseDate(raw)
	if err != nil {
		panic(err)
	}
	return &d
}

func TestNewUsesWallClock(t *testing.T) {
	require.NotNil(t, New(newFakeRepo()).now)
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name         string
		in           domain.TaskInput
		wantStatus   string
		wantPriority string
		wantErr      error
	}{
		{name: "explicit status and priority", in: domain.TaskInput{Title: "t", Status: "doing", Priority: "urgent"}, wantStatus: "doing", wantPriority: "urgent"},
		{name: "defaults status and priority", in: domain.TaskInput{Title: " t "}, wantStatus: "todo", wantPriority: "medium"},
		{name: "trims priority", in: domain.TaskInput{Title: "t", Priority: " low "}, wantStatus: "todo", wantPriority: "low"},
		{name: "future due date", in: domain.TaskInput{Title: "t", DueDate: date("2026-11-15")}, wantStatus: "todo", wantPriority: "medium"},
		{name: "due date today", in: domain.TaskInput{Title: "t", DueDate: date("2026-10-30")}, wantStatus: "todo", wantPriority: "medium"},
		{name: "due date yesterday tolerated for timezones behind UTC", in: domain.TaskInput{Title: "t", DueDate: date("2026-10-29")}, wantStatus: "todo", wantPriority: "medium"},
		{name: "rejects past due date", in: domain.TaskInput{Title: "t", DueDate: date("2026-10-28")}, wantErr: domain.ErrInvalidInput},
		{name: "rejects empty title", in: domain.TaskInput{Title: "  "}, wantErr: domain.ErrInvalidInput},
		{name: "rejects unknown status", in: domain.TaskInput{Title: "t", Status: "blocked"}, wantErr: domain.ErrInvalidInput},
		{name: "rejects unknown priority", in: domain.TaskInput{Title: "t", Priority: "critical"}, wantErr: domain.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := newService(newFakeRepo()).Create(context.Background(), 1, tt.in)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "t", task.Title)
			require.Equal(t, int64(1), task.UserID)
			require.Equal(t, tt.wantStatus, task.Status)
			require.Equal(t, tt.wantPriority, task.Priority)
			require.True(t, domain.SameDate(tt.in.DueDate, task.DueDate))
		})
	}
}

func TestCreateRepoError(t *testing.T) {
	repo := newFakeRepo()
	repo.err = errors.New("boom")
	_, err := newService(repo).Create(context.Background(), 1, domain.TaskInput{Title: "title"})
	require.Error(t, err)
}

func TestList(t *testing.T) {
	tests := []struct {
		name    string
		in      domain.ListQuery
		want    domain.ListQuery
		wantErr error
	}{
		{
			name: "defaults to board position ascending",
			in:   domain.ListQuery{UserID: 1, Limit: 50},
			want: domain.ListQuery{UserID: 1, Sort: "position", Order: "asc", Limit: 50},
		},
		{
			name: "priority defaults to highest first",
			in:   domain.ListQuery{UserID: 1, Sort: "priority"},
			want: domain.ListQuery{UserID: 1, Sort: "priority", Order: "desc"},
		},
		{
			name: "created_at defaults to newest first",
			in:   domain.ListQuery{UserID: 1, Sort: "created_at"},
			want: domain.ListQuery{UserID: 1, Sort: "created_at", Order: "desc"},
		},
		{
			name: "due_date defaults to closest first",
			in:   domain.ListQuery{UserID: 1, Sort: "due_date"},
			want: domain.ListQuery{UserID: 1, Sort: "due_date", Order: "asc"},
		},
		{
			name: "keeps explicit order and trimmed filters",
			in:   domain.ListQuery{UserID: 1, Status: " done ", Priority: " high ", Overdue: true, Sort: "due_date", Order: "desc"},
			want: domain.ListQuery{UserID: 1, Status: "done", Priority: "high", Overdue: true, Sort: "due_date", Order: "desc"},
		},
		{name: "rejects unknown status", in: domain.ListQuery{Status: "blocked"}, wantErr: domain.ErrInvalidInput},
		{name: "rejects unknown priority", in: domain.ListQuery{Priority: "critical"}, wantErr: domain.ErrInvalidInput},
		{name: "rejects unknown sort", in: domain.ListQuery{Sort: "title; DROP TABLE tasks"}, wantErr: domain.ErrInvalidInput},
		{name: "rejects unknown order", in: domain.ListQuery{Order: "sideways"}, wantErr: domain.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()

			_, err := newService(repo).List(context.Background(), tt.in)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, repo.query)
		})
	}
}

func TestListReturnsUserTasks(t *testing.T) {
	svc := newService(newFakeRepo())
	_, err := svc.Create(context.Background(), 1, domain.TaskInput{Title: "a"})
	require.NoError(t, err)
	tasks, err := svc.List(context.Background(), domain.ListQuery{UserID: 1, Limit: 50})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
}

func TestListError(t *testing.T) {
	repo := newFakeRepo()
	repo.err = errors.New("boom")
	_, err := newService(repo).List(context.Background(), domain.ListQuery{UserID: 1, Limit: 50})
	require.Error(t, err)
}

func TestGet(t *testing.T) {
	svc := newService(newFakeRepo())
	created, err := svc.Create(context.Background(), 1, domain.TaskInput{Title: "a"})
	require.NoError(t, err)
	got, err := svc.Get(context.Background(), created.ID, 1)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
}

func TestGetNotFound(t *testing.T) {
	_, err := newService(newFakeRepo()).Get(context.Background(), 99, 1)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name         string
		existingDue  *domain.Date
		id           int64
		in           domain.TaskInput
		wantStatus   string
		wantPriority string
		wantErr      error
	}{
		{name: "updates fields", id: 1, in: domain.TaskInput{Title: "b", Description: "d", Status: "done", Priority: "high", DueDate: date("2026-11-20")}, wantStatus: "done", wantPriority: "high"},
		{name: "defaults status and priority", id: 1, in: domain.TaskInput{Title: "b"}, wantStatus: "todo", wantPriority: "medium"},
		{name: "keeps an overdue due date unchanged", existingDue: date("2026-10-01"), id: 1, in: domain.TaskInput{Title: "b", DueDate: date("2026-10-01")}, wantStatus: "todo", wantPriority: "medium"},
		{name: "clears an overdue due date", existingDue: date("2026-10-01"), id: 1, in: domain.TaskInput{Title: "b"}, wantStatus: "todo", wantPriority: "medium"},
		{name: "rejects moving due date to the past", existingDue: date("2026-10-01"), id: 1, in: domain.TaskInput{Title: "b", DueDate: date("2026-10-02")}, wantErr: domain.ErrInvalidInput},
		{name: "rejects adding a past due date", id: 1, in: domain.TaskInput{Title: "b", DueDate: date("2026-10-02")}, wantErr: domain.ErrInvalidInput},
		{name: "past due date on missing task", id: 99, in: domain.TaskInput{Title: "b", DueDate: date("2026-10-02")}, wantErr: domain.ErrNotFound},
		{name: "rejects empty title", id: 1, in: domain.TaskInput{Title: ""}, wantErr: domain.ErrInvalidInput},
		{name: "rejects unknown status", id: 1, in: domain.TaskInput{Title: "b", Status: "arbitrary"}, wantErr: domain.ErrInvalidInput},
		{name: "rejects unknown priority", id: 1, in: domain.TaskInput{Title: "b", Priority: "critical"}, wantErr: domain.ErrInvalidInput},
		{name: "not found", id: 99, in: domain.TaskInput{Title: "b"}, wantErr: domain.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.tasks[1] = &domain.Task{ID: 1, UserID: 1, Title: "a", Status: "todo", Priority: "medium", DueDate: tt.existingDue}

			task, err := newService(repo).Update(context.Background(), tt.id, 1, tt.in)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "b", task.Title)
			require.Equal(t, tt.wantStatus, task.Status)
			require.Equal(t, tt.wantPriority, task.Priority)
			require.True(t, domain.SameDate(tt.in.DueDate, task.DueDate))
		})
	}
}

func TestDelete(t *testing.T) {
	svc := newService(newFakeRepo())
	created, err := svc.Create(context.Background(), 1, domain.TaskInput{Title: "a"})
	require.NoError(t, err)
	require.NoError(t, svc.Delete(context.Background(), created.ID, 1))
}

func TestDeleteNotFound(t *testing.T) {
	require.ErrorIs(t, newService(newFakeRepo()).Delete(context.Background(), 99, 1), domain.ErrNotFound)
}

func ptr(v int64) *int64 { return &v }

func TestMove(t *testing.T) {
	tests := []struct {
		name       string
		move       domain.Move
		repoErr    error
		wantStatus string
		wantErr    error
	}{
		{name: "moves to another column", move: domain.Move{ID: 1, UserID: 1, Status: "doing", BeforeID: ptr(2), AfterID: ptr(3)}, wantStatus: "doing"},
		{name: "trims status", move: domain.Move{ID: 1, UserID: 1, Status: " done "}, wantStatus: "done"},
		{name: "rejects empty status", move: domain.Move{ID: 1, UserID: 1, Status: " "}, wantErr: domain.ErrInvalidInput},
		{name: "rejects unknown status", move: domain.Move{ID: 1, UserID: 1, Status: "archived"}, wantErr: domain.ErrInvalidInput},
		{name: "rejects itself as previous", move: domain.Move{ID: 1, UserID: 1, Status: "todo", BeforeID: ptr(1)}, wantErr: domain.ErrInvalidInput},
		{name: "rejects itself as next", move: domain.Move{ID: 1, UserID: 1, Status: "todo", AfterID: ptr(1)}, wantErr: domain.ErrInvalidInput},
		{name: "rejects same neighbors", move: domain.Move{ID: 1, UserID: 1, Status: "todo", BeforeID: ptr(2), AfterID: ptr(2)}, wantErr: domain.ErrInvalidInput},
		{name: "not found", move: domain.Move{ID: 99, UserID: 1, Status: "todo"}, wantErr: domain.ErrNotFound},
		{name: "conflict from repository", move: domain.Move{ID: 1, UserID: 1, Status: "todo"}, repoErr: domain.ErrConflict, wantErr: domain.ErrConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			svc := newService(repo)
			_, err := svc.Create(context.Background(), 1, domain.TaskInput{Title: "a"})
			require.NoError(t, err)
			repo.err = tt.repoErr

			task, err := svc.Move(context.Background(), tt.move)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, task.Status)
		})
	}
}

func TestHistory(t *testing.T) {
	repo := newFakeRepo()
	repo.tasks[1] = &domain.Task{ID: 1, UserID: 1}
	repo.history = []domain.HistoryEntry{{ID: 7, TaskID: 1, Action: domain.ActionMoved}}

	entries, err := newService(repo).History(context.Background(), 1, 1, 20, 0)

	require.NoError(t, err)
	require.Equal(t, repo.history, entries)
}

func TestHistoryErrors(t *testing.T) {
	_, err := newService(newFakeRepo()).History(context.Background(), 99, 1, 20, 0)
	require.ErrorIs(t, err, domain.ErrNotFound)

	repo := newFakeRepo()
	repo.err = errors.New("boom")
	_, err = newService(repo).History(context.Background(), 1, 1, 20, 0)
	require.ErrorIs(t, err, repo.err)
}
