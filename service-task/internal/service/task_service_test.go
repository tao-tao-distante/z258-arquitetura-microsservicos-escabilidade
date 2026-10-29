package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
)

type fakeRepo struct {
	tasks map[int64]*domain.Task
	next  int64
	err   error
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

func (f *fakeRepo) List(ctx context.Context, userID int64, limit, offset int) ([]domain.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []domain.Task
	for _, t := range f.tasks {
		if t.UserID == userID {
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

func TestCreateSuccess(t *testing.T) {
	svc := New(newFakeRepo())
	task, err := svc.Create(context.Background(), 1, "title", "desc", "doing")
	require.NoError(t, err)
	require.Equal(t, "doing", task.Status)
}

func TestCreateDefaultsStatus(t *testing.T) {
	svc := New(newFakeRepo())
	task, err := svc.Create(context.Background(), 1, "title", "desc", "")
	require.NoError(t, err)
	require.Equal(t, "todo", task.Status)
}

func TestCreateRejectsInvalidStatus(t *testing.T) {
	svc := New(newFakeRepo())
	_, err := svc.Create(context.Background(), 1, "title", "desc", "blocked")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestCreateValidation(t *testing.T) {
	svc := New(newFakeRepo())
	_, err := svc.Create(context.Background(), 1, "  ", "desc", "")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestCreateRepoError(t *testing.T) {
	repo := newFakeRepo()
	repo.err = errors.New("boom")
	svc := New(repo)
	_, err := svc.Create(context.Background(), 1, "title", "", "")
	require.Error(t, err)
}

func TestList(t *testing.T) {
	svc := New(newFakeRepo())
	_, err := svc.Create(context.Background(), 1, "a", "", "")
	require.NoError(t, err)
	tasks, err := svc.List(context.Background(), 1, 50, 0)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
}

func TestListError(t *testing.T) {
	repo := newFakeRepo()
	repo.err = errors.New("boom")
	svc := New(repo)
	_, err := svc.List(context.Background(), 1, 50, 0)
	require.Error(t, err)
}

func TestGet(t *testing.T) {
	svc := New(newFakeRepo())
	created, err := svc.Create(context.Background(), 1, "a", "", "")
	require.NoError(t, err)
	got, err := svc.Get(context.Background(), created.ID, 1)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
}

func TestGetNotFound(t *testing.T) {
	svc := New(newFakeRepo())
	_, err := svc.Get(context.Background(), 99, 1)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdate(t *testing.T) {
	svc := New(newFakeRepo())
	created, err := svc.Create(context.Background(), 1, "a", "", "")
	require.NoError(t, err)
	updated, err := svc.Update(context.Background(), created.ID, 1, "b", "d", "done")
	require.NoError(t, err)
	require.Equal(t, "b", updated.Title)
	require.Equal(t, "done", updated.Status)
}

func TestUpdateValidation(t *testing.T) {
	svc := New(newFakeRepo())
	_, err := svc.Update(context.Background(), 1, 1, "", "", "todo")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestUpdateRejectsUnknownStatus(t *testing.T) {
	svc := New(newFakeRepo())
	_, err := svc.Update(context.Background(), 1, 1, "title", "", "arbitrary")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestUpdateAcceptsFrontendDoingStatus(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo)
	created, err := svc.Create(context.Background(), 1, "title", "", "")
	require.NoError(t, err)
	updated, err := svc.Update(context.Background(), created.ID, 1, "title", "", "doing")
	require.NoError(t, err)
	require.Equal(t, "doing", updated.Status)
}

func TestUpdateDefaultsStatus(t *testing.T) {
	svc := New(newFakeRepo())
	created, err := svc.Create(context.Background(), 1, "a", "", "")
	require.NoError(t, err)
	updated, err := svc.Update(context.Background(), created.ID, 1, "b", "", "")
	require.NoError(t, err)
	require.Equal(t, "todo", updated.Status)
}

func TestUpdateNotFound(t *testing.T) {
	svc := New(newFakeRepo())
	_, err := svc.Update(context.Background(), 99, 1, "a", "", "todo")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDelete(t *testing.T) {
	svc := New(newFakeRepo())
	created, err := svc.Create(context.Background(), 1, "a", "", "")
	require.NoError(t, err)
	require.NoError(t, svc.Delete(context.Background(), created.ID, 1))
}

func TestDeleteNotFound(t *testing.T) {
	svc := New(newFakeRepo())
	require.ErrorIs(t, svc.Delete(context.Background(), 99, 1), domain.ErrNotFound)
}
