package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
)

// UserRepository define a persistência de usuários.
type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id int64) (*domain.User, error)
	UpdateName(ctx context.Context, id int64, name string) (*domain.User, error)
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
}

const userColumns = `id, email, name, password_hash, created_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*domain.User, error) {
	var u domain.User
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

// PostgresUserRepository implementa UserRepository com PostgreSQL.
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository cria o repositório com o pool informado.
func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// Create insere um usuário e devolve o registro criado.
func (r *PostgresUserRepository) Create(ctx context.Context, email, passwordHash string) (*domain.User, error) {
	const query = `INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING ` + userColumns
	u, err := scanUser(r.db.QueryRowContext(ctx, query, email, passwordHash))
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, domain.ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// FindByEmail busca um usuário pelo e-mail.
func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE email = $1`
	return findOne(r.db.QueryRowContext(ctx, query, email), "find user")
}

// FindByID busca um usuário pelo identificador.
func (r *PostgresUserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	return findOne(r.db.QueryRowContext(ctx, query, id), "find user by id")
}

// UpdateName altera o nome de exibição e devolve o usuário atualizado.
func (r *PostgresUserRepository) UpdateName(ctx context.Context, id int64, name string) (*domain.User, error) {
	const query = `UPDATE users SET name = $2 WHERE id = $1 RETURNING ` + userColumns
	return findOne(r.db.QueryRowContext(ctx, query, id, name), "update user name")
}

// UpdatePassword substitui o hash de senha do usuário.
func (r *PostgresUserRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	const query = `UPDATE users SET password_hash = $2 WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id, passwordHash)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update user password rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func findOne(row rowScanner, op string) (*domain.User, error) {
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return u, nil
}

// Migrate cria a tabela users e adiciona colunas novas de forma idempotente.
func Migrate(ctx context.Context, db *sql.DB) error {
	const ddl = `CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrate users: %w", err)
	}
	const addName = `ALTER TABLE users ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT ''`
	if _, err := db.ExecContext(ctx, addName); err != nil {
		return fmt.Errorf("migrate users name column: %w", err)
	}
	return nil
}
