package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
	"github.com/timaogurtzova/gophermart/internal/model"
)

const (
	insertUserQuery = `
		INSERT INTO users (login, password_hash)
		VALUES ($1, $2)
		RETURNING id, login, password_hash, created_at
	`
	insertBalanceQuery = `
		INSERT INTO balances (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`
	selectUserByLoginQuery = `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE login = $1
	`
)

var (
	// ErrDatabaseNotConfigured возвращается, когда репозиторий создаётся без подключения к БД.
	ErrDatabaseNotConfigured = errors.New("database is not configured")
	// ErrUserAlreadyExists возвращается, когда пользователь с таким логином уже зарегистрирован.
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrUserNotFound возвращается, когда пользователь с таким логином не найден.
	ErrUserNotFound = errors.New("user not found")
)

// UserRepository описывает контракт хранилища пользователей.
type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (model.User, error)
	FindByLogin(ctx context.Context, login string) (model.User, error)
}

// UserDBRepository хранит пользователей в PostgreSQL.
type UserDBRepository struct {
	db *sql.DB
}

// NewUserRepository создаёт PostgreSQL-репозиторий пользователей.
func NewUserRepository(db *sql.DB) (*UserDBRepository, error) {
	if db == nil {
		return nil, ErrDatabaseNotConfigured
	}

	return &UserDBRepository{db: db}, nil
}

// Create создаёт пользователя и его накопительный счёт.
func (r *UserDBRepository) Create(ctx context.Context, login, passwordHash string) (model.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.User{}, err
	}
	defer tx.Rollback()

	var user model.User
	err = tx.QueryRowContext(ctx, insertUserQuery, login, passwordHash).
		Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.User{}, ErrUserAlreadyExists
		}

		return model.User{}, err
	}

	if _, err = tx.ExecContext(ctx, insertBalanceQuery, user.ID); err != nil {
		return model.User{}, err
	}

	if err = tx.Commit(); err != nil {
		return model.User{}, err
	}

	return user, nil
}

// FindByLogin возвращает пользователя по логину.
func (r *UserDBRepository) FindByLogin(ctx context.Context, login string) (model.User, error) {
	var user model.User
	err := r.db.QueryRowContext(ctx, selectUserByLoginQuery, login).
		Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err == nil {
		return user, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrUserNotFound
	}

	return model.User{}, err
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
