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

var (
	// ErrOrderAlreadyUploadedByUser возвращается, когда заказ уже загружен этим пользователем.
	ErrOrderAlreadyUploadedByUser = errors.New("order already uploaded by user")
	// ErrOrderAlreadyUploadedByAnotherUser возвращается, когда заказ уже загружен другим пользователем.
	ErrOrderAlreadyUploadedByAnotherUser = errors.New("order already uploaded by another user")
)

// OrderRepository описывает контракт хранилища заказов.
type OrderRepository interface {
	Upload(ctx context.Context, userID int64, number string) error
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
