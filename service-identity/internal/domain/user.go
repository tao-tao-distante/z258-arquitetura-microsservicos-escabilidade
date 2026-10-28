package domain

import (
	"errors"
	"time"
)

// Erros de domínio retornados pelo service-identity.
var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInput       = errors.New("invalid input")
)

// User representa um usuário persistido.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
