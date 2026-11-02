package service

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/auth"
)

const maxNameRunes = 100

// Profile devolve os dados do usuário autenticado.
func (s *AuthService) Profile(ctx context.Context, userID int64) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find profile: %w", err)
	}
	return user, nil
}

// UpdateName valida e grava o nome de exibição do usuário.
func (s *AuthService) UpdateName(ctx context.Context, userID int64, name string) (*domain.User, error) {
	name = strings.TrimSpace(name)
	if !validName(name) {
		return nil, domain.ErrInvalidInput
	}
	user, err := s.repo.UpdateName(ctx, userID, name)
	if err != nil {
		return nil, fmt.Errorf("update name: %w", err)
	}
	return user, nil
}

// ChangePassword confere a senha atual e grava o hash da nova senha.
func (s *AuthService) ChangePassword(ctx context.Context, userID int64, current, next string) error {
	if current == "" || !validPassword(next) || current == next {
		return domain.ErrInvalidInput
	}
	user, err := s.Profile(ctx, userID)
	if err != nil {
		return err
	}
	if !auth.CheckPassword(user.PasswordHash, current) {
		return domain.ErrWrongPassword
	}
	hash, err := s.hashPassword(next)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.repo.UpdatePassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func validName(name string) bool {
	if name == "" || utf8.RuneCountInString(name) > maxNameRunes {
		return false
	}
	return strings.IndexFunc(name, unicode.IsControl) < 0
}
