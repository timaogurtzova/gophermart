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

const withdrawBalanceQuery = `
	WITH updated_balance AS (
		UPDATE balances
		SET current_balance = current_balance - $3,
			withdrawn_total = withdrawn_total + $3,
			updated_at = NOW()
		WHERE user_id = $1
		  AND current_balance >= $3
		RETURNING user_id
	)
	INSERT INTO withdrawals (user_id, order_number, sum)
	SELECT user_id, $2, $3
	FROM updated_balance
	RETURNING id
`

const selectWithdrawalsByUserIDQuery = `
	SELECT id, user_id, order_number, sum, processed_at
	FROM withdrawals
	WHERE user_id = $1
	ORDER BY processed_at DESC
`

var (
	// ErrBalanceNotFound возвращается, когда накопительный счёт пользователя не найден.
	ErrBalanceNotFound = errors.New("balance not found")
	// ErrInsufficientBalance возвращается, когда баллов недостаточно для списания.
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// BalanceRepository описывает контракт хранилища накопительных счетов.
type BalanceRepository interface {
	GetByUserID(ctx context.Context, userID int64) (model.Balance, error)
	Withdraw(ctx context.Context, userID int64, orderNumber string, sum model.Points) error
	FindWithdrawalsByUserID(ctx context.Context, userID int64) ([]model.Withdrawal, error)
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

// Withdraw списывает баллы и сохраняет факт списания.
func (r *BalanceDBRepository) Withdraw(ctx context.Context, userID int64, orderNumber string, sum model.Points) error {
	var id int64
	err := r.db.QueryRowContext(ctx, withdrawBalanceQuery, userID, orderNumber, sum).Scan(&id)
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrInsufficientBalance
	}

	return err
}

// FindWithdrawalsByUserID возвращает списания пользователя от новых к старым.
func (r *BalanceDBRepository) FindWithdrawalsByUserID(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	rows, err := r.db.QueryContext(ctx, selectWithdrawalsByUserIDQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	withdrawals := make([]model.Withdrawal, 0)
	for rows.Next() {
		var withdrawal model.Withdrawal
		if err := rows.Scan(
			&withdrawal.ID,
			&withdrawal.UserID,
			&withdrawal.OrderNumber,
			&withdrawal.Sum,
			&withdrawal.ProcessedAt,
		); err != nil {
			return nil, err
		}

		withdrawals = append(withdrawals, withdrawal)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return withdrawals, nil
}
