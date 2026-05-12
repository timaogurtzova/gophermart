package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/timaogurtzova/gophermart/internal/model"
)

const insertOrderQuery = `
	WITH inserted AS (
		INSERT INTO orders (user_id, number, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (number) DO NOTHING
		RETURNING user_id, TRUE AS created
	)
	SELECT user_id, created
	FROM inserted
	UNION ALL
	SELECT user_id, FALSE AS created
	FROM orders
	WHERE number = $2
	  AND NOT EXISTS (SELECT 1 FROM inserted)
	LIMIT 1
`

const selectOrdersByUserIDQuery = `
	SELECT id, user_id, number, status, accrual::text, uploaded_at, updated_at
	FROM orders
	WHERE user_id = $1
	ORDER BY uploaded_at DESC
`

const selectOrdersForProcessingQuery = `
	SELECT id, user_id, number, status, accrual::text, uploaded_at, updated_at
	FROM orders
	WHERE status IN ($1, $2)
	ORDER BY uploaded_at ASC
	LIMIT $3
`

const updateOrderStatusQuery = `
	UPDATE orders
	SET status = $2,
		updated_at = NOW()
	WHERE id = $1
	  AND status IN ($3, $4)
`

const updateOrderProcessedQuery = `
	UPDATE orders
	SET status = $2,
		accrual = $3,
		updated_at = NOW()
	WHERE id = $1
	  AND status IN ($4, $5)
	RETURNING user_id
`

const addBalanceAccrualQuery = `
	UPDATE balances
	SET current_balance = current_balance + $2,
		updated_at = NOW()
	WHERE user_id = $1
`

var (
	// ErrOrderAlreadyUploadedByUser возвращается, когда заказ уже загружен этим пользователем.
	ErrOrderAlreadyUploadedByUser = errors.New("order already uploaded by user")
	// ErrOrderAlreadyUploadedByAnotherUser возвращается, когда заказ уже загружен другим пользователем.
	ErrOrderAlreadyUploadedByAnotherUser = errors.New("order already uploaded by another user")
)

// OrderRepository описывает контракт хранилища заказов.
type OrderRepository interface {
	Upload(ctx context.Context, userID int64, number string) error
	FindByUserID(ctx context.Context, userID int64) ([]model.Order, error)
	FindForProcessing(ctx context.Context, limit int) ([]model.Order, error)
	UpdateStatus(ctx context.Context, orderID int64, status model.OrderStatus) error
	ApplyAccrual(ctx context.Context, orderID int64, accrual *model.Points) error
}

// OrderDBRepository хранит загруженные номера заказов в PostgreSQL.
type OrderDBRepository struct {
	db *sql.DB
}

// NewOrderRepository создаёт PostgreSQL-репозиторий заказов.
func NewOrderRepository(db *sql.DB) (*OrderDBRepository, error) {
	if db == nil {
		return nil, ErrDatabaseNotConfigured
	}

	return &OrderDBRepository{db: db}, nil
}

// Upload сохраняет номер заказа со статусом NEW.
func (r *OrderDBRepository) Upload(ctx context.Context, userID int64, number string) error {
	var ownerID int64
	var created bool
	err := r.db.QueryRowContext(ctx, insertOrderQuery, userID, number, model.OrderStatusNew).Scan(&ownerID, &created)
	if err != nil {
		return err
	}

	if created {
		return nil
	}

	if ownerID == userID {
		return ErrOrderAlreadyUploadedByUser
	}

	return ErrOrderAlreadyUploadedByAnotherUser
}

// FindByUserID возвращает заказы пользователя от новых к старым.
func (r *OrderDBRepository) FindByUserID(ctx context.Context, userID int64) ([]model.Order, error) {
	rows, err := r.db.QueryContext(ctx, selectOrdersByUserIDQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanOrders(rows)
}

// FindForProcessing возвращает заказы, которым ещё нужен расчёт начислений.
func (r *OrderDBRepository) FindForProcessing(ctx context.Context, limit int) ([]model.Order, error) {
	rows, err := r.db.QueryContext(
		ctx,
		selectOrdersForProcessingQuery,
		model.OrderStatusNew,
		model.OrderStatusProcessing,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanOrders(rows)
}

// UpdateStatus обновляет статус заказа без начисления баллов.
func (r *OrderDBRepository) UpdateStatus(ctx context.Context, orderID int64, status model.OrderStatus) error {
	_, err := r.db.ExecContext(
		ctx,
		updateOrderStatusQuery,
		orderID,
		status,
		model.OrderStatusNew,
		model.OrderStatusProcessing,
	)
	return err
}

// ApplyAccrual завершает обработку заказа и начисляет баллы на счёт пользователя.
func (r *OrderDBRepository) ApplyAccrual(ctx context.Context, orderID int64, accrual *model.Points) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var accrualValue any
	if accrual != nil {
		accrualValue = *accrual
	}

	var userID int64
	err = tx.QueryRowContext(
		ctx,
		updateOrderProcessedQuery,
		orderID,
		model.OrderStatusProcessed,
		accrualValue,
		model.OrderStatusNew,
		model.OrderStatusProcessing,
	).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tx.Commit()
		}

		return err
	}

	if accrual != nil && accrual.IsPositive() {
		if _, err = tx.ExecContext(ctx, addBalanceAccrualQuery, userID, *accrual); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func scanOrders(rows *sql.Rows) ([]model.Order, error) {
	orders := make([]model.Order, 0)
	for rows.Next() {
		var order model.Order
		var accrual sql.NullString

		if err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Number,
			&order.Status,
			&accrual,
			&order.UploadedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if accrual.Valid {
			points, err := model.NewPoints(accrual.String)
			if err != nil {
				return nil, err
			}

			order.Accrual = &points
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}
