package model

import "time"

// OrderStatus описывает статус обработки расчёта по номеру заказа.
type OrderStatus string

// Points хранит значение баллов из PostgreSQL NUMERIC без преобразования в float64.
type Points string

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
