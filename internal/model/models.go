package model

import (
	"database/sql/driver"
	"time"

	"github.com/shopspring/decimal"
)

// OrderStatus описывает статус обработки расчёта по номеру заказа.
type OrderStatus string

// Points хранит значение PostgreSQL NUMERIC как decimal без преобразования в float64.
//
// Это защищает баланс от ошибок округления: начисления и списания могут
// выполняться атомарными SQL-операциями с NUMERIC, а если расчёты появятся в Go,
// доменная модель уже не будет зависеть от float64.
type Points decimal.Decimal

// NewPoints создаёт значение баллов из строкового представления NUMERIC.
func NewPoints(value string) (Points, error) {
	points, err := decimal.NewFromString(value)
	if err != nil {
		return Points{}, err
	}

	return Points(points), nil
}

// String возвращает строковое представление баллов без незначащих нулей.
func (p Points) String() string {
	points := decimal.Decimal(p)
	if points.Sign() == 0 {
		return "0"
	}

	return points.String()
}

// MarshalJSON сериализует баллы как JSON-число.
func (p Points) MarshalJSON() ([]byte, error) {
	return []byte(p.String()), nil
}

// Scan читает значение NUMERIC из database/sql.
func (p *Points) Scan(value any) error {
	var points decimal.Decimal
	if err := points.Scan(value); err != nil {
		return err
	}

	*p = Points(points)
	return nil
}

// Value возвращает значение баллов для записи через database/sql.
func (p Points) Value() (driver.Value, error) {
	return p.String(), nil
}

const (
	// OrderStatusNew означает, что заказ загружен в систему, но не попал в обработку.
	OrderStatusNew OrderStatus = "NEW"
	// OrderStatusProcessing означает, что вознаграждение за заказ рассчитывается.
	OrderStatusProcessing OrderStatus = "PROCESSING"
	// OrderStatusInvalid означает, что система расчёта вознаграждений отказала в расчёте.
	OrderStatusInvalid OrderStatus = "INVALID"
	// OrderStatusProcessed означает, что данные по заказу проверены и расчёт получен.
	OrderStatusProcessed OrderStatus = "PROCESSED"
)

// User описывает зарегистрированного пользователя накопительной системы.
type User struct {
	ID           int64
	Login        string
	PasswordHash string
	CreatedAt    time.Time
}

// Order описывает номер заказа, загруженный зарегистрированным пользователем.
type Order struct {
	ID         int64
	UserID     int64
	Number     string
	Status     OrderStatus
	Accrual    *Points
	UploadedAt time.Time
	UpdatedAt  time.Time
}

// Balance описывает накопительный счёт зарегистрированного пользователя.
type Balance struct {
	UserID    int64
	Current   Points
	Withdrawn Points
	UpdatedAt time.Time
}

// Withdrawal описывает списание баллов с накопительного счёта пользователя.
type Withdrawal struct {
	ID          int64
	UserID      int64
	OrderNumber string
	Sum         Points
	ProcessedAt time.Time
}
