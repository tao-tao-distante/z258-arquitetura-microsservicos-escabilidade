package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/auth"
)

type fakeRepo struct {
	users     map[string]*domain.User
	next      int64
	err       error
	createErr error
	findErr   error
}

func newFakeRepo() *fakeRepo { return &fakeRepo{users: map[string]*domain.User{}} }

func (f *fakeRepo) Create(ctx context.Context, email, hash string) (*domain.User, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.err != nil {
		return nil, f.err
	}
	if _, ok := f.users[email]; ok {
		return nil, errors.New("duplicate")
	}
	f.next++
	u := &domain.User{ID: f.next, Email: email, PasswordHash: hash, CreatedAt: time.Now()}
	f.users[email] = u
	return u, nil
}

func (f *fakeRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.err != nil {
		return nil, f.err
	}
	u, ok := f.users[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func TestRegisterSuccess(t *testing.T) {
	svc := New(newFakeRepo(), "secret", time.Hour)
	u, token, err := svc.Register(context.Background(), " A@B.COM ", "password")
	require.NoError(t, err)
	require.Equal(t, "a@b.com", u.Email)
	require.NotEmpty(t, token)
	claims, err := auth.ParseToken("secret", token)
	require.NoError(t, err)
	require.Equal(t, u.ID, claims.UserID)
}

func TestRegisterDuplicate(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, "secret", time.Hour)
	_, _, err := svc.Register(context.Background(), "a@b.com", "password")
	require.NoError(t, err)
	_, _, err = svc.Register(context.Background(), "A@B.COM", "password")
	require.ErrorIs(t, err, domain.ErrEmailTaken)
}

func TestRegisterRejectsInvalidCredentialsInput(t *testing.T) {
	tests := []struct{ email, password string }{
		{"invalid", "password"},
		{"a@b.com", "short"},
		{"a@b.com", string(make([]byte, 73))},
	}
	for _, tt := range tests {
		svc := New(newFakeRepo(), "secret", time.Hour)
		_, _, err := svc.Register(context.Background(), tt.email, tt.password)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	}
}

func TestRegisterMapsCreateConflict(t *testing.T) {
	repo := newFakeRepo()
	repo.users["a@b.com"] = &domain.User{ID: 1, Email: "a@b.com"}
	repo.createErr = domain.ErrEmailTaken
	svc := New(repo, "secret", time.Hour)
	_, _, err := svc.Register(context.Background(), "new@b.com", "password")
	require.ErrorIs(t, err, domain.ErrEmailTaken)
}

func TestRegisterRepoError(t *testing.T) {
	repo := newFakeRepo()
	repo.createErr = errors.New("boom")
	svc := New(repo, "secret", time.Hour)
	_, _, err := svc.Register(context.Background(), "a@b.com", "password")
	require.Error(t, err)
}

func TestRegisterFindByEmailError(t *testing.T) {
	repo := newFakeRepo()
	repo.findErr = errors.New("boom")
	svc := New(repo, "secret", time.Hour)
	_, _, err := svc.Register(context.Background(), "a@b.com", "password")
	require.Error(t, err)
}

func TestRegisterHashError(t *testing.T) {
	svc := New(newFakeRepo(), "secret", time.Hour)
	svc.hashPassword = func(string) (string, error) { return "", errors.New("boom") }
	_, _, err := svc.Register(context.Background(), "a@b.com", "password")
	require.Error(t, err)
}

func TestRegisterTokenError(t *testing.T) {
	svc := New(newFakeRepo(), "secret", time.Hour)
	svc.generateToken = func(string, int64, string, time.Duration) (string, error) {
		return "", errors.New("boom")
	}
	_, _, err := svc.Register(context.Background(), "a@b.com", "password")
	require.ErrorContains(t, err, "generate token")
}

func TestLoginSuccess(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, "secret", time.Hour)
	_, _, err := svc.Register(context.Background(), "a@b.com", "password")
	require.NoError(t, err)
	_, token, err := svc.Login(context.Background(), "A@B.COM", "password")
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestLoginInvalidPassword(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, "secret", time.Hour)
	_, _, err := svc.Register(context.Background(), "a@b.com", "password")
	require.NoError(t, err)
	_, _, err = svc.Login(context.Background(), "a@b.com", "badpassword")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLoginUserNotFound(t *testing.T) {
	svc := New(newFakeRepo(), "secret", time.Hour)
	_, _, err := svc.Login(context.Background(), "none@b.com", "password")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLoginRepoError(t *testing.T) {
	repo := newFakeRepo()
	repo.err = errors.New("boom")
	svc := New(repo, "secret", time.Hour)
	_, _, err := svc.Login(context.Background(), "a@b.com", "password")
	require.Error(t, err)
}

func TestLoginRejectsInvalidInput(t *testing.T) {
	svc := New(newFakeRepo(), "secret", time.Hour)
	_, _, err := svc.Login(context.Background(), "invalid", "password")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestLoginTokenError(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, "secret", time.Hour)
	_, _, err := svc.Register(context.Background(), "a@b.com", "password")
	require.NoError(t, err)
	svc.generateToken = func(string, int64, string, time.Duration) (string, error) {
		return "", errors.New("boom")
	}
	_, _, err = svc.Login(context.Background(), "a@b.com", "password")
	require.ErrorContains(t, err, "generate token")
}
