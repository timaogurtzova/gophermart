package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/timaogurtzova/gophermart/internal/model"
)

const selectBalanceByUserIDQuery = `
	SELECT user_id, current_balance, withdrawn_total, updated_at
	FROM balances
	WHERE user_id = $1
`

// ErrBalanceNotFound возвращается, когда накопительный счёт пользователя не найден.
var ErrBalanceNotFound = errors.New("balance not found")

// BalanceRepository описывает контракт хранилища накопительных счетов.
type BalanceRepository interface {
	GetByUserID(ctx context.Context, userID int64) (model.Balance, error)
}

// BalanceDBRepository хранит накопительные счета пользователей в PostgreSQL.
type BalanceDBRepository struct {
	db *sql.DB
}

// NewBalanceRepository создаёт PostgreSQL-репозиторий накопительных счетов.
func NewBalanceRepository(db *sql.DB) (*BalanceDBRepository, error) {
	if db == nil {
		return nil, ErrDatabaseNotConfigured
	}

	return &BalanceDBRepository{db: db}, nil
}

// GetByUserID возвращает накопительный счёт пользователя.
func (r *BalanceDBRepository) GetByUserID(ctx context.Context, userID int64) (model.Balance, error) {
	var balance model.Balance

	err := r.db.QueryRowContext(ctx, selectBalanceByUserIDQuery, userID).
		Scan(&balance.UserID, &balance.Current, &balance.Withdrawn, &balance.UpdatedAt)
	if err == nil {
		return balance, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return model.Balance{}, ErrBalanceNotFound
	}

	return model.Balance{}, err
}
