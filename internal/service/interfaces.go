package service

import (
	"context"

	"github.com/timaogurtzova/gophermart/internal/model"
)

// UserService описывает контракт сервиса регистрации и аутентификации.
type UserService interface {
	Register(ctx context.Context, login, password string) (model.User, error)
	Login(ctx context.Context, login, password string) (model.User, error)
}

// OrderService описывает контракт сервиса загрузки номеров заказов.
type OrderService interface {
	UploadOrder(ctx context.Context, userID int64, number string) error
}
