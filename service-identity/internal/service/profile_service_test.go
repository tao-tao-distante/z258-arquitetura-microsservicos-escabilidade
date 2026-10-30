package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/auth"
)

func (f *fakeRepo) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeRepo) UpdateName(ctx context.Context, id int64, name string) (*domain.User, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	u, err := f.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	u.Name = name
	return u, nil
}

func (f *fakeRepo) UpdatePassword(ctx context.Context, id int64, hash string) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	u, err := f.FindByID(ctx, id)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}

func seededService(t *testing.T) (*AuthService, *fakeRepo, *domain.User) {
	t.Helper()
	repo := newFakeRepo()
	svc := New(repo, "secret", time.Hour)
	u, _, err := svc.Register(context.Background(), "a@b.com", "password")
	require.NoError(t, err)
	return svc, repo, u
}

func TestProfile(t *testing.T) {
	svc, repo, u := seededService(t)

	got, err := svc.Profile(context.Background(), u.ID)
	require.NoError(t, err)
	require.Equal(t, "a@b.com", got.Email)

	_, err = svc.Profile(context.Background(), 999)
	require.ErrorIs(t, err, domain.ErrNotFound)

	repo.findErr = errors.New("boom")
	_, err = svc.Profile(context.Background(), u.ID)
	require.ErrorContains(t, err, "find profile")
	require.NotErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "trims spaces", input: "  Ana Souza  ", want: "Ana Souza"},
		{name: "accepts unicode up to 100 runes", input: strings.Repeat("ç", 100), want: strings.Repeat("ç", 100)},
		{name: "rejects empty", input: "   ", wantErr: domain.ErrInvalidInput},
		{name: "rejects over 100 runes", input: strings.Repeat("a", 101), wantErr: domain.ErrInvalidInput},
		{name: "rejects control characters", input: "Ana\nSouza", wantErr: domain.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, u := seededService(t)
			got, err := svc.UpdateName(context.Background(), u.ID, tt.input)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got.Name)
		})
	}
}

func TestUpdateNameRepositoryErrors(t *testing.T) {
	svc, repo, u := seededService(t)

	_, err := svc.UpdateName(context.Background(), 999, "Ana")
	require.ErrorIs(t, err, domain.ErrNotFound)

	repo.updateErr = errors.New("boom")
	_, err = svc.UpdateName(context.Background(), u.ID, "Ana")
	require.ErrorContains(t, err, "update name")
	require.NotErrorIs(t, err, domain.ErrNotFound)
}

func TestChangePassword(t *testing.T) {
	tests := []struct {
		name    string
		current string
		next    string
		wantErr error
	}{
		{name: "changes password", current: "password", next: "new-password"},
		{name: "rejects wrong current password", current: "wrong-pass", next: "new-password", wantErr: domain.ErrWrongPassword},
		{name: "rejects short new password", current: "password", next: "short", wantErr: domain.ErrInvalidInput},
		{name: "rejects new password over 72 bytes", current: "password", next: strings.Repeat("a", 73), wantErr: domain.ErrInvalidInput},
		{name: "rejects same password", current: "password", next: "password", wantErr: domain.ErrInvalidInput},
		{name: "rejects empty current password", current: "", next: "new-password", wantErr: domain.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, u := seededService(t)
			err := svc.ChangePassword(context.Background(), u.ID, tt.current, tt.next)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.True(t, auth.CheckPassword(u.PasswordHash, tt.next))
			_, _, err = svc.Login(context.Background(), "a@b.com", tt.next)
			require.NoError(t, err)
		})
	}
}

func TestChangePasswordDependencyErrors(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		svc, _, _ := seededService(t)
		err := svc.ChangePassword(context.Background(), 999, "password", "new-password")
		require.ErrorIs(t, err, domain.ErrNotFound)
	})
	t.Run("find error", func(t *testing.T) {
		svc, repo, u := seededService(t)
		repo.findErr = errors.New("boom")
		err := svc.ChangePassword(context.Background(), u.ID, "password", "new-password")
		require.ErrorContains(t, err, "find profile")
	})
	t.Run("hash error", func(t *testing.T) {
		svc, _, u := seededService(t)
		svc.hashPassword = func(string) (string, error) { return "", errors.New("boom") }
		err := svc.ChangePassword(context.Background(), u.ID, "password", "new-password")
		require.ErrorContains(t, err, "hash password")
	})
	t.Run("update error", func(t *testing.T) {
		svc, repo, u := seededService(t)
		repo.updateErr = errors.New("boom")
		err := svc.ChangePassword(context.Background(), u.ID, "password", "new-password")
		require.ErrorContains(t, err, "update password")
	})
}
