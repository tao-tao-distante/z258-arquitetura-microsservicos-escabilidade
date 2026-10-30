package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/repository"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/auth"
)

// AuthService implementa registro e login com hash bcrypt e JWT.
type AuthService struct {
	repo          repository.UserRepository
	secret        string
	tokenTTL      time.Duration
	hashPassword  func(string) (string, error)
	generateToken func(string, int64, string, time.Duration) (string, error)
}

// New cria o serviço de autenticação com repositório, segredo JWT e TTL.
func New(repo repository.UserRepository, secret string, ttl time.Duration) *AuthService {
	return &AuthService{
		repo:          repo,
		secret:        secret,
		tokenTTL:      ttl,
		hashPassword:  auth.HashPassword,
		generateToken: auth.GenerateToken,
	}
}

// Register valida as credenciais, cria o usuário e emite um JWT.
func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, string, error) {
	email, err := normalizeCredentials(email, password)
	if err != nil {
		return nil, "", err
	}
	if _, err := s.repo.FindByEmail(ctx, email); err == nil {
		return nil, "", domain.ErrEmailTaken
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, "", fmt.Errorf("check existing user: %w", err)
	}

	hash, err := s.hashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}
	user, err := s.repo.Create(ctx, email, hash)
	if errors.Is(err, domain.ErrEmailTaken) {
		return nil, "", domain.ErrEmailTaken
	}
	if err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}
	token, err := s.generateToken(s.secret, user.ID, user.Email, s.tokenTTL)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}
	return user, token, nil
}

// Login confere as credenciais e emite um JWT.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, string, error) {
	email, err := normalizeCredentials(email, password)
	if err != nil {
		return nil, "", err
	}
	user, err := s.repo.FindByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, "", domain.ErrInvalidCredentials
	}
	if err != nil {
		return nil, "", fmt.Errorf("find user: %w", err)
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return nil, "", domain.ErrInvalidCredentials
	}
	token, err := s.generateToken(s.secret, user.ID, user.Email, s.tokenTTL)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}
	return user, token, nil
}

func normalizeCredentials(email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(email) > 254 || !validPassword(password) {
		return "", domain.ErrInvalidInput
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", domain.ErrInvalidInput
	}
	return email, nil
}

func validPassword(password string) bool {
	return len(password) >= 8 && len(password) <= 72
}
